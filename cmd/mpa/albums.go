// Copyright 2017 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
)

func (s *server) ServeAlbums(w http.ResponseWriter, r *http.Request) {
	login := strings.TrimPrefix(r.URL.Path, "/albums/")
	title := "All albums"
	if len(login) == len(r.URL.Path) {
		http.Error(w, s.tr("Page not found"), http.StatusNotFound)
		return
	}
	if r.URL.Path == "/albums" {
		login = ""
	}
	var rows *sql.Rows
	var err error
	if login != "" {
		rows, err = s.db.db.Query("SELECT aid, image_id, is_portrait, name from albums WHERE owner_id=(SELECT uid FROM users WHERE login=?)", login)
	} else {
		rows, err = s.db.db.Query("SELECT aid, image_id, is_portrait, name from albums")
		title = "Albums of user " + login
	}
	if err != nil {
		http.Error(w, s.tr("Internal server error"), http.StatusInternalServerError)
		log.Println(err)
		return
	}
	defer rows.Close()

	type img struct {
		Src   string
		Class string
		Href  string
		Title string
	}
	data := struct {
		Title  string
		Lang   string
		Photos []img
	}{
		Title: title,
		Lang:  s.lang,
	}
	for rows.Next() {
		var albumID int64
		var imageID int64
		var portrait bool
		var name string
		if err := rows.Scan(&albumID, &imageID, &portrait, &name); err != nil {
			http.Error(w, s.tr("Internal server error"), http.StatusInternalServerError)
			log.Println(err)
			return
		}
		class := "preview"
		if portrait {
			class = "preview portrait"
		}
		data.Photos = append(data.Photos, img{Src: fmt.Sprintf("/preview/%d", imageID), Class: class, Href: fmt.Sprintf("/album/%d", albumID), Title: name})
	}
	if err := rows.Err(); err != nil {
		http.Error(w, s.tr("Internal server error"), http.StatusInternalServerError)
		log.Println(err)
		return
	}
	if login != "" && len(data.Photos) == 0 {
		var uid int64
		err = s.db.db.QueryRow("SELECT uid FROM users WHERE login=?", login).Scan(&uid)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, s.tr("Page not found"), http.StatusNotFound)
				return
			}
			http.Error(w, s.tr("Internal server error"), http.StatusInternalServerError)
			log.Println(err)
			return
		}
	}
	s.executeTemplate(w, "album.html", &data, http.StatusOK)
}
