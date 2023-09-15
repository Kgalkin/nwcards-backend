package db

import (
	"encoding/json"
	"fmt"
	"log"
)

type IEffect interface {
	IsApplicable(order *Order) bool
	Apply(order *Order) error
}

type Bonus struct {
	Id       string                 `json:"id"`
	State    string                 `json:"state"`
	IsGlobal bool                   `json:"isGlobal"`
	Type     string                 `json:"type"`
	Data     map[string]interface{} `json:"data"`
}

func GetBonuses() ([]*Bonus, error) {
	return getBonusesFromQuery("SELECT * from bonuses")
}

func GetBonusesByIds(ids []string) ([]*Bonus, error) {
	strIds := ""
	for i, id := range ids {
		if i != 0 {
			strIds += ","
		}
		strIds += id
	}
	bonuses, er := getBonusesFromQuery(fmt.Sprintf("SELECT * from bonuses where id in (%s)", strIds))
	if er != nil {
		return nil, er
	}
	return bonuses, nil
}

func getBonusesFromQuery(query string) ([]*Bonus, error) {
	rows, er := db.Query(query)
	if er != nil {
		log.Println(er)
		return nil, er
	}
	defer rows.Close()
	bonuses := make([]*Bonus, 0)
	for rows.Next() {
		bonus := new(Bonus)
		var dataStr []uint8
		er := rows.Scan(&bonus.Id, &bonus.State, &bonus.IsGlobal, &dataStr, &bonus.Type)
		if er != nil {
			log.Println(er)
			return nil, er
		}
		er = json.Unmarshal(dataStr, &bonus.Data)
		if er != nil {
			log.Println(er)
			return nil, er
		}
		bonuses = append(bonuses, bonus)
	}
	return bonuses, nil
}
