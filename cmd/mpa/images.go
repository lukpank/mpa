// Copyright 2017 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

package main

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

func (s *server) ServeImageOrig(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r.URL.Path, "/image/orig/")
	if err != nil {
		http.Error(w, s.tr("Page not found"), http.StatusNotFound)
		return
	}
	var sha256sum string
	err = s.db.db.QueryRow("SELECT sha256sum FROM images where iid=?", id).Scan(&sha256sum)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, s.tr("Page not found"), http.StatusNotFound)
			return
		}
		http.Error(w, s.tr("Internal server error"), http.StatusInternalServerError)
		log.Println(err)
		return
	}
	filename := filepath.Join(s.db.imagesDir, sha256sum[:3], sha256sum[3:])
	http.ServeFile(w, r, filename)
}

var ErrPrefixNotFound = errors.New("prefix not found")

func idFromPath(path, prefix string) (int64, error) {
	idStr := strings.TrimPrefix(path, prefix)
	if len(idStr) == len(path) {
		return 0, ErrPrefixNotFound
	}
	return strconv.ParseInt(idStr, 10, 64)
}
