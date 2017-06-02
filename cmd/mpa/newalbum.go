// Copyright 2017 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rwcarlsen/goexif/exif"
)

func (s *server) ServeNewAlbum(w http.ResponseWriter, r *http.Request) {
	s.executeTemplate(w, "newalbum.html", &struct{ Lang string }{s.lang}, http.StatusOK)
}

func (s *server) ServeApiNewAlbum(w http.ResponseWriter, r *http.Request) {
	session, err := s.SessionData(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		log.Println(err)
		return
	}
	mr, err := r.MultipartReader()
	if err != nil {
		http.Error(w, s.tr("Error parsing form"), http.StatusBadRequest)
		log.Println(err)
		return
	}
	tempDir, err := ioutil.TempDir(s.db.uploadDir, "tmp")
	if err != nil {
		http.Error(w, s.tr("Internal server error"), http.StatusInternalServerError)
		log.Println(err)
		return
	}
	defer os.RemoveAll(tempDir)
	var meta struct {
		Name         string
		Descriptions map[string]string
	}
	var files []*uploadInfo
	var imgCnt int
	var errs []imageError
	m := make(map[string]*uploadInfo)
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println(err)
			return
		}
		formName := p.FormName()
		if formName == "metadata" {
			if err := json.NewDecoder(p).Decode(&meta); err != nil {
				http.Error(w, s.tr("Error parsing metadata"), http.StatusBadRequest)
				log.Println(err)
				return
			}
			fmt.Println(meta)
			continue
		}
		idx := strings.TrimPrefix(formName, "image:")
		if len(idx) == len(formName) {
			http.Error(w, s.tr("Error parsing form"), http.StatusBadRequest)
			log.Println("unexpected form name " + formName)
			return
		}

		imgCnt++
		filename := filepath.Join(tempDir, strconv.Itoa(len(files)))
		n, sha256, err := writeFileSha256(filename, p)
		if err != nil {
			errs = append(errs, imageError{err, p.FileName(), s.tr("Internal server error")})
			m[idx] = &uploadInfo{}
			continue
		}
		isPort, err := isPortrait(filename)
		if err != nil {
			errs = append(errs, imageError{err, p.FileName(), s.tr("Could not determine image size")})
			m[idx] = &uploadInfo{}
			continue
		}
		var created time.Time
		t, err := exifDateTimeFromFile(filename)
		if err != nil {
			created = time.Now().UTC()
			errs = append(errs, imageError{err, p.FileName(), s.tr("Could not determine image time, current time assumed")})
		} else {
			created = t
		}
		inf := &uploadInfo{tmpFileName: filename, formName: formName, userFileName: p.FileName(), sha256: sha256, isPortrait: isPort, created: created}
		files = append(files, inf)
		m[idx] = inf
		fmt.Println(p.Header, n, p.FormName(), p.FileName(), sha256)
	}
	if meta.Name == "" {
		http.Error(w, s.tr("Album name not specified"), http.StatusBadRequest)
		log.Println("Bad request: Album name not specified")
		return
	}
	if len(files) == 0 {
		if len(errs) > 0 {
			http.Error(w, s.tr("No uploaded image was successfully processed"), http.StatusBadRequest)
			log.Println("Bad request: no uploaded image was successfully processed")
		} else {
			http.Error(w, s.tr("No images uploaded"), http.StatusBadRequest)
			log.Println("Bad request: no images uploaded")
		}
		return
	}
	files[0].isAlbumImage = true
	for idx, descr := range meta.Descriptions {
		inf := m[idx]
		if m[idx] == nil {
			http.Error(w, s.tr("Error parsing form"), http.StatusBadRequest)
			log.Println(err)
			return
		}
		inf.description = descr
	}
	n, albumID, errs2 := s.db.AddAlbum(session.Uid, meta.Name, files, s.tr)
	errs = append(errs, errs2...)
	if errs != nil {
		log.Println("album: ", n)
		for _, e := range errs {
			fmt.Printf("%s: %s: %s\n", e.FileName, e.Msg, e.err)
		}
	}
	if n == 0 {
		http.Error(w, s.tr("Internal server error"), http.StatusInternalServerError)
		return
	}
	msg := ""
	if n == imgCnt {
		msg = s.tr("All uploaded files added to the new album.")
	} else {
		msg = fmt.Sprintf(s.tr("%d out of %d uploaded files added to the new album."), n, imgCnt)
	}
	s.executeTemplate(w, "newalbumok.html", struct {
		Message  string
		Problems []imageError
		Href     string
	}{msg, errs, fmt.Sprintf("/album/%d", albumID)}, http.StatusOK)
}

type uploadInfo struct {
	tmpFileName  string
	formName     string
	userFileName string
	description  string
	sha256       string
	isPortrait   bool
	isAlbumImage bool
	created      time.Time
}

type imageError struct {
	err      error
	FileName string
	Msg      string
}

func isPortrait(filename string) (bool, error) {
	f, err := os.Open(filename)
	if err != nil {
		return false, err
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return false, err
	}
	return cfg.Height > cfg.Width, nil
}

func (db *DB) AddAlbum(uid int64, name string, files []*uploadInfo, tr func(string) string) (n int, albumID int64, errs []imageError) {
	db.filesMu.Lock()
	defer db.filesMu.Unlock()
	var toRemove struct {
		files []string // new files to remove
	}
	var fs []*uploadInfo
	defer func() {
		for _, fn := range toRemove.files {
			if err := os.Remove(fn); err != nil {
				errs = append(errs, imageError{err, fn, ""})
			}
		}
	}()
	for _, inf := range files {
		dirName := filepath.Join(db.imagesDir, inf.sha256[:3])
		destFilename := filepath.Join(dirName, inf.sha256[3:])
		if err := ensureDirExists(dirName, 0755); err != nil {
			errs = append(errs, imageError{err, inf.userFileName, tr("Internal server error")})
			continue
		}
		_, err := os.Stat(destFilename)
		if err == nil {
			// File alread exists
			fs = append(fs, inf)
			continue
		}
		if !os.IsNotExist(err) {
			errs = append(errs, imageError{err, inf.userFileName, tr("Internal server error")})
			continue
		}
		if err := os.Rename(inf.tmpFileName, destFilename); err != nil {
			errs = append(errs, imageError{err, inf.userFileName, tr("Internal server error")})
			continue
		}
		fs = append(fs, inf)
		toRemove.files = append(toRemove.files, destFilename)
	}

	tx, err := db.db.Begin()
	if err != nil {
		errs = append(errs, imageError{err, "", tr("Internal server error")})
		return
	}
	defer tx.Rollback()
	now := time.Now().UTC().Unix()
	r, err := tx.Exec("INSERT INTO albums (owner_id, created, modified, name) VALUES (?, ?, ?, ?)",
		uid, now, now, name)
	if err != nil {
		errs = append(errs, imageError{err, "", tr("Internal server error")})
		return
	}
	albumID, err = r.LastInsertId()
	if err != nil {
		errs = append(errs, imageError{err, "", tr("Internal server error")})
		return
	}
	var imageID int64
	isPortrait := false
	for _, inf := range fs {
		r, err := tx.Exec("INSERT INTO images (sha256sum, album_id, title, is_portrait, created, owner_file_name) VALUES (?, ?, ?, ?, ?, ?)",
			inf.sha256, albumID, inf.description, inf.isPortrait, inf.created, inf.userFileName)
		if err != nil {
			errs = append(errs, imageError{err, inf.userFileName, tr("Internal server error")})
			return
		}
		if inf.isAlbumImage {
			imageID, err = r.LastInsertId()
			isPortrait = inf.isPortrait
			if err != nil {
				errs = append(errs, imageError{err, inf.userFileName, tr("Internal server error")})
				return
			}
		}
	}
	_, err = tx.Exec("UPDATE albums SET image_id=?, is_portrait=? WHERE aid=?", imageID, isPortrait, albumID)
	if err != nil {
		errs = append(errs, imageError{err, "", tr("Internal server error")})
		return
	}
	if err := tx.Commit(); err != nil {
		errs = append(errs, imageError{err, "", tr("Internal server error")})
		return
	}
	// transaction commited so we do not want to remove new files
	toRemove.files = nil
	n = len(fs)
	return
}

func writeFileSha256(filename string, r io.Reader) (int64, string, error) {
	f, err := os.Create(filename)
	if err != nil {
		return 0, "", err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), r)
	if err != nil {
		return n, "", err
	}
	return n, hex.EncodeToString(h.Sum(nil)), f.Close()
}

func exifDateTimeFromFile(filename string) (time.Time, error) {
	var t time.Time
	f, err := os.Open(filename)
	if err != nil {
		return t, err
	}
	defer f.Close()
	x, err := exif.Decode(f)
	if err != nil {
		return t, err
	}
	t, err = x.DateTime()
	if err != nil {
		return t, err
	}
	return t, nil
}