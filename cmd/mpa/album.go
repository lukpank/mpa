// Copyright 2017 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
)

func (s *server) ServeAlbum(w http.ResponseWriter, r *http.Request) {
	albumID, err := idFromPath(r.URL.Path, "/album/")
	if err != nil {
		http.Error(w, s.tr("Page not found"), http.StatusNotFound)
		return
	}
	var name string
	err = s.db.db.QueryRow("SELECT name FROM albums WHERE aid=?", albumID).Scan(&name)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, s.tr("Page not found"), http.StatusNotFound)
			return
		}
		http.Error(w, s.tr("Internal server error"), http.StatusInternalServerError)
		log.Println(err)
		return
	}
	rows, err := s.db.db.Query("SELECT iid, is_portrait, title from images WHERE album_id=? ORDER BY created", albumID)
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
		URL    string
		Lang   string
		Images []img
	}{
		Title: name,
		URL:   pathQuery(r),
		Lang:  s.lang,
	}
	for rows.Next() {
		var id int64
		var portrait bool
		var title string
		if err := rows.Scan(&id, &portrait, &title); err != nil {
			log.Println(err)
			http.Error(w, s.tr("Internal server error"), http.StatusInternalServerError)
			return
		}
		class := "preview"
		if portrait {
			class = "preview portrait"
		}
		data.Images = append(data.Images, img{Src: fmt.Sprintf("/preview/%d", id), Class: class, Href: fmt.Sprintf("/view/%d#%d", albumID, id), Title: title})
	}
	if err := rows.Err(); err != nil {
		log.Println(err)
		http.Error(w, s.tr("Internal server error"), http.StatusInternalServerError)
		return
	}
	s.executeTemplate(w, "album.html", &data, http.StatusOK)
}

func pathQuery(r *http.Request) string {
	if q := r.URL.RawQuery; q != "" {
		return r.URL.Path + "?" + q
	}
	return r.URL.Path
}
