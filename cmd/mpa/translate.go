// Copyright 2016 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

package main

import "html/template"

var translations = map[string]translation{
	"en": enTranslation,
	"pl": plTranslation,
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

var plTranslation = translation{
	"lang-code": "pl",

	"Add description or delete":                    "Dodaj opis lub usuń",
	"Bad request: error parsing form":              "Błędne zapytanie: błąd parsowania formularza",
	"Click to add description or delete the image": "Kliknij aby dodać opis lub usunąć obraz",
	"Delete":                       "Usuń",
	"Description":                  "Opis",
	"Drop images or click here":    "Upuść obrazy lub kliknij tutaj",
	"Incorrect login or password.": "Niepoprawny login lub hasło.",
	"Internal server error":        "Wewnętrzny błąd serwera",
	"Login":                        "Login",
	"Method not allowed":           "Niedozwolona metoda",
	"Password":                     "Hasło",
	"Please specify album name and add at least one image": "Proszę określić nazwę albumu i dodać co najmniej jeden obraz",
	"Please use POST.":                                     "Proszę użyć POST.",
	"Update":                                               "Uaktualnij",
	"login|Submit":                                         "Zaloguj się",
}
