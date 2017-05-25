// Copyright 2016 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

package main

import (
	"database/sql"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"github.com/mxk/go-sqlite/sqlite3"
)

type DB struct {
	db *sql.DB
}

var ErrSingleThread = errors.New("single threaded sqlite3 is not supported")

func OpenDB(filename string) (*DB, error) {
	if sqlite3.SingleThread() {
		return nil, ErrSingleThread
	}
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil, err
	}
	return &DB{db}, nil
}

func (db *DB) AuthenticateUser(login string, password []byte) error {
	var h []byte
	if err := db.db.QueryRow("SELECT passwordhash FROM users WHERE login=?", login).Scan(&h); err != nil {
		if err == sql.ErrNoRows {
			return ErrAuth
		}
		return err
	}
	err := bcrypt.CompareHashAndPassword(h, password)
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return ErrAuth
	}
	return err
}
