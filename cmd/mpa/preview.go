// Copyright 2017 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

package main

import (
	"image"
	"image/jpeg"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/nfnt/resize"
)

func (s *server) ServePreview(w http.ResponseWriter, r *http.Request) {
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
