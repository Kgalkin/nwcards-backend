package db

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

type Order struct {
	Id      int       `json:"id"`
	Created time.Time `json:"created"`
	State   string    `json:"state"`
	Data    OrderData `json:"data"`
}

func (o *OrderData) Scan(src interface{}) error {
	val, ok := src.([]uint8)
	if !ok {
		return fmt.Errorf("Data field must be a string, got #{src} instead")
	}
	return json.Unmarshal(val, o)
}

type OrderData struct {
	Email          string      `json:"email"`
	Index          string      `json:"index"`
	Address        string      `json:"address"`
	Name           string      `json:"name"`
	DeliveryOption string      `json:"deliveryOption"`
	Items          []OrderItem `json:"items"`
}

type OrderItem struct {
	Id    int64 `json:"id"`
	Count int   `json:"count"`
}

func GetOrders() ([]*Order, error) {
	rows, er := db.Query("SELECT * FROM orders")
	if er != nil {
		log.Println(er)
		return nil, er
	}
	defer rows.Close()
	orders := make([]*Order, 0)
	for rows.Next() {
		order := new(Order)
		er := rows.Scan(&order.Id, &order.Data, &order.State, &order.Created)
		if er != nil {
			log.Println(er)
			return nil, er
		}
		orders = append(orders, order)
	}
	if er = rows.Err(); er != nil {
		return nil, er
	}
	return orders, nil
}

func CreateOrder(order Order) error {
	data, _ := json.Marshal(order.Data)
	_, er := db.Query("INSERT INTO orders (data) VALUES ($1)", data)
	if er != nil {
		log.Println(er)
		return er
	}
	return nil
}
