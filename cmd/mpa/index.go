package main

import (
	"log"
	"net/http"
)

func (s *server) ServeIndex(w http.ResponseWriter, r *http.Request) {
	session, err := s.SessionData(r)
	if err != nil {
		log.Println(err)
		s.internalError(w, err, s.tr("Session error"))
		return
	}
	me, others, err := s.db.MeAndOtherUsers(session.Uid)
	if err != nil {
		log.Println(err)
		s.internalError(w, err, s.tr("Internal server error"))
		return
	}
	data := struct {
		Lang   string
		Admin  bool
		Me     userAlbusCnt
		Others []userAlbusCnt
	}{s.lang, session.Admin, me, others}
	s.executeTemplate(w, "index.html", &data, http.StatusOK)
}

type userAlbusCnt struct {
	Login     string
	Name      string
	Surname   string
	AlbumsCnt int64
}

func (db *DB) MeAndOtherUsers(uid int64) (me userAlbusCnt, others []userAlbusCnt, err error) {
	rows, err := db.db.Query(`
SELECT users.uid, users.login, users.name, users.surname, count(albums.owner_id)
FROM users LEFT OUTER JOIN albums
ON users.uid = albums.owner_id
GROUP BY users.uid
ORDER BY users.surname, users.name
`)
	if err != nil {
		return userAlbusCnt{}, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var u userAlbusCnt
		var id int64
		if err := rows.Scan(&id, &u.Login, &u.Name, &u.Surname, &u.AlbumsCnt); err != nil {
			return userAlbusCnt{}, nil, err
		}
		if id != uid {
			others = append(others, u)
		} else {
			me = u
		}
	}
	if err := rows.Err(); err != nil {
		return userAlbusCnt{}, nil, err
	}
	return
}
