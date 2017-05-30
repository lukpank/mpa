// Copyright 2016 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

package main

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/bgentry/speakeasy"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

type DB struct {
	db *sql.DB

	filesDir   string
	imagesDir  string
	previewDir string
	uploadDir  string

	filesMu sync.Mutex // protect against concurrent file write operations
}

var ErrSingleThread = errors.New("single threaded sqlite3 is not supported")

func OpenDB(filename string) (*DB, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil, err
	}
	filesDir := filename + ".mpa"
	return &DB{db: db, filesDir: filesDir,
		imagesDir:  filepath.Join(filesDir, "images"),
		previewDir: filepath.Join(filesDir, "preview"),
		uploadDir:  filepath.Join(filesDir, "upload")}, nil
}

func (db *DB) EnsureDirs() error {
	info, err := os.Stat(db.filesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("storage directory for images %s does not exist, create or rename it if you renamed database file", db.filesDir)
		}
		return err
	} else if !info.IsDir() {
		return fmt.Errorf("file %s exists but is not a directory (expected storage directory for images)", db.filesDir)
	}
	if err := ensureDirExists(db.imagesDir, 0755); err != nil {
		return err
	}
	if err := ensureDirExists(db.previewDir, 0755); err != nil {
		return err
	}
	return ensureDirExists(db.uploadDir, 0755)
}

func ensureDirExists(path string, perm os.FileMode) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return os.Mkdir(path, perm)
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("file %s exists but is not a directory", path)
	}
	return nil
}

func (db *DB) Init(lang string) (err error) {
	tx, err := db.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = createMPATable(tx, lang)
	if err == nil {
		_, err = tx.Exec(`
CREATE TABLE users(
uid INTEGER PRIMARY KEY,
login TEXT UNIQUE,
name TEXT,
surname TEXT,
email TEXT UNIQUE,
admin_level INTEGER,
require_password_change INTEGER DEFAULT 1,
passwordhash BLOB)
`)
	}
	if err == nil {
		_, err = tx.Exec(`
CREATE TABLE albums(
aid INTEGER PRIMARY KEY,
owner_id INTEGER,
image_id INTEGER,
is_portrait INTEGER,
created INTEGER,
modified INTEGER,
name TEXT)
`)
	}
	if err == nil {
		_, err = tx.Exec(`
CREATE TABLE images(
iid INTEGER PRIMARY KEY,
album_id INTEGER,
sha256sum TEXT,
title TEXT,
is_portrait INTEGER,
created INTEGER,
owner_file_name TEXT)
`)
	}
	if err == nil {
		err = db.askAddUser(tx)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (db *DB) askAddUser(tx Execer) error {
	sc := bufio.NewScanner(os.Stdin)
	login, err := ask(sc, "Login: ")
	if err != nil {
		return err
	}
	name, err := ask(sc, "Name: ")
	if err != nil {
		return err
	}
	surname, err := ask(sc, "Surname: ")
	if err != nil {
		return err
	}
	email, err := ask(sc, "Email: ")
	if err != nil {
		return err
	}
	pass, err := speakeasy.Ask("Password: ")
	if err != nil {
		return err
	}
	repeat, err := speakeasy.Ask("Retype password: ")
	if err != nil {
		return err
	}
	if repeat != pass {
		return errors.New("failed to add user: passwords do not match")
	}
	return db.AddUser(tx, login, name, surname, email, 1, []byte(pass))
}

func ask(sc *bufio.Scanner, prompt string) (string, error) {
	if _, err := fmt.Print(prompt); err != nil {
		return "", err
	}
	if !sc.Scan() {
		if err := sc.Err(); err != nil {
			return "", err
		}
		return "", io.EOF
	}
	return sc.Text(), nil
}

func createMPATable(tx *sql.Tx, lang string) error {
	_, err := tx.Exec("CREATE TABLE mpa(key TEXT UNIQUE, value TEXT)")
	if err == nil {
		_, err = tx.Exec("INSERT INTO mpa (key, value) VALUES ('db_version', '1')")
	}
	if err == nil {
		_, err = tx.Exec("INSERT INTO mpa (key, value) VALUES ('lang', ?)", lang)
	}
	return err
}

func (db *DB) GetMPAOptions() (lang string, err error) {
	rows, err := db.db.Query("SELECT key, value FROM mpa")
	if err != nil {
		return "", err
	}
	defer rows.Close()
	mask := 0
	var key, value string
	for rows.Next() {
		err := rows.Scan(&key, &value)
		if err != nil {
			return "", err
		}
		switch key {
		case "db_version":
			mask |= 1
			i, err := strconv.Atoi(value)
			if err != nil {
				return "", fmt.Errorf("error parsing db_version: %v", err)
			}
			if i != 1 {
				return "", fmt.Errorf("expected db_version 1 but found %d", i)
			}
		case "lang":
			mask |= 2
			lang = value
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if mask&1 == 0 {
		return "", errors.New("missing db_version in mpa table")
	}
	if mask&2 == 0 {
		return "", errors.New("missing lang in mpa table")
	}
	return
}

type Execer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

func (db *DB) AddUser(tx Execer, login, name, surname, email string, adminLevel int, password []byte) error {
	p, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = tx.Exec("INSERT INTO users (login, name, surname, email, admin_level, passwordhash) VALUES (?, ?, ?, ?, ?, ?)", login, name, surname, email, adminLevel, p)
	return err
}

func (db *DB) AuthenticateUser(login string, password []byte) (int64, bool, error) {
	var uid int64
	var admin bool
	var h []byte
	if err := db.db.QueryRow("SELECT uid, admin_level, passwordhash FROM users WHERE login=?", login).Scan(&uid, &admin, &h); err != nil {
		if err == sql.ErrNoRows {
			return 0, false, ErrAuth
		}
		return 0, false, err
	}
	err := bcrypt.CompareHashAndPassword(h, password)
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return 0, false, ErrAuth
	}
	return uid, admin, err
}
