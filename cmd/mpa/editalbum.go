// Copyright 2017 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

package main

import (
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func (s *server) ServeEditAlbum(w http.ResponseWriter, r *http.Request) {
	albumID, err := idFromPath(r.URL.Path, "/edit/album/")
	if err != nil {
		s.error(w, s.tr("Page not found"), "", http.StatusNotFound)
		return
	}
	session, err := s.SessionData(r)
	if err != nil {
		log.Println(err)
		s.error(w, s.tr("Authorization error"), "", http.StatusUnauthorized)
		return
	}
	var name string
	var ownerID int64
	err = s.db.db.QueryRow("SELECT name, owner_id FROM albums WHERE aid=?", albumID).Scan(&name, &ownerID)
	if err != nil {
		if err == sql.ErrNoRows {
			s.error(w, s.tr("Page not found"), "", http.StatusNotFound)
			return
		}
		log.Println(err)
		s.error(w, s.tr("Internal server error"), "", http.StatusInternalServerError)
		return
	}
	if ownerID != session.Uid {
		s.error(w, s.tr("Authorization error"), s.tr("To edit album you must be its owner"), http.StatusForbidden)
		return
	}

	rows, err := s.db.db.Query("SELECT iid, is_portrait, title from images WHERE album_id=? ORDER BY created", albumID)
	if err != nil {
		log.Println(err)
		s.error(w, s.tr("Internal server error"), "", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type img struct {
		Src   string `json:"-"`
		Class string `json:"-"`
		Id    int64  `json:"id"`
		Title string `json:"title"`
	}
	data := struct {
		Title     string
		URL       string
		SubmitURL string
		Lang      string
		Images    []img
	}{
		Title:     name,
		URL:       pathQuery(r),
		SubmitURL: fmt.Sprintf("/api/edit/album/%d", albumID),
		Lang:      s.lang,
	}
	for rows.Next() {
		var id int64
		var portrait bool
		var title string
		if err := rows.Scan(&id, &portrait, &title); err != nil {
			log.Println(err)
			s.error(w, s.tr("Internal server error"), "", http.StatusInternalServerError)
			return
		}
		class := "preview"
		if portrait {
			class = "preview portrait"
		}
		data.Images = append(data.Images, img{Src: fmt.Sprintf("/preview/%d", id), Class: class, Id: id, Title: title})
	}
	if err := rows.Err(); err != nil {
		log.Println(err)
		s.error(w, s.tr("Internal server error"), "", http.StatusInternalServerError)
		return
	}
	s.executeTemplate(w, "editalbum.html", &data, http.StatusOK)
}

func (s *server) ServeAPIEditAlbum(w http.ResponseWriter, r *http.Request) {
	albumID, err := idFromPath(r.URL.Path, "/api/edit/album/")
	if err != nil {
		http.Error(w, s.tr("Page not found"), http.StatusNotFound)
		return
	}
	session, err := s.SessionData(r)
	if err != nil {
		log.Println(err)
		// Forbidden used as API calls expect modal login served on Unauthorized.
		// Actually it is probably internal server error.
		http.Error(w, s.tr("Authorization error"), http.StatusForbidden)
		return
	}
	var name string
	var ownerID int64
	err = s.db.db.QueryRow("SELECT name, owner_id FROM albums WHERE aid=?", albumID).Scan(&name, &ownerID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, s.tr("Page not found"), http.StatusNotFound)
			return
		}
		log.Println(err)
		http.Error(w, s.tr("Internal server error"), http.StatusInternalServerError)
		return
	}
	if ownerID != session.Uid {
		http.Error(w, s.tr("Authorization error")+": "+s.tr("To edit album you must be its owner"), http.StatusForbidden)
		return
	}

	tempDir, err := ioutil.TempDir(s.db.uploadDir, "tmp")
	if err != nil {
		log.Println(err)
		http.Error(w, s.tr("Internal server error"), http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tempDir)
	d, ok := s.upload(w, r, tempDir)
	if !ok {
		return
	}
	if d.meta.Name == "" {
		log.Println("Bad request: Album name not specified")
		http.Error(w, s.tr("Album name not specified"), http.StatusBadRequest)
		return
	}

	if d.meta.Name == name && d.imgCnt == 0 && len(d.meta.Edit.Deleted) == 0 && len(d.meta.Edit.Titles) == 0 {
		log.Println("Bad request: No changes to the album requested")
		http.Error(w, s.tr("No changes to the album requested"), http.StatusBadRequest)
		return
	}
	for idx, title := range d.meta.Titles {
		inf := d.m[idx]
		if d.m[idx] == nil {
			log.Println("Error parsing form: unexpected index")
			http.Error(w, s.tr("Error parsing form"), http.StatusBadRequest)
			return
		}
		inf.title = title
	}
	rs := s.db.EditAlbum(session.Uid, albumID, d.meta.Name, d.meta.Edit.Deleted, d.meta.Edit.Titles, d.files, s.tr)
	n := len(rs.Jobs)
	d.errs = append(d.errs, rs.Errs...)
	if d.errs != nil {
		log.Println("album:", albumID, "new:", n)
		for _, e := range d.errs {
			fmt.Printf("%s: %s: %s\n", e.FileName, e.Msg, e.err)
		}
	}
	if rs.Status != http.StatusOK {
		http.Error(w, d.errs[len(d.errs)-1].Msg, rs.Status)
		return
	}
	if n > 0 {
		go s.preparePreviews(rs.Jobs)
	}
	data := struct {
		Title    string
		Messages []string
		Problems []imageError
		Href     string
	}{Problems: d.errs, Href: fmt.Sprintf("/album/%d", albumID)}

	if rs.Deleted {
		data.Title = s.tr("Album deleted")
		data.Messages = append(data.Messages, s.tr("No images left in the album, album deleted."))
	} else {
		data.Title = s.tr("Album updated")
		if d.meta.Name != name {
			data.Messages = append(data.Messages, s.tr("Album name modified."))
		}
		if len(d.meta.Edit.Titles) > 0 {
			if rs.TitlesCnt == len(d.meta.Edit.Titles) {
				data.Messages = append(data.Messages, s.tr("All requsted image titles modified."))
			} else {
				data.Messages = append(data.Messages, fmt.Sprintf(s.tr("%d out of %d requsted image titles modified."), n, d.imgCnt))
			}
		}
		if d.imgCnt > 0 {
			if n == d.imgCnt {
				data.Messages = append(data.Messages, s.tr("All uploaded files added to the album."))
			} else {
				data.Messages = append(data.Messages, fmt.Sprintf(s.tr("%d out of %d uploaded files added to the album."), n, d.imgCnt))
			}
		}
		if len(d.meta.Edit.Deleted) > 0 {
			if rs.DeletedCnt == len(d.meta.Edit.Deleted) {
				data.Messages = append(data.Messages, s.tr("All images deleted from the album have been successfully deleted."))
			} else {
				data.Messages = append(data.Messages, fmt.Sprintf(s.tr("%d of %d images deleted from the album have been successfully deleted."), rs.DeletedCnt, len(d.meta.Edit.Deleted)))
			}
		}
	}
	s.executeTemplate(w, "editalbumok.html", &data, http.StatusOK)
}

type EditAlbumResult struct {
	Status     int
	Deleted    bool
	DeletedCnt int
	TitlesCnt  int
	Jobs       []previewJob
	Errs       []imageError
}

func (db *DB) EditAlbum(uid int64, albumID int64, name string, deleted []int64, titles map[string]string, files []*uploadInfo, tr func(string) string) (rs EditAlbumResult) {
	rs.Status = http.StatusInternalServerError
	db.filesMu.Lock()
	defer db.filesMu.Unlock()
	var toRemove struct {
		files []string // new files (on failure) or deleted files (on success) to remove
	}
	var fs []*uploadInfo
	defer func() {
		for _, fn := range toRemove.files {
			if err := os.Remove(fn); err != nil {
				rs.Errs = append(rs.Errs, imageError{err, fn, ""})
			}
		}
	}()

	tx, err := db.db.Begin()
	if err != nil {
		rs.Errs = append(rs.Errs, imageError{err, "", tr("Internal server error")})
		return
	}
	defer tx.Rollback()
	now := time.Now().UTC().Unix()
	_, err = tx.Exec("UPDATE albums SET name=?, modified=? WHERE aid=? AND owner_id=?", name, now, albumID, uid)
	if err != nil {
		rs.Status = http.StatusForbidden
		rs.Errs = append(rs.Errs, imageError{err, "", tr("Album does not exist or you are not its owner")})
		return
	}

	checkDeleteSHA256 := make([]string, 0, len(deleted))
	for _, imageID := range deleted {
		var sha256sum string
		if err := tx.QueryRow("SELECT sha256sum FROM images WHERE iid=? AND album_id=?", imageID, albumID).Scan(&sha256sum); err != nil {
			rs.Errs = append(rs.Errs, imageError{err, fmt.Sprintf("image=%d", imageID), ""})
			continue

		}
		r, err := tx.Exec("DELETE FROM images WHERE iid=? AND album_id=?", imageID, albumID)
		if err != nil {
			rs.Errs = append(rs.Errs, imageError{err, fmt.Sprintf("image=%d", imageID), tr("Internal server error")})
			return
		}
		cnt, err := r.RowsAffected()
		if err != nil {
			rs.Errs = append(rs.Errs, imageError{err, fmt.Sprintf("image=%d", imageID), tr("Internal server error")})
			return
		}
		if cnt == 0 {
			continue
		}
		rs.DeletedCnt++
		checkDeleteSHA256 = append(checkDeleteSHA256, sha256sum)
	}
	if rs.DeletedCnt != len(deleted) {
		rs.Errs = append(rs.Errs, imageError{errors.New("Not found in DB"), tr("%d of %d deleted"), tr("Not found in this album")})
	}

	for idStr, title := range titles {
		imageID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			rs.Status = http.StatusBadRequest
			rs.Errs = append(rs.Errs, imageError{err, "", tr("Error parsing image ID")})
			continue
		}
		_, err = tx.Exec("UPDATE images SET title=? WHERE iid=? AND album_id=?", title, imageID, albumID)
		if err != nil {
			rs.Errs = append(rs.Errs, imageError{err, fmt.Sprintf("image=%d", imageID), tr("Internal server error")})
			return
		}
		rs.TitlesCnt++
	}

	for _, inf := range files {
		dirName := filepath.Join(db.imagesDir, inf.sha256[:3])
		destFilename := filepath.Join(dirName, inf.sha256[3:])
		if err := ensureDirExists(dirName, 0755); err != nil {
			rs.Errs = append(rs.Errs, imageError{err, inf.userFileName, tr("Internal server error")})
			continue
		}
		_, err := os.Stat(destFilename)
		if err == nil {
			// File already exists
			fs = append(fs, inf)
			continue
		}
		if !os.IsNotExist(err) {
			rs.Errs = append(rs.Errs, imageError{err, inf.userFileName, tr("Internal server error")})
			continue
		}
		if err := os.Rename(inf.tmpFileName, destFilename); err != nil {
			rs.Errs = append(rs.Errs, imageError{err, inf.userFileName, tr("Internal server error")})
			continue
		}
		fs = append(fs, inf)
		toRemove.files = append(toRemove.files, destFilename)
	}

	var albumImageID int64 = -1
	albumIsPortrait := false
	jobs := make([]previewJob, 0, len(fs))
	for _, inf := range fs {
		r, err := tx.Exec("INSERT INTO images (sha256sum, album_id, title, is_portrait, created, owner_file_name) VALUES (?, ?, ?, ?, ?, ?)",
			inf.sha256, albumID, inf.title, inf.isPortrait, inf.created, inf.userFileName)
		if err != nil {
			rs.Errs = append(rs.Errs, imageError{err, inf.userFileName, tr("Internal server error")})
			return
		}
		id, err := r.LastInsertId()
		if err != nil {
			rs.Errs = append(rs.Errs, imageError{err, inf.userFileName, tr("Internal server error")})
			return
		}
		jobs = append(jobs, previewJob{id, inf.sha256})
		if inf.isAlbumImage {
			albumImageID = id
			albumIsPortrait = inf.isPortrait
		}
	}
	var toRemoveOnSuccess []string
	previewExt := []string{".1", ".2"}
	for _, sha256sum := range checkDeleteSHA256 {
		var exists bool
		err := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM images WHERE sha256sum=? LIMIT 1)", sha256sum).Scan(&exists)
		if err != nil {
			rs.Errs = append(rs.Errs, imageError{err, "", tr("Internal server error")})
			return
		}
		if exists {
			continue
		}
		toRemoveOnSuccess = append(toRemoveOnSuccess, filepath.Join(db.imagesDir, sha256sum[:3], sha256sum[3:]))
		filename := filepath.Join(db.previewDir, sha256sum[:3], sha256sum[3:])
		for _, ext := range previewExt {
			delFileName := filename + ext
			_, err := os.Stat(delFileName)
			if err != nil && os.IsNotExist(err) {
				continue
			}
			toRemoveOnSuccess = append(toRemoveOnSuccess, delFileName)
		}
	}

	var imageID int64
	var isPortrait bool
	err = tx.QueryRow("SELECT iid, is_portrait from images WHERE album_id=? ORDER BY created LIMIT 1", albumID).Scan(&imageID, &isPortrait)
	if err != nil {
		if err != sql.ErrNoRows {
			rs.Errs = append(rs.Errs, imageError{err, "", tr("Internal server error")})
			return
		}
		_, err = tx.Exec("DELETE FROM albums WHERE aid=?", albumID)
		if err != nil {
			rs.Errs = append(rs.Errs, imageError{err, "", tr("Internal server error")})
			return
		}
		if err := tx.Commit(); err != nil {
			rs.Errs = append(rs.Errs, imageError{err, "", tr("Internal server error")})
			return
		}
		// transaction commited (album deleted) so we do remove other set of files
		toRemove.files = toRemoveOnSuccess
		rs.Status = http.StatusOK
		rs.Deleted = true
		return
	}
	if albumImageID == -1 {
		albumImageID = imageID
		albumIsPortrait = isPortrait
	}
	_, err = tx.Exec("UPDATE albums SET image_id=?, is_portrait=? WHERE aid=?", albumImageID, albumIsPortrait, albumID)
	if err != nil {
		rs.Errs = append(rs.Errs, imageError{err, "", tr("Internal server error")})
		return
	}
	if err := tx.Commit(); err != nil {
		rs.Errs = append(rs.Errs, imageError{err, "", tr("Internal server error")})
		return
	}
	// transaction commited so we do remove other set of files
	toRemove.files = toRemoveOnSuccess
	rs.Jobs = jobs
	rs.Status = http.StatusOK
	return
}
