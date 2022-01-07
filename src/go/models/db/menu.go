package db

import "github.com/lib/pq"

type MenuItem struct {
	Name string  `json:"name"`
	Tags []int64 `json:"tags"`
}

func GetMenuItems() ([]*MenuItem, error) {
	items := make([]*MenuItem, 0)
	rows, er := db.Query("SELECT * FROM menu")
	if er != nil {
		return nil, er
	}
	for rows.Next() {
		item := new(MenuItem)
		er = rows.Scan(&item.Name, pq.Array(&item.Tags))
		if er != nil {
			return nil, er
		}
		items = append(items, item)
	}
	return items, nil
}
