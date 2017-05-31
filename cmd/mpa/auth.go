// Copyright 2016 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

package main

import (
	"errors"
	"net/http"
	"strings"
	"time"
)

const (
	sessionDuration   = 3600 // session duration in seconds
	sessionCookieName = "mpa_sid"
)

var ErrAuth = errors.New("failed to authenticate")

func (s *server) authenticate(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err == nil {
			var extend bool
			if extend, err = s.s.CheckSession(cookie.Value, sessionDuration*time.Second); err == nil {
				if extend {
					s.setSessionCookie(w, cookie.Value, 2*sessionDuration)
				}
				h(w, r)
				return
			}
		}
		api := strings.HasPrefix(r.URL.Path, "/api/")
		if err != nil && err != ErrAuth && err != http.ErrNoCookie {
			if api {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			} else {
				s.internalError(w, err, s.tr("Authentication error"))
			}
			return
		}
		if api {
			w.WriteHeader(http.StatusUnauthorized)
		}
		path := r.URL.Path
		if r.URL.RawQuery != "" {
			path += "?" + r.URL.RawQuery
		}
		s.loginPage(w, r, path, "", !api)
	}
}

func (s *server) authorizeAsAdmin(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := s.SessionData(r)
		if err == nil && session.Admin {
			h(w, r)
			return
		}
		if err != nil && err != ErrAuth {
			s.internalError(w, err, s.tr("Authorization error"))
			return
		}
		s.error(w, s.tr("Authorization error"), s.tr("Admin account required"), http.StatusUnauthorized)
	}
}

func (s *server) SessionData(r *http.Request) (SessionData, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return SessionData{}, err
	}
	return s.s.SessionData(cookie.Value)
}

func (s *server) serveLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.error(w, s.tr("Method not allowed"), s.tr("Please use POST."), http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.parseFormError(w, err)
		return
	}
	login := r.PostForm.Get("login")
	password := r.PostForm.Get("password")
	redirect := r.PostForm.Get("redirect")
	uid, admin, err := s.db.AuthenticateUser(login, []byte(password))
	if err != nil {
		if err == ErrAuth {
			w.WriteHeader(http.StatusUnauthorized)
			s.loginPage(w, r, redirect, s.tr("Incorrect login or password."), true)
		} else {
			s.internalError(w, err, s.tr("Authentication error"))
		}
		return
	}
	sid, err := s.s.NewSession(sessionDuration*time.Second, SessionData{uid, admin})
	if err != nil {
		s.internalError(w, err, s.tr("Authentication error"))
		return
	}
	s.setSessionCookie(w, sid, 2*sessionDuration)
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func (s *server) setSessionCookie(w http.ResponseWriter, sid string, duration int) {
	expires := time.Now().Add(time.Duration(duration) * time.Second)
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Path: "/", Value: sid, MaxAge: duration, Expires: expires, Secure: s.secure})
}

func (s *server) loginPage(w http.ResponseWriter, r *http.Request, path, msg string, fullPage bool) {
	s.executeTemplate(w, "login.html", &struct {
		Lang              string
		Redirect, Message string
		FullPage          bool
	}{s.lang, path, msg, fullPage}, http.StatusOK)
}

func (s *server) serveLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Path: "/", MaxAge: -1, Secure: s.secure})
	path := strings.TrimPrefix(r.URL.Path, "/logout")
	if len(path) == len(r.URL.Path) || path == "" {
		path = "/"
	}
	if r.URL.RawQuery != "" {
		path += "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, path, http.StatusSeeOther)
}
