package db

import "encoding/json"

type Map map[string]interface{}

func (data *Map) Scan(src interface{}) error {
	uintVal, ok := src.([]uint8)
	if !ok {
		return nil
	}
	return json.Unmarshal(uintVal, data)
}
