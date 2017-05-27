// Copyright 2017 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
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
	if r.Method == "POST" {
		s.serveNewAlbumUpload(w, r)
		return
	}

	data := struct {
		Lang string
	}{s.lang}
	if err := s.t.ExecuteTemplate(w, "new.html", &data); err != nil {
		log.Println(err)
	}
}

// TODO: use /api/new with special authenticate which does not send login page
func (s *server) serveNewAlbumUpload(w http.ResponseWriter, r *http.Request) {
	mr, err := r.MultipartReader()
	if err != nil {
		http.Error(w, s.tr("Bad request: error parsing form")+": ", http.StatusBadRequest)
		log.Println(err)
		return
	}
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Println(err)
			return
		}
		n, err := io.Copy(ioutil.Discard, p)
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Println(p.Header, n)
	}

}
