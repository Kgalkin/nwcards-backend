package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/lib/pq"
	"log"
)

type StoreItemData struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	//Price       int    `json:"price"`
}

func (i *StoreItemData) Scan(src interface{}) error {
	uintVal, ok := src.([]uint8)
	if !ok {
		return fmt.Errorf("Data field must be a string, got #{src} instead")
	}
	return json.Unmarshal(uintVal, i)
}

type Tags struct {
	Tags []string
}

type StoreItem struct {
	Id      int64         `json:"id"`
	Data    StoreItemData `json:"data"`
	Price   int           `json:"price"`
	InStock int           `json:"inStock"`
	Tags    []string      `json:"tags"`
}

func (d StoreItemData) String() string {
	s, _ := json.Marshal(d)
	return string(s)
}

func UpdateItemCount(item OrderItem) error {
	data, _ := ItemWithId(item.Id)
	if data.InStock < item.Count {
		er := errors.New("Instock < count")
		log.Println(er)
		return er
	}
	_, er := db.Query("UPDATE store_items SET instock = instock - $1 WHERE id = $2", item.Count, item.Id)
	if er != nil {
		log.Println(er)
		return er
	}
	return nil
}

func ItemWithId(id int64) (*StoreItem, error) {
	rows, err := db.Query("SELECT * FROM store_items WHERE id = $1", id)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	item := new(StoreItem)
	defer rows.Close()
	rows.Next()
	err = rows.Scan(&item.Id, &item.Data, &item.Price, &item.InStock, pq.Array(&item.Tags))
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if err = rows.Err(); err != nil {
		log.Println(err)
		return nil, err
	}
	return item, nil
}

func AllItems() ([]*StoreItem, error) {
	rows, err := db.Query("SELECT * FROM store_items")
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer rows.Close()
	items := make([]*StoreItem, 0)
	for rows.Next() {
		item := new(StoreItem)
		err := rows.Scan(&item.Id, &item.Data, &item.Price, &item.InStock, pq.Array(&item.Tags))
		if err != nil {
			log.Println(err)
			return nil, err
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		log.Println(err)
		return nil, err
	}
	return items, nil
}
