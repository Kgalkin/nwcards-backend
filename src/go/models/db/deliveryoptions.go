package db

import (
	"encoding/json"
	"fmt"
	"log"
)

type DeliveryOption struct {
	Description string       `json:"description"`
	Price       int          `json:"price,omitempty"`
	Id          int          `json:"id"`
	Data        DeliveryData `json:"data,omitempty"`
}

type DeliveryData struct {
	Description            string                  `json:"description,omitempty"`
	Constraints            []DeliveryConstraint    `json:"constraints,omitempty"`
	AdditionalRequirements []AdditionalRequirement `json:"additionalRequirements,omitempty"`
	PriceRequest           bool                    `json:"priceRequest,omitempty"`
	PriceModifiers         []PriceModifier         `json:"priceModifiers,omitempty"`
}

type AdditionalRequirement struct {
	Type        string            `json:"type"`
	Values      []string          `json:"values"`
	Disabled    bool              `json:"disabled,omitempty"`
	Placeholder string            `json:"placeholder,omitempty"`
	Description string            `json:"description,omitempty"`
	LabelStyle  map[string]string `json:"labelStyle,omitempty"`
}

type PriceModifier struct {
	Type        string       `json:"type"`
	Value       string       `json:"value"`
	Order       string       `json:"priority"`
	Constraints []Constraint `json:"constraints"`
}

type Constraint struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}
type DeliveryConstraint struct {
	Type         string `json:"type,omitempty"`
	Max          int    `json:"max,omitempty"`
	Min          int    `json:"min,omitempty"`
	Text         string `json:"text,omitempty"`
	FieldName    string `json:"fieldName,omitempty"`
	Equals       string `json:"equals,omitempty"`
	FailOnAbsent bool   `json:"failOnAbsent,omitempty"`
	Priority     int    `json:"priority,omitempty"` //if there is priority on option it is counted as Unique
}

func (*DeliveryOption) Scan(src interface{}) error {
	uintVal, ok := src.([]uint8)
	if !ok {
		return fmt.Errorf("Data field must be a string, got #{src} instead\n")
	}
	return json.Unmarshal(uintVal, &DeliveryOption{})
}

func (data *DeliveryData) Scan(src interface{}) error {
	uintVal, ok := src.([]uint8)
	if !ok {
		return nil
	}
	return json.Unmarshal(uintVal, data)
}

func ResolveDeliveryOption(id int) (*DeliveryOption, error) {
	rows, er := db.Query("SELECT * FROM delivery_options WHERE id = $1", id)
	if er != nil {
		log.Println(er)
		return nil, er
	}
	defer rows.Close()
	options := make([]*DeliveryOption, 0)
	for rows.Next() {
		option := new(DeliveryOption)
		er := rows.Scan(&option.Description, &option.Price, &option.Id, &option.Data)
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
		er := rows.Scan(&option.Description, &option.Price, &option.Id, &option.Data)
		if er != nil {
			log.Println(er)
			return nil, er
		}
		options = append(options, option)
	}
	return options, nil
}
