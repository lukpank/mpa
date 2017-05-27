// Copyright 2017 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

package main

import (
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
	tempDir, err := ioutil.TempDir(s.uploadDir, "tmp")
	if err != nil {
		http.Error(w, s.tr("Internal server error"), http.StatusInternalServerError)
		log.Println(err)
		return
	}
	defer os.RemoveAll(tempDir)
	type info struct {
		filename     string
		formName     string
		userFileName string
		description  string
	}
	var meta struct {
		Name         string
		Descriptions map[string]string
	}
	var files []*info
	m := make(map[string]*info)
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
		n, err := writeFile(filename, p)
		if err != nil {
			http.Error(w, s.tr("Internal server error"), http.StatusInternalServerError)
			log.Println(err)
			return
		}
		inf := &info{filename: filename, formName: formName, userFileName: p.FileName()}
		files = append(files, inf)
		m[idx] = inf
		fmt.Println(p.Header, n, p.FormName(), p.FileName())
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
}

func writeFile(filename string, r io.Reader) (int64, error) {
	f, err := os.Create(filename)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	n, err := io.Copy(f, r)
	if err != nil {
		return n, err
	}
	return n, f.Close()
}
