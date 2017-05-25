// Copyright 2017 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

package main

import (
	"flag"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

func main() {
	dbFileName := flag.String("f", "", "sqlite3 database file name")
	httpAddr := flag.String("http", ":8080", "HTTP listen address")
	insecureCookie := flag.Bool("insecure_cookie", false, "if client should send cookie over plain HTTP connection")
	flag.Parse()
	if *dbFileName == "" {
		log.Fatal("option -f is requiered")
	}
	db, err := OpenDB(*dbFileName)
	if err != nil {
		log.Fatal(err)
	}
	s, err := newServer(db, !*insecureCookie)
	if err != nil {
		log.Fatal("error: ", err)
	}
	http.HandleFunc("/", s.authenticate(s.ServeAlbum))
	http.HandleFunc("/preview/", s.authenticate(s.ServePreview))
	http.HandleFunc("/view/", s.authenticate(s.ServeView))
	http.HandleFunc("/login", s.serveLogin)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))
	log.Fatal(http.ListenAndServe(*httpAddr, &logger{http.DefaultServeMux}))
}

type server struct {
	db     *DB
	t      *template.Template
	s      *Sessions
	tr     func(string) string
	secure bool // if client should send cookie only on HTTPS encrypted connection
}

func newServer(db *DB, secure bool) (*server, error) {
	tr := translations["en"]
	m := template.FuncMap{"tr": tr.translate, "htmlTr": tr.htmlTranslate}
	t, err := template.New("html").Funcs(m).ParseFiles("templates/album.html", "templates/login.html", "templates/view.html")
	if err != nil {
		return nil, err
	}
	return &server{db: db, t: t, s: NewSessions(), tr: tr.translate, secure: secure}, nil
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
		Href  string
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
				class = "preview portrait"
			}
			data.Photos = append(data.Photos, img{Src: "/preview/" + name, Class: class, Href: "/view/album/" + name})
		}
	}
	if err := s.t.ExecuteTemplate(w, "album.html", &data); err != nil {
		log.Println(err)
	}
}

func (s *server) ServeView(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/view")
	if len(path) == len(r.URL.Path) || len(path) == 0 || path[0] != '/' {
		http.Error(w, "path prefix not found", http.StatusBadRequest)
		return
	}
	data := struct {
		Title  string
		Src    string
		Photos []string
		Index  int
	}{
		Title: path,
		Src:   "/static" + path,
	}
	infos, err := ioutil.ReadDir("static/album")
	if err != nil {
		log.Println(err)
		http.Error(w, "list dir error", http.StatusInternalServerError)
		return
	}
	for _, info := range infos {
		if name := info.Name(); strings.HasSuffix(name, ".jpg") {
			photo := "/static/album/" + name
			if data.Src == photo {
				data.Index = len(data.Photos)
			}
			data.Photos = append(data.Photos, photo)
		}
	}
	if err := s.t.ExecuteTemplate(w, "view.html", &data); err != nil {
		log.Println(err)
	}
}

func (s *server) error(w http.ResponseWriter, title, text string, code int) {
	w.Header().Set("Content-Type", "text/plain")
	http.Error(w, title+": "+text, code)
}

func (s *server) parseFormError(w http.ResponseWriter, err error) {
	s.error(w, s.tr("Bad request: error parsing form"), err.Error(), http.StatusBadRequest)
}

func (s *server) internalError(w http.ResponseWriter, err error) {
	s.error(w, s.tr("Internal server error"), err.Error(), http.StatusInternalServerError)
}
