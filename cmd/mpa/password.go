// Copyright 2017 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

package main

import (
	"log"
	"net/http"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

type changePasswordData struct {
	Lang           string
	Redirect       string
	PasswordMsg    string
	NewPasswordMsg string
	Message        string
}

func (s *server) ServeChangePassword(w http.ResponseWriter, r *http.Request) {
	d := changePasswordData{Lang: s.lang}
	if r.Method != "POST" {
		path := r.URL.Path
		if r.URL.RawQuery != "" {
			path += "?" + r.URL.RawQuery
		}
		d.Redirect = path
		s.executeTemplate(w, "password.html", &d, http.StatusOK)
		return
	}
	if err := r.ParseForm(); err != nil {
		d.Message = s.tr("Error parsing form")
		s.executeTemplate(w, "password.html", &d, http.StatusBadRequest)
		return
	}
	password := r.PostForm.Get("password")
	newPassword := r.PostForm.Get("new_password")
	repeatPassword := r.PostForm.Get("repeat_password")
	d.Redirect = r.PostForm.Get("redirect")

	session, err := s.SessionData(r)
	if err != nil {
		log.Println(err)
		d.Message = s.tr("Session retrieving error")
		s.executeTemplate(w, "password.html", &d, http.StatusInternalServerError)
		return
	}
	msg, ok := checkPasswordStrength(newPassword, s.tr)
	if !ok {
		d.NewPasswordMsg = msg
	}
	if newPassword != repeatPassword {
		ok = false
		d.Message = s.tr("New and repeated passwords does not match")
	}
	_, _, err = s.db.AuthenticateUserByUid(session.Uid, []byte(password))
	if err != nil {
		if err == ErrAuth {
			d.PasswordMsg = s.tr("Incorrect password")
			s.executeTemplate(w, "password.html", &d, http.StatusUnauthorized)
		} else {
			log.Println(err)
			d.PasswordMsg = s.tr("Internal server error")
			s.executeTemplate(w, "password.html", &d, http.StatusInternalServerError)
		}
		return
	}
	if ok && newPassword == password {
		ok = false
		// this case is important when we force user to change password
		d.NewPasswordMsg = s.tr("New password and current password are identical")
	}
	if !ok {
		s.executeTemplate(w, "password.html", &d, http.StatusBadRequest)
		return
	}
	if err := s.db.ChangePassword(session.Uid, []byte(newPassword)); err != nil {
		log.Println(err)
		d.Message = s.tr("Internal server error")
		s.executeTemplate(w, "password.html", &d, http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func checkPasswordStrength(password string, tr func(string) string) (string, bool) {
	n := 0
	lower := false
	upper := false
	digit := false
	punct := false
	for _, r := range password {
		n++
		if unicode.IsLower(r) {
			lower = true
		}
		if unicode.IsUpper(r) {
			upper = true
		}
		if unicode.IsDigit(r) {
			digit = true
		}
		if unicode.IsPunct(r) || unicode.IsSymbol(r) {
			punct = true
		}
	}
	if n < 8 {
		return tr("Password must have at least 8 characters"), false
	}
	if !lower || !upper || !digit || !punct {
		return tr("Password must contain at least one lowercase letter, one uppercase letter, one digit and one other character"), false
	}
	return "", true
}

func (db *DB) ChangePassword(uid int64, password []byte) error {
	p, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = db.db.Exec("UPDATE users SET passwordhash=? WHERE uid=?", p, uid)
	return err
}
