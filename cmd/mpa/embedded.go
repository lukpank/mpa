// Copyright 2017 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

// +build embedded

package main

import (
	"html/template"
	"net/http"
	"path/filepath"
)

// newTemplates return templates parsed from static assets
func newTemplate(name string, funcMap template.FuncMap, filenames ...string) (*template.Template, error) {
	t := template.New(name).Funcs(funcMap)
	for _, fn := range filenames {
		var err error
		name := filepath.Base(fn)
		s, err := FSString(false, "/"+fn)
		if err != nil {
			return nil, err
		}
		if _, err = t.New(name).Parse(s); err != nil {
			return nil, err
		}
	}
	return t, nil
}

func newDir(path string) http.FileSystem {
	return Dir(false, "/"+path)
}
