package db

import (
	"database/sql"
	"fmt"
	"log"
)

type Tag struct {
	Value string `json:"value"`
	Id    int64  `json:"id"`
}

func GetTags() ([]*Tag, error) {
	rows, err := db.Query("SELECT * FROM tags")
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer rows.Close()
	return readTags(rows)
}

func CreateTags(tags []string) ([]*Tag, error) {
	sqlExpr := "INSERT INTO tags (value) VALUES "
	for _, tag := range tags {
		sqlExpr += fmt.Sprintf("('%s'),", tag)
	}
	sqlExpr = sqlExpr[:len(sqlExpr)-1]
	sqlExpr += " on conflict DO NOTHING; SELECT * FROM tags;"
	rows, err := db.Query(sqlExpr)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer rows.Close()
	return readTags(rows)
}

func readTags(rows *sql.Rows) ([]*Tag, error) {
	items := make([]*Tag, 0)
	for rows.Next() {
		item := new(Tag)
		err := rows.Scan(&item.Value, &item.Id)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		log.Println(err)
		return nil, err
	}
	return items, nil
}
