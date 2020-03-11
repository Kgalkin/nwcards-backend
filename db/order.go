package db

import (
	"encoding/json"
	"log"
)

type Order struct {
	Id      int         `json:"id"`
	Email   string      `json:"email"`
	Index   string      `json:"index"`
	Address string      `json:"address"`
	Items   []OrderItem `json:"items"`
}

type OrderItem struct {
	Id    int64 `json:"id"`
	Count int   `json:"count"`
}

func CreateOrder(order Order) error {
	data, _ := json.Marshal(order.Items)
	_, er := db.Query("INSERT INTO orders (data) VALUES ($1)", data)
	if er != nil {
		log.Fatal(er)
		return er
	}
	return nil
}
