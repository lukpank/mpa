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

	"login|Submit":           "Submit",
	"person|Name":            "Name",
	"submit|Change password": "Change password",
	"title|Change password":  "Change password",
}

var plTranslation = translation{
	"lang-code": "pl",

	"%d of %d images deleted from the album have been successfully deleted.": "%d z %d obrazów usuniętych z albumu zostało poprawnie usuniętych.",
	"%d out of %d requsted image titles modified.":                           "Wprowadzono %d z %d żądanych zmian tytułów.",
	"%d out of %d uploaded files added to the album.":                        "%d z %d przesłanych plików dodano do albumu.",
	"%d out of %d uploaded files added to the new album.":                    "%d z %d przesłanych plików dodano do nowego albumu.",
	"Add title or delete":                                                    "Dodaj tytuł lub usuń",
	"Add user":                                                               "Dodaj użytkownika",
	"Admin account required":                                                 "Wymagane konto administratora",
	"Admin":                                                                  "Admin",
	"Album deleted":                                                          "Album usunęty",
	"Album does not exist or you are not its owner":                          "Album nie istnieje albo nie jesteś jego właścicielem",
	"Album name modified.":                                                   "Zmodyfikowano nazwę albumu",
	"Album name not specified":                                               "Nie określono nazwy albumu",
	"Album name":                                                             "Nazwa albumu",
	"Album updated":                                                          "Album uaktualniony",
	"Albums":                                                                 "Albumy",
	"All albums":                                                             "Wszystkie albumy",
	"All images deleted from the album have been successfully deleted.": "Wszystkie obrazy usunięte z albumu zostały pomyślnie usunięte.",
	"All requsted image titles modified.":                               "Wprowadzono wszystkie żądane zmiany tytułów.",
	"All uploaded files added to the album.":                            "Wszystkie przesłane pliki dodano do albumu.",
	"All uploaded files added to the new album.":                        "Wszystkie przesłane pliki dodano do nowego albumu.",
	"Authorization error":                                               "Błąd upoważnienia",
	"Bad request: error parsing form":                                   "Błędne zapytanie: błąd parsowania formularza",
	"Click to add title or delete the image":                            "Kliknij aby dodać tytuł lub usunąć obraz",
	"Close":                                                "Zamknij",
	"Connection error":                                     "Błąd połączenia",
	"Could not determine image size":                       "Nie udało się określić rozmiaru obrazu",
	"Could not determine image time, current time assumed": "Nie udało się określić czasu obrazu, przyjęto aktualny czas",
	"Current password":                                     "Aktualne hasło",
	"Delete":                                               "Usuń",
	"Down":                                                 "Dół",
	"Drop images or click here": "Upuść obrazy lub kliknij tutaj",
	"Edit album":                "Edytuj album",
	"Editing album":             "Edycja albumu",
	"Email already registered":  "Email już zarejestrowany",
	"Email":                     "Email",
	"Error during template execution": "Błąd podczas wykonania szablonu",
	"Error parsing form":              "Błąd parsowania formularza",
	"Error parsing metadata":          "Błąd parsowania metadanych",
	"Error":                           "Błąd",
	"Field":                           "Pole",
	"File":                            "Plik",
	"Incorrect email address":                         "Niepoprawny adres email",
	"Incorrect login or password.":                    "Niepoprawny login lub hasło.",
	"Incorrect password":                              "Niepoprawne hasło",
	"Internal server error":                           "Wewnętrzny błąd serwera",
	"Login already registered":                        "Login już zarejestrowany",
	"Login must have at least three characters":       "Login musi mieć przynajmniej 3 litery",
	"Login must start with lowercase letter":          "Login musi zaczynać się on małej litery",
	"Login required":                                  "Wymagane zalogowanie",
	"Login":                                           "Login",
	"Logout":                                          "Wyloguj",
	"Method not allowed":                              "Niedozwolona metoda",
	"My albums":                                       "Moje albumy",
	"Name may not be empty":                           "Imię nie może być puste",
	"New album created":                               "Utworzono nowy album",
	"New album":                                       "Nowy album",
	"New and repeated passwords does not match":       "Nowe i powtórzone hasła są różne",
	"New password and current password are identical": "Nowe hasło i aktualne hasło są identyczne",
	"New password":                                    "Nowe hasło",
	"New user":                                        "Nowy użytkownik",
	"No changes or empty album name":                  "Brak zmian lub pusta nazwa albumu",
	"No changes to the album requested":               "Nie zażądano żadnych zmian w albumie",
	"No images left in the album, album deleted.":     "Żaden obraz nie został w albumie, album usunięto.",
	"No images uploaded":                              "Nie przesłano żadnych obrazów",
	"No uploaded image was successfully processed":    "Żaden z przesłanych obrazów nie został pomyślnie przetworzony",
	"Only lowercase letters and digits allowed":       "Tylko małe liter y cyfry dozwolone",
	"Other users":                                     "Inni użytkownicy",
	"Password change required":                        "Wymagana zmiana hasła",
	"Password must have at least 8 characters":        "Hasło musi mieć przynajmniej 8 znaków",
	"Password": "Hasło",
	"Please specify album name and add at least one image": "Proszę określić nazwę albumu i dodać co najmniej jeden obraz",
	"Please use POST.":                                     "Proszę użyć POST.",
	"Problem":                                              "Problem",
	"Problems":                                             "Problemy",
	"Repeat password":                                      "Powtórzone hasło",
	"See the album":                                        "Zobacz ten album",
	"See the new album":                                    "Zobacz ten nowy album",
	"Session error":                                        "Błąd sesji",
	"Session retrieving error":                             "Błąd pobierania sesji",
	"Surname may not be empty":                             "Nazwisko nie może być puste",
	"Surname":                                              "Nazwisko",
	"Title":                                                "Tytuł",
	"To edit album you must be its owner": "Aby edytować album musisz być jego właścicielem",
	"Up":                     "Góra",
	"Update":                 "Uaktualnij",
	"Upload":                 "Prześlij",
	"Value":                  "Wartość",
	"Your password":          "Twoje hasło",
	"albums":                 "albumy",
	"login|Submit":           "Zaloguj się",
	"no":                     "nie",
	"person|Name":            "Imię",
	"submit|Change password": "Zmień hasło",
	"title|Change password":  "Zmiana hasła",
	"yes": "tak",

	"Password must contain at least one lowercase letter, one uppercase letter, one digit and one other character": "Hasło musi zawierać co najmniej jedną małą literę, jedną dużą literę, jedną cyfrę i jeden inny znak",
}
