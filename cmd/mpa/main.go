// Copyright 2017 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
)

var Version = "mpa-0.1"

func main() {
	dbFileName := flag.String("f", "", "sqlite3 database file name")
	dbInit := flag.String("init", "", "initialize the database file (argument is options such as lang=en or lang=pl)")
	httpAddr := flag.String("http", ":8080", "HTTP listen address")
	insecureCookie := flag.Bool("insecure_cookie", false, "if client should send cookie over plain HTTP connection")
	version := flag.Bool("v", false, "show program version")
	flag.Parse()
	if *version {
		fmt.Println(Version)
		return
	}
	if *dbFileName == "" {
		log.Fatal("option -f is requiered")
	}
	db, err := OpenDB(*dbFileName)
	if err != nil {
		log.Fatal(err)
	}
	filesDir := *dbFileName + ".mpa"
	if *dbInit != "" {
		lang, err := parseOptions(*dbInit)
		if err != nil {
			log.Fatal("failed to initialize database: ", err)
		}
		if err = db.Init(lang); err != nil {
			log.Fatal("failed to initialize database: ", err)
		}
		if err := ensureDirExists(filepath.Join(filesDir), 0700); err != nil {
			log.Fatal("error: ", err)
		}
		return
	}
	s, err := newServer(db, !*insecureCookie, filesDir)
	if err != nil {
		log.Fatal("error: ", err)
	}
	http.HandleFunc("/", s.authenticate(s.ServeIndex))
	http.HandleFunc("/new/album", s.authenticate(s.ServeNewAlbum))
	http.HandleFunc("/api/new/album", s.authenticate(s.ServeAPINewAlbum))
	http.HandleFunc("/edit/album/", s.authenticate(s.ServeEditAlbum))
	http.HandleFunc("/api/edit/album/", s.authenticate(s.ServeAPIEditAlbum))
	http.HandleFunc("/albums/", s.authenticate(s.ServeAlbums))
	http.HandleFunc("/album/", s.authenticate(s.ServeAlbum))
	http.HandleFunc("/preview/", s.authenticate(s.ServePreview))
	http.HandleFunc("/view/", s.authenticate(s.ServeView))
	http.HandleFunc("/image/", s.authenticate(s.ServeImage))
	http.HandleFunc("/api/image/", s.authenticate(s.ServeImage))
	http.HandleFunc("/image/orig/", s.authenticate(s.ServeImageOrig))
	http.HandleFunc("/login", s.ServeLogin)
	http.HandleFunc("/api/login", s.ServeAPILogin)
	http.HandleFunc("/logout/", s.ServeLogout)
	http.HandleFunc("/password", s.authenticate(s.ServeChangePassword))
	http.HandleFunc("/new/user", s.authenticate(s.authorizeAsAdmin(s.ServeNewUser)))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(newDir("static/"))))
	http.HandleFunc("/favicon.ico", ServeFavicon)
	log.Fatal(http.ListenAndServe(*httpAddr, &logger{http.DefaultServeMux}))
}

func parseOptions(options string) (lang string, err error) {
	mask := 0
	for _, s := range strings.Split(options, ",") {
		switch {
		case strings.HasPrefix(s, "lang="):
			lang = strings.TrimPrefix(s, "lang=")
			if lang != "en" && lang != "pl" {
				return "", fmt.Errorf("unsupported language: %s", lang)
			}
			mask |= 1
		default:
			return "", fmt.Errorf("unsupported option: %s", s)
		}
	}
	if mask&1 == 0 {
		return "", errors.New("please specify option lang=en or lang=pl")
	}
	return
}

type server struct {
	db      *DB
	t       *template.Template
	s       *Sessions
	tr      func(string) string
	lang    string
	secure  bool // if client should send cookie only on HTTPS encrypted connection
	preview chan previewRequest
}

func newServer(db *DB, secure bool, filesDir string) (*server, error) {
	lang, err := db.GetMPAOptions()
	if err != nil {
		return nil, err
	}
	tr := translations[lang]
	if tr == nil {
		log.Printf("unsupported translation language %s, using en (i.e., English) instead", lang)
		tr = translations["en"]
	}
	m := template.FuncMap{"tr": tr.translate, "htmlTr": tr.htmlTranslate}
	t, err := newTemplate("html", m,
		"templates/album.html",
		"templates/editalbum.html",
		"templates/editalbumok.html",
		"templates/error.html",
		"templates/index.html",
		"templates/login.html",
		"templates/loginapi.html",
		"templates/newalbum.html",
		"templates/newalbumok.html",
		"templates/newuser.html",
		"templates/newuserok.html",
		"templates/password.html",
		"templates/view.html")
	if err != nil {
		return nil, err
	}
	if err := db.EnsureDirs(); err != nil {
		return nil, err
	}
	c := make(chan previewRequest)
	s := &server{db: db, t: t, s: NewSessions(), tr: tr.translate, lang: lang, secure: secure, preview: c}
	go s.previewMaster(runtime.NumCPU())
	return s, nil
}

func (s *server) executeTemplate(w http.ResponseWriter, name string, data interface{}, code int) {
	var b bytes.Buffer
	if err := s.t.ExecuteTemplate(&b, name, data); err != nil {
		log.Println(err)
		s.templateExecutionError(w)
		return
	}
	w.WriteHeader(code)
	if _, err := b.WriteTo(w); err != nil {
		log.Println(err)
	}
}

// templateExecutionError is used internally by executeTemplate to avoid infinite recursion
func (s *server) templateExecutionError(w http.ResponseWriter) {
	data := struct {
		Lang, Title, Text string
	}{s.lang, s.tr("Internal server error"), s.tr("Error during template execution")}
	var b bytes.Buffer
	if err := s.t.ExecuteTemplate(&b, "error.html", &data); err != nil {
		log.Println(err)
		w.Header().Set("Content-Type", "text/plain")
		http.Error(w, data.Title+": "+data.Text, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusInternalServerError)
	if _, err := b.WriteTo(w); err != nil {
		log.Println(err)
	}
}

func ServeFavicon(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/static/favicon.png", http.StatusSeeOther)
}

func (s *server) error(w http.ResponseWriter, title, text string, code int) {
	s.executeTemplate(w, "error.html", &struct {
		Lang, Title, Text string
	}{s.lang, title, text}, code)
}

func (s *server) parseFormError(w http.ResponseWriter, err error) {
	log.Println(err)
	s.error(w, s.tr("Bad request"), s.tr("Error parsing form"), http.StatusBadRequest)
}

func (s *server) internalError(w http.ResponseWriter, err error, msg string) {
	log.Println(err)
	s.error(w, s.tr("Internal server error"), msg, http.StatusInternalServerError)
}
