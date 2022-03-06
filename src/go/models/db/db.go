package db

import (
	"NorthwindREST/src/go/props"
	"database/sql"
	"fmt"
)

var db *sql.DB

func init() {
	props := props.Get()
	dataSource := fmt.Sprintf("host = %s user=%s password=%s dbname=%s",
		props["db.host"], props["db.user"], props["db.pass"], props["db.name"])
	var err error
	db, err = sql.Open("postgres", dataSource)
	if err != nil {
		panic(err)
	}
	if err = db.Ping(); err != nil {
		panic(err)
	}
}
