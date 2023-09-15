package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
)

type Tag struct {
	Value    string `json:"value"`
	Id       int64  `json:"id"`
	IsSystem bool   `json:"isSystem,omitempty"`
	Data     Map    `json:"data,omitempty"`
}

type Map map[string]interface{}

func (data *Map) Scan(src interface{}) error {
	uintVal, ok := src.([]uint8)
	if !ok {
		return nil
	}
	return json.Unmarshal(uintVal, data)
}

func GetTags(params map[string][]string) ([]*Tag, error) {
	isSystem := params["isSystem"]
	query := "SELECT * FROM tags"
	if isSystem != nil {
		query += " WHERE is_system = " + isSystem[0]
	}
	rows, err := db.Query(query)
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
		err := rows.Scan(&item.Value, &item.Id, &item.Data, &item.IsSystem)
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
