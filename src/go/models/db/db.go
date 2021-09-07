package db

import (
	"database/sql"
	_ "github.com/lib/pq"
)

var db *sql.DB

func InitDB(dataSourceName string) {
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		panic(err)
	}
	if err = db.Ping(); err != nil {
		panic(err)
	}
}
