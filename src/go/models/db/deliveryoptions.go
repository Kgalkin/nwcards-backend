package db

import (
	"encoding/json"
	"fmt"
	"log"
)

type DeliveryOption struct {
	Description string `json:"description"`
	Price       int    `json:"price"`
	Id          int    `json:"id"`
}

func (*DeliveryOption) Scan(src interface{}) error {
	uintVal, ok := src.([]uint8)
	if !ok {
		return fmt.Errorf("Data field must be a string, got #{src} instead\n")
	}
	return json.Unmarshal(uintVal, &DeliveryOption{})
}

func resolveDeliveryOption(id int) (*DeliveryOption, error) {
	rows, er := db.Query("SELECT * FROM delivery_options WHERE id = $1", id)
	if er != nil {
		log.Println(er)
		return nil, er
	}
	defer rows.Close()
	options := make([]*DeliveryOption, 0)
	for rows.Next() {
		option := new(DeliveryOption)
		er := rows.Scan(&option.Description, &option.Price, &option.Id)
		if er != nil {
			log.Println(er)
			return nil, er
		}
		options = append(options, option)
	}
	if len(options) != 1 {
		return nil, fmt.Errorf("Can not define delivery option with ID: %d\n", id)
	}
	return options[0], nil
}

func GetDeliveryOptions() ([]*DeliveryOption, error) {
	rows, er := db.Query("SELECT * FROM delivery_options")
	if er != nil {
		log.Println(er)
		return nil, er
	}
	defer rows.Close()
	options := make([]*DeliveryOption, 0)
	for rows.Next() {
		option := new(DeliveryOption)
		er := rows.Scan(&option.Description, &option.Price, &option.Id)
		if er != nil {
			log.Println(er)
			return nil, er
		}
		options = append(options, option)
	}
	return options, nil
}
