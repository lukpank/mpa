// Copyright 2017 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

package main

import (
	"database/sql"
	"image"
	"image/jpeg"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/nfnt/resize"
)

func (s *server) ServeImage(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r.URL.Path, "/image/")
	if err != nil {
		http.Error(w, s.tr("Page not found"), http.StatusNotFound)
		return
	}
	if filename, ok := s.ensurePreview(w, r, id, ".1"); ok {
		http.ServeFile(w, r, filename)
	}
}

func (s *server) ServeAPIImage(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r.URL.Path, "/api/image/")
	if err != nil {
		http.Error(w, s.tr("Page not found"), http.StatusNotFound)
		return
	}
	if _, ok := s.ensurePreview(w, r, id, ".1"); ok {
		w.WriteHeader(http.StatusOK)
	}
}

func (s *server) ServePreview(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r.URL.Path, "/preview/")
	if err != nil {
		http.Error(w, s.tr("Page not found"), http.StatusNotFound)
		return
	}
	if filename, ok := s.ensurePreview(w, r, id, ".2"); ok {
		http.ServeFile(w, r, filename)
	}
}

func (s *server) ensurePreview(w http.ResponseWriter, r *http.Request, id int64, ext string) (string, bool) {
	var sha256sum string
	err := s.db.db.QueryRow("SELECT sha256sum FROM images where iid=?", id).Scan(&sha256sum)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, s.tr("Page not found"), http.StatusNotFound)
			return "", false
		}
		http.Error(w, s.tr("Internal server error"), http.StatusInternalServerError)
		log.Println(err)
		return "", false
	}
	filename := filepath.Join(s.db.previewDir, sha256sum[:3], sha256sum[3:]+ext)
	if _, err := os.Stat(filename); err != nil {
		if !os.IsNotExist(err) {
			http.Error(w, s.tr("Internal server error"), http.StatusInternalServerError)
			log.Println(err)
			return "", false
		}
		result := make(chan error)
		s.preview <- previewRequest{id, sha256sum, result}
		if err = <-result; err != nil {
			http.Error(w, s.tr("Internal server error"), http.StatusInternalServerError)
			log.Println(err)
			return "", false
		}
	}
	return filename, true
}

type previewRequest struct {
	id        int64
	sha256sum string
	result    chan<- error
}

func (s *server) previewWorker() {
	for req := range s.preview {
		log.Printf("creating preview for image %d (%s)\n", req.id, req.sha256sum[:7])
		req.result <- s.createPreviews(req.sha256sum)
	}
}

func (s *server) createPreviews(sha256sum string) error {
	dirName := filepath.Join(s.db.previewDir, sha256sum[:3])
	filename := filepath.Join(dirName, sha256sum[3:])
	filename1 := filename + ".1"
	filename2 := filename + ".2"
	exists := 0
	if _, err := os.Stat(filename1); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		exists++
	}
	if _, err := os.Stat(filename2); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		exists++
	}
	if exists == 2 {
		return nil
	}
	img, err := s.readImage(sha256sum)
	if err != nil {
		return err
	}

	if err := ensureDirExists(dirName, 0755); err != nil {
		if !os.IsExist(err) {
			return err
		}
	}

	if err := s.createPreview(filename1, img, 1280); err != nil {
		return err
	}
	return s.createPreview(filename2, img, 320)
}

func (s *server) readImage(sha256sum string) (image.Image, error) {
	filename := filepath.Join(s.db.imagesDir, sha256sum[:3], sha256sum[3:])
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return img, f.Close()
}

func (s *server) createPreview(filename string, img image.Image, maxSize uint) error {
	var pw, ph uint
	size := img.Bounds().Size()
	if size.Y > size.X {
		pw = 0
		ph = maxSize
	} else {
		pw = maxSize
		ph = 0
	}
	img = resize.Resize(pw, ph, img, resize.Lanczos3)
	f, err := ioutil.TempFile(s.db.uploadDir, "tmp")
	if err != nil {
		return err
	}
	tmpFileName := f.Name()
	defer func() {
		if tmpFileName != "" {
			_ = os.Remove(tmpFileName)
		}
	}()
	defer f.Close()
	if err := jpeg.Encode(f, img, nil); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpFileName, filename); err != nil {
		return err
	}
	tmpFileName = ""
	return nil
}
