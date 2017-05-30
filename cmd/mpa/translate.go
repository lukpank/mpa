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
	"person|Name":  "Name",
}

var plTranslation = translation{
	"lang-code": "pl",

	"Add description or delete": "Dodaj opis lub usuń",
	"Add user":                  "Dodaj użytkownika",
	"Admin account required":    "Wymagane konto administratora",
	"Admin":                     "Admin",
	"Bad request: error parsing form":              "Błędne zapytanie: błąd parsowania formularza",
	"Click to add description or delete the image": "Kliknij aby dodać opis lub usunąć obraz",
	"Delete":                                    "Usuń",
	"Description":                               "Opis",
	"Drop images or click here":                 "Upuść obrazy lub kliknij tutaj",
	"Email already registered":                  "Email już zarejestrowany",
	"Email":                                     "Email",
	"Error parsing form":                        "Błąd parsowania formularza",
	"Field":                                     "Pole",
	"Incorrect email address":                   "Niepoprawny adres email",
	"Incorrect login or password.":              "Niepoprawny login lub hasło.",
	"Incorrect password":                        "Niepoprawne hasło",
	"Internal server error":                     "Wewnętrzny błąd serwera",
	"Login already registered":                  "Login już zarejestrowany",
	"Login must have at least three characters": "Login musi mieć przynajmniej 3 litery",
	"Login must start with lowercase letter":    "Login musi zaczynać się on małej litery",
	"Login":                                     "Login",
	"Method not allowed":                        "Niedozwolona metoda",
	"Name may not be empty":                     "Imię nie może być puste",
	"Only lowercase letters and digits allowed": "Tylko małe liter y cyfry dozwolone",
	"Password": "Hasło",
	"Please specify album name and add at least one image": "Proszę określić nazwę albumu i dodać co najmniej jeden obraz",
	"Please use POST.":                                     "Proszę użyć POST.",
	"Session retrieving error":                             "Błąd pobierania sesji",
	"Surname may not be empty":                             "Nazwisko nie może być puste",
	"Surname":                                              "Nazwisko",
	"Update":                                               "Uaktualnij",
	"Value":                                                "Wartość",
	"Your password":                                        "Twoje hasło",
	"login|Submit":                                         "Zaloguj się",
	"no":                                                   "nie",
	"person|Name":                                          "Imię",
	"yes":                                                  "tak",
}
