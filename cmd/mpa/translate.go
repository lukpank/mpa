// Copyright 2016 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

package main

import "html/template"

var translations = map[string]translation{
	"en": enTranslation,
}

type translation map[string]string

func (t translation) translate(s string) string {
	if t[s] != "" {
		return t[s]
	}
	return s
}

func (t translation) htmlTranslate(s string) template.HTML {
	return template.HTML(t.translate(s))
}

var enTranslation = translation{
	"lang-code": "en",

	"login|Submit": "Submit",
}
