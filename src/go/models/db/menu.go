package db

type MenuItem struct {
	Name string `json:"name"`
	Data Map    `json:"data"`
}

func GetMenuItems() ([]*MenuItem, error) {
	items := make([]*MenuItem, 0)
	rows, er := db.Query("SELECT * FROM menu")
	if er != nil {
		return nil, er
	}
	if er = rows.Err(); er != nil {
		return nil, er
	}
	defer rows.Close()
	for rows.Next() {
		item := new(MenuItem)
		er = rows.Scan(&item.Name, &item.Data)
		if er != nil {
			return nil, er
		}
		items = append(items, item)
	}
	return items, nil
}
