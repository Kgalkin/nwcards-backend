package db

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"
)

type Timestamp time.Time

func (t Timestamp) MarshalJSON() ([]byte, error) {
	stamp := fmt.Sprintf("%d", time.Time(t).UnixMilli())
	return []byte(stamp), nil
}

type Order struct {
	Id      int       `json:"id"`
	Created Timestamp `json:"created"`
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
	Price int   `json:"price"`
}

func GetOrders() ([]*Order, error) {
	rows, er := db.Query("SELECT * FROM orders ORDER BY created desc")
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
	ids := ""
	for i, item := range order.Data.Items {
		if i > 0 {
			ids += ","
		}
		ids += strconv.FormatInt(item.Id, 10)
	}
	items, er := GetItems(map[string][]string{"ids": {ids}})
	if er != nil {
		log.Println(er)
		return er
	}
	for i, _ := range order.Data.Items {
		item := &order.Data.Items[i]
		for _, it := range items.Items {
			if it.Id == item.Id {
				item.Price = it.Price
				break
			}
		}

	}
	data, _ := json.Marshal(order.Data)
	_, er = db.Exec("INSERT INTO orders (data) VALUES ($1)", data)
	if er != nil {
		log.Println(er)
		return er
	}
	return nil
}
