// Copyright 2017 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func (s *server) ServeIndex(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Lang string
	}{s.lang}
	if err := s.t.ExecuteTemplate(w, "index.html", &data); err != nil {
		log.Println(err)
	}
}

func (s *server) ServeNewAlbum(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Lang string
	}{s.lang}
	if err := s.t.ExecuteTemplate(w, "new.html", &data); err != nil {
		log.Println(err)
	}
}

func (s *server) ServeApiNewAlbum(w http.ResponseWriter, r *http.Request) {
	uid, err := s.SessionUid(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		log.Println(err)
		return
	}
	fmt.Println("uid:", uid)
	mr, err := r.MultipartReader()
	if err != nil {
		http.Error(w, s.tr("Bad request: error parsing form")+": ", http.StatusBadRequest)
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
				http.Error(w, s.tr("Internal server error"), http.StatusBadRequest)
				log.Println(err)
				return
			}
			fmt.Println(meta)
			continue
		}
		idx := strings.TrimPrefix(formName, "image:")
		if len(idx) == len(formName) {
			http.Error(w, s.tr("Bad request: error parsing form"), http.StatusBadRequest)
			log.Println("unexpected form name " + formName)
			return
		}

		filename := filepath.Join(tempDir, strconv.Itoa(len(files)))
		n, sha256, err := writeFileSha256(filename, p)
		if err != nil {
			http.Error(w, s.tr("Internal server error"), http.StatusInternalServerError)
			log.Println(err)
			return
		}
		inf := &uploadInfo{tmpFileName: filename, formName: formName, userFileName: p.FileName(), sha256: sha256}
		files = append(files, inf)
		m[idx] = inf
		fmt.Println(p.Header, n, p.FormName(), p.FileName(), sha256)
	}
	if meta.Name == "" {
		http.Error(w, s.tr("Bad request: name not specified"), http.StatusBadRequest)
		log.Println(err)
		return
	}
	for idx, descr := range meta.Descriptions {
		inf := m[idx]
		if m[idx] == nil {
			http.Error(w, s.tr("Bad request: error parsing form"), http.StatusBadRequest)
			log.Println(err)
			return
		}
		inf.description = descr
	}
	if errs := s.db.AddAlbum(meta.Name, files); errs != nil {
		for _, err := range errs {
			log.Println(err)
		}
	}
}

type uploadInfo struct {
	tmpFileName  string
	formName     string
	userFileName string
	description  string
	sha256       string
}

// AddAlarm (TODO) should also return error messages for end users
func (db *DB) AddAlbum(name string, files []*uploadInfo) (errs []error) {
	db.filesMu.Lock()
	defer db.filesMu.Unlock()
	var toRemove struct {
		files []string // new files to remove
	}
	defer func() {
		for _, fn := range toRemove.files {
			if err := os.Remove(fn); err != nil {
				errs = append(errs, err)
			}
		}
	}()
	for _, inf := range files {
		dirName := filepath.Join(db.imagesDir, inf.sha256[:3])
		destFilename := filepath.Join(dirName, inf.sha256[3:])
		if err := ensureDirExists(dirName, 0755); err != nil {
			errs = append(errs, err)
			continue
		}
		if err := os.Rename(inf.tmpFileName, destFilename); err != nil {
			errs = append(errs, err)
			continue
		}
		toRemove.files = append(toRemove.files, destFilename)
	}

	// TODO: if db transactions submitted than:
	toRemove.files = nil
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
