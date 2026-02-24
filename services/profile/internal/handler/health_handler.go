package handler

import (
	"database/sql"
	"net/http"
)

func Health() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }
}

func Ready(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if err := db.Ping(); err != nil {
			w.WriteHeader(503)
			return
		}
		w.WriteHeader(200)
	}
}
