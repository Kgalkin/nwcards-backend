package model

import (
	"encoding/json"
	"fmt"
)

type StoreItemData struct {
	title       string
	description string
	price       int
}
type StoreItem struct {
	id   int64
	data StoreItemData
}

func (i *StoreItemData) Scan(src interface{}) error {
	uintVal, ok := src.([]uint8)
	if !ok {
		return fmt.Errorf("metas field must be a string, got %T instead", src)
	}
	return json.Unmarshal(uintVal, i)
}

func AllItems() ([]*StoreItem, error) {
	rows, err := db.Query("SELECT * FROM store_items")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*StoreItem, 0)
	for rows.Next() {
		item := new(StoreItem)
		err := rows.Scan(&item.id, &item.data)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
