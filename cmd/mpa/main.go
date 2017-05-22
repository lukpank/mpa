// Copyright 2017 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

package main

import (
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

func main() {
	s, err := newServer()
	if err != nil {
		log.Fatal("error: ", err)
	}
	http.HandleFunc("/", s.ServeAlbum)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))
	log.Fatal(http.ListenAndServe(":8080", nil))
}

type server struct {
	t *template.Template
}

func newServer() (*server, error) {
	t, err := template.New("html").ParseFiles("templates/album.html")
	if err != nil {
		return nil, err
	}
	return &server{t}, nil
}

func (s *server) ServeAlbum(w http.ResponseWriter, r *http.Request) {
	infos, err := ioutil.ReadDir("static/album")
	if err != nil {
		log.Println(err)
		http.Error(w, "list dir error", http.StatusInternalServerError)
		return
	}
	type img struct {
		Src   string
		Class string
	}
	var data struct {
		Title  string
		Photos []img
	}
	data.Title = "My album"
	for _, info := range infos {
		if name := info.Name(); strings.HasSuffix(name, ".jpg") {
			class := "preview"
			portrait, err := isPortrait(filepath.Join("static/album", name))
			if err != nil {
				log.Println(err)
			}
			if portrait {
				class = "preview-portrait"
			}
			data.Photos = append(data.Photos, img{"/static/album/" + name, class})
		}
	}
	if err := s.t.ExecuteTemplate(w, "album.html", &data); err != nil {
		log.Println(err)
	}
}
