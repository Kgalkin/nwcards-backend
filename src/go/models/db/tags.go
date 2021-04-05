package db

import (
	"fmt"
	"log"
)

func GetTags() ([]*string, error) {
	rows, err := db.Query("SELECT * FROM tags")
	if err != nil {
		log.Println(err)
		return nil, err
	}
	items := make([]*string, 0)
	for rows.Next() {
		item := new(string)
		err := rows.Scan(&item)
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

func CreateTags(tags []string) ([]*string, error) {
	sqlExpr := "INSERT INTO tags (tag) VALUES "
	for _, tag := range tags {
		sqlExpr += fmt.Sprintf("('%s'),", tag)
	}
	sqlExpr = sqlExpr[:len(sqlExpr)-1]
	sqlExpr += " on conflict DO NOTHING returning *"
	rows, err := db.Query(sqlExpr)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	items := make([]*string, 0)
	for rows.Next() {
		item := new(string)
		err := rows.Scan(&item)
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
