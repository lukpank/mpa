// Copyright 2017 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

package main

import (
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
	data := struct {
		Lang string
	}{s.lang}
	if err := s.t.ExecuteTemplate(w, "new.html", &data); err != nil {
		log.Println(err)
	}
}
