// Copyright 2017 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

package main

import (
	"database/sql"
	"image"
	"image/jpeg"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/nfnt/resize"
)

func (s *server) ServePreviewOld(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/preview/")
	if len(path) == len(r.URL.Path) {
		http.Error(w, "path prefix not found", http.StatusBadRequest)
		return
	}
	previewPath := filepath.Join("preview", path)
	_, err := os.Stat(previewPath)
	if err == nil {
		http.ServeFile(w, r, previewPath)
		return
	}
	if !os.IsNotExist(err) {
		log.Println(err)
		http.Error(w, "stat error", http.StatusInternalServerError)
		return
	}
	albumPath := filepath.Join("static/album", path)
	f, err := os.Open(albumPath)
	if err != nil {
		log.Println(err)
		http.Error(w, "open error", http.StatusInternalServerError)
		return
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	f.Close()
	if err != nil {
		log.Println(err)
		http.Error(w, "image decode error", http.StatusInternalServerError)
		return
	}
	var pw, ph uint
	size := img.Bounds().Size()
	if size.Y > size.X {
		pw = 0
		ph = 320
	} else {
		pw = 320
		ph = 0
	}
	img = resize.Resize(pw, ph, img, resize.Lanczos3)
	f, err = os.Create(previewPath)
	if err != nil {
		log.Println(err)
		http.Error(w, "preview save error", http.StatusInternalServerError)
		return
	}
	defer f.Close()
	if err := jpeg.Encode(f, img, nil); err != nil {
		log.Println(err)
		http.Error(w, "preview save error", http.StatusInternalServerError)
		return
	}
	if err := f.Close(); err != nil {
		log.Println(err)
		http.Error(w, "preview save error", http.StatusInternalServerError)
		return
	}
	http.ServeFile(w, r, previewPath)
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

func (s *server) ServeImage(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r.URL.Path, "/image/")
	if err != nil {
		http.Error(w, s.tr("Page not found"), http.StatusNotFound)
		return
	}
	s.servePreview(w, r, id, ".1")
}

func (s *server) ServePreview(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r.URL.Path, "/preview/")
	if err != nil {
		// TODO: temporary for the static album to still work
		s.ServePreviewOld(w, r)
		return
		http.Error(w, s.tr("Page not found"), http.StatusNotFound)
		return
	}
	s.servePreview(w, r, id, ".2")
}

func (s *server) servePreview(w http.ResponseWriter, r *http.Request, id int64, ext string) {
	var sha256sum string
	err := s.db.db.QueryRow("SELECT sha256sum FROM images where iid=?", id).Scan(&sha256sum)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, s.tr("Page not found"), http.StatusNotFound)
			return
		}
		http.Error(w, s.tr("Internal server error"), http.StatusInternalServerError)
		log.Println(err)
		return
	}
	filename := filepath.Join(s.db.previewDir, sha256sum[:3], sha256sum[3:]+ext)
	if _, err := os.Stat(filename); err != nil {
		if !os.IsNotExist(err) {
			http.Error(w, s.tr("Internal server error"), http.StatusInternalServerError)
			log.Println(err)
			return
		}
		result := make(chan error)
		s.preview <- previewRequest{id, sha256sum, result}
		if err = <-result; err != nil {
			http.Error(w, s.tr("Internal server error"), http.StatusInternalServerError)
			log.Println(err)
			return
		}
	}
	http.ServeFile(w, r, filename)
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
	img, err := s.readImage(sha256sum)
	if err != nil {
		return err
	}
	dirName := filepath.Join(s.db.previewDir, sha256sum[:3])
	if err := ensureDirExists(dirName, 0755); err != nil {
		if !os.IsExist(err) {
			return err
		}
	}
	filename := filepath.Join(dirName, sha256sum[3:])
	if err := s.createPreview(filename+".1", img, 1280); err != nil {
		return err
	}
	return s.createPreview(filename+".2", img, 320)
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
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := jpeg.Encode(f, img, nil); err != nil {
		return err
	}
	return f.Close()
}
