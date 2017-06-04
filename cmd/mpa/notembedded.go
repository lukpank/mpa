// Copyright 2017 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

// +build !embedded

package main

import (
	"html/template"
	"net/http"
)

// newTemplates return templates parsed from filesystem
func newTemplate(name string, funcMap template.FuncMap, filenames ...string) (*template.Template, error) {
	return template.New(name).Funcs(funcMap).ParseFiles(filenames...)
}

func newDir(path string) http.Dir {
	return http.Dir(path)
}
