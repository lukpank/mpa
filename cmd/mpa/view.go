// Copyright 2017 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

package main

import (
	"database/sql"
	"log"
	"net/http"
)

func (s *server) ServeView(w http.ResponseWriter, r *http.Request) {
	albumID, err := idFromPath(r.URL.Path, "/view/")
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
	rows, err := s.db.db.Query("SELECT iid from images WHERE album_id=? ORDER BY created", albumID)
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
	}
	data := struct {
		Title  string
		Lang   string
		Photos []int64
	}{
		Title: name,
		Lang:  s.lang,
	}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			log.Println(err)
			http.Error(w, s.tr("Internal server error"), http.StatusInternalServerError)
			return
		}
		data.Photos = append(data.Photos, id)
	}
	if err := rows.Err(); err != nil {
		log.Println(err)
		http.Error(w, s.tr("Internal server error"), http.StatusInternalServerError)
		return
	}
	s.executeTemplate(w, "view.html", &data, http.StatusOK)
}
