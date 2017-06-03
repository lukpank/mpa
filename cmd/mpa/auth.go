// Copyright 2016 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	sessionDuration   = 3600 // session duration in seconds
	sessionCookieName = "mpa_sid"
)

type sessionKey struct{}

var ErrAuth = errors.New("failed to authenticate")

func (s *server) authenticate(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := strings.HasPrefix(r.URL.Path, "/api/")
		path := r.URL.Path
		if r.URL.RawQuery != "" {
			path += "?" + r.URL.RawQuery
		}

		cookie, err := r.Cookie(sessionCookieName)
		var extend bool
		var session SessionData
		if err == nil {
			extend, session, err = s.s.CheckSession(cookie.Value, sessionDuration*time.Second)
			if err == nil {
				r = r.WithContext(context.WithValue(r.Context(), sessionKey{}, session))
			}
		}
		if err == ErrNoSuchSession || err == http.ErrNoCookie {
			s.loginPage(w, r, path, "", !api, http.StatusUnauthorized)
			return
		}
		if err != nil {
			log.Println(err)
			if api {
				http.Error(w, s.tr("Internal server error"), http.StatusInternalServerError)
			} else {
				s.loginPage(w, r, path, s.tr("Internal server error"), !api, http.StatusInternalServerError)
			}
			return
		}

		if extend {
			s.setSessionCookie(w, cookie.Value, 2*sessionDuration)
		}
		if session.RequirePasswordChange {
			if api {
				http.Error(w, s.tr("Password change required"), http.StatusUnauthorized)
			} else {
				s.ServeChangePassword(w, r)
			}
		} else {
			h(w, r)
		}
	}
}

func (s *server) authorizeAsAdmin(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := s.SessionData(r)
		if err == nil && session.Admin {
			h(w, r)
			return
		}
		if err != nil {
			s.internalError(w, err, s.tr("Session error"))
			return
		}
		s.error(w, s.tr("Authorization error"), s.tr("Admin account required"), http.StatusUnauthorized)
	}
}

func (s *server) SessionData(r *http.Request) (SessionData, error) {
	v := r.Context().Value(sessionKey{})
	if session, ok := v.(SessionData); ok {
		return session, nil
	}
	return SessionData{}, ErrNoSuchSession
}

func (s *server) SessionSetPasswordChanged(r *http.Request) error {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return err
	}
	return s.s.SessionSetPasswordChanged(cookie.Value)
}

func (s *server) ServeLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.loginPage(w, r, "/", "", true, http.StatusUnauthorized)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.parseFormError(w, err)
		return
	}
	login := r.PostForm.Get("login")
	password := r.PostForm.Get("password")
	redirect := r.PostForm.Get("redirect")
	data, err := s.db.AuthenticateUser(login, []byte(password))
	if err != nil {
		if err == ErrAuth {
			s.loginPage(w, r, redirect, s.tr("Incorrect login or password."), true, http.StatusUnauthorized)
		} else {
			log.Println(err)
			s.loginPage(w, r, redirect, s.tr("Internal server error"), true, http.StatusUnauthorized)
		}
		return
	}
	sid, err := s.s.NewSession(sessionDuration*time.Second, data)
	if err != nil {
		s.internalError(w, err, s.tr("Session error"))
		return
	}
	s.setSessionCookie(w, sid, 2*sessionDuration)
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func (s *server) ServeAPILogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, s.tr("Method not allowed"), http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseMultipartForm(4096); err != nil {
		http.Error(w, s.tr("Error parsing form"), http.StatusBadRequest)
		return
	}
	login := r.PostForm.Get("login")
	password := r.PostForm.Get("password")
	data, err := s.db.AuthenticateUser(login, []byte(password))
	if err != nil {
		if err == ErrAuth {
			http.Error(w, s.tr("Incorrect login or password."), http.StatusUnauthorized)
		} else {
			log.Println(err)
			http.Error(w, s.tr("Internal server error"), http.StatusInternalServerError)
		}
		return
	}
	sid, err := s.s.NewSession(sessionDuration*time.Second, data)
	if err != nil {
		log.Println(err)
		http.Error(w, s.tr("Internal server error"), http.StatusInternalServerError)
		return
	}
	s.setSessionCookie(w, sid, 2*sessionDuration)
	w.WriteHeader(http.StatusOK) // for status logging to work properly
}

func (s *server) setSessionCookie(w http.ResponseWriter, sid string, duration int) {
	expires := time.Now().Add(time.Duration(duration) * time.Second)
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Path: "/", Value: sid, MaxAge: duration, Expires: expires, Secure: s.secure})
}

func (s *server) loginPage(w http.ResponseWriter, r *http.Request, path, msg string, fullPage bool, code int) {
	t := "login.html"
	if !fullPage {
		t = "loginapi.html"
	}
	s.executeTemplate(w, t, &struct {
		Lang              string
		Redirect, Message string
	}{s.lang, path, msg}, code)
}

func (s *server) ServeLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		log.Println(err)
	} else {
		s.s.Remove(cookie.Value)
	}
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
