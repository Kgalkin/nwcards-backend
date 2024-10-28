package db

import (
	"database/sql"
	"fmt"
	"nwcards-backend/src/go/props"
	"strconv"
)

var db *sql.DB

func init() {
	props := props.Get()
	dataSource := fmt.Sprintf("host = %s user=%s password=%s dbname=%s%s",
		props["db.host"], props["db.user"], props["db.pass"], props["db.name"], props["db.additional.props"])
	var err error
	db, err = sql.Open("postgres", dataSource)
	if err != nil {
		panic(err)
	}
	if err = db.Ping(); err != nil {
		panic(err)
	}
}

func ParamsToDbRequest(params map[string][]string, orderBy string) (string, string, int64) {
	where := ""
	limit := "100"
	size := params["size"]
	if len(size) > 0 {
		limit = size[0]
	}
	offset := "0"
	offsetQuery := params["offset"]
	if len(offsetQuery) > 0 {
		offset = offsetQuery[0]
	}
	tags := params["tags"]
	if len(tags) > 0 {
		taglist := ""
		for _, tag := range tags {
			taglist += ", " + tag
		}
		method := "&&"
		if m := params["tags.method"]; len(m) > 0 && m[0] == "and" {
			method = "@>"
		}
		where = fmt.Sprintf("\nWHERE tags %s '{%s}'", method, taglist[2:])
	}
	ids := params["ids"]
	if len(ids) > 0 {
		word := "WHERE"
		if where != "" {
			word = "AND"
		}
		where += fmt.Sprintf("\n%s id in (%s)", word, ids[0])
	}
	name := params["name"]
	if len(name) > 0 {
		word := "WHERE"
		if where != "" {
			word = "AND"
		}
		where += fmt.Sprintf("\n%s data->>'name' ilike '%%%s%%'", word, name[0])
	}
	state := params["state"]
	if len(state) > 0 {
		word := "WHERE"
		if where != "" {
			word = "AND"
		}
		where += fmt.Sprintf("\n%s state = '%s'", word, state[0])
	}
	offsetInt, err := strconv.ParseInt(offset, 10, 64)
	if err != nil {
		panic(err)
	}
	sizeInt, err := strconv.ParseInt(limit, 10, 64)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s\n%s\nLIMIT %s\nOFFSET %s", where, orderBy, limit, offset),
		where,
		offsetInt + sizeInt
}
