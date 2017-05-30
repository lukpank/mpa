// Copyright 2017 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

package main

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"
	"net/mail"
	"strings"

	"github.com/mattn/go-sqlite3"
)

func (s *server) ServeNewUser(w http.ResponseWriter, r *http.Request) {
	data := loginData{Lang: s.lang}
	code := http.StatusOK
	if r.Method == "POST" {
		code = s.createNewUser(w, r, &data)
		if code == http.StatusOK {
			return
		}
	}
	s.executeTemplate(w, "newuser.html", &data, code)
}

type loginData struct {
	Lang        string
	Login       string
	LoginMsg    string
	Name        string
	NameMsg     string
	Surname     string
	SurnameMsg  string
	Email       string
	EmailMsg    string
	Admin       bool
	Message     string
	TmpPassword string
}

func (s *server) createNewUser(w http.ResponseWriter, r *http.Request, d *loginData) int {
	if err := r.ParseForm(); err != nil {
		d.Message = s.tr("Error parsing form")
		return http.StatusBadRequest
	}
	d.Login = r.Form.Get("login")
	d.Admin = r.Form.Get("admin") == "on"
	d.Name = r.Form.Get("name")
	d.Surname = r.Form.Get("surname")
	d.Email = r.Form.Get("email")
	var ok bool
	d.LoginMsg, ok = checkLoginName(d.Login, s.tr)
	if d.Name == "" {
		d.NameMsg = s.tr("Name may not be empty")
		ok = false
	}
	if d.Surname == "" {
		d.SurnameMsg = s.tr("Surname may not be empty")
		ok = false
	}
	_, err := mail.ParseAddress(d.Email)
	if err != nil {
		d.EmailMsg = s.tr("Incorrect email address")
		ok = false
	}
	session, err := s.SessionData(r)
	if err != nil {
		log.Println(err)
		d.Message = s.tr("Session retrieving error")
		return http.StatusInternalServerError
	}
	_, admin, err := s.db.AuthenticateUserByUid(session.Uid, []byte(r.Form.Get("password")))
	if err != nil {
		if err == ErrAuth {
			d.Message = s.tr("Incorrect password")
			return http.StatusUnauthorized
		} else {
			log.Println(err)
			d.Message = s.tr("Internal server error")
			return http.StatusInternalServerError
		}
	}
	if !admin {
		d.Message = s.tr("Admin account required")
		return http.StatusUnauthorized
	}
	if !ok {
		return http.StatusBadRequest
	}
	randomPass, err := randomPassword()
	if err != nil {
		log.Println(err)
		d.Message = s.tr("Internal server error")
		return http.StatusInternalServerError
	}
	adminLevel := 0
	if d.Admin {
		adminLevel = 1
	}
	if err := s.db.AddUser(s.db.db, d.Login, d.Name, d.Surname, d.Email, adminLevel, randomPass); err != nil {
		if err, ok := err.(sqlite3.Error); ok {
			if err.Code == sqlite3.ErrConstraint && err.ExtendedCode == sqlite3.ErrConstraintUnique {
				e := err.Error()
				if strings.Contains(e, "users.login") {
					d.LoginMsg = s.tr("Login already registered")
					return http.StatusConflict
				}
				if strings.Contains(e, "users.email") {
					d.EmailMsg = s.tr("Email already registered")
					return http.StatusConflict
				}
			}
		}
		log.Println(err)
		d.Message = s.tr("Internal server error")
		return http.StatusInternalServerError
	}
	d.TmpPassword = string(randomPass)
	s.executeTemplate(w, "newuserok.html", &d, http.StatusOK)
	return http.StatusOK
}

func checkLoginName(name string, tr func(string) string) (string, bool) {
	if len(name) < 3 {
		return tr("Login must have at least three characters"), false
	}
	if name[0] < 'a' && name[0] > 'z' {
		return tr("Login must start with lowercase letter"), false
	}
	for i := 1; i < len(name); i++ {
		if !(name[i] >= 'a' && name[i] <= 'z') && !(name[i] >= '0' && name[i] <= '9') {
			return tr("Only lowercase letters and digits allowed"), false
		}
	}
	return "", true
}

func randomPassword() ([]byte, error) {
	n := base64.StdEncoding.EncodedLen(6)
	buf := make([]byte, 6+n)
	b := buf[:6]
	randomPass := buf[6:]
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	base64.StdEncoding.Encode(randomPass, b)
	return randomPass, nil
}
