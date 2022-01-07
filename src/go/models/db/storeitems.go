package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/lib/pq"
	"log"
	"strconv"
)

type StoreItemData struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Links       struct {
		Original string   `json:"original"`
		Short    string   `json:"short"`
		Other    []string `json:"other"`
	} `json:"links"`
	//Price       int    `json:"price"`
}

func (i *StoreItemData) Scan(src interface{}) error {
	uintVal, ok := src.([]uint8)
	if !ok {
		return fmt.Errorf("Data field must be a string, got #{src} instead")
	}
	return json.Unmarshal(uintVal, i)
}

type StoreItem struct {
	Id      int64         `json:"id"`
	Data    StoreItemData `json:"data"`
	Price   int           `json:"price"`
	InStock int           `json:"inStock"`
	Tags    []int64       `json:"tags"`
}

type StoreItemsPage struct {
	Items      []*StoreItem `json:"items"`
	TotalCount int64        `json:"totalCount"`
	NextOffset int64        `json:"nextOffset"`
}

func (d StoreItemData) String() string {
	s, _ := json.Marshal(d)
	return string(s)
}

func UpdateItemCount(item OrderItem) error {
	data, _ := GetItem(item.Id)
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

func UpdateItem(item StoreItem) (*StoreItem, error) {
	_, er := db.Query(`UPDATE store_items SET data = $2,
 	price = $3,
  	instock = $4,
  	tags = $5 
  	WHERE id = $1`,
		item.Id, item.Data.String(), item.Price, item.InStock, pq.Array(item.Tags))
	if er != nil {
		log.Println(er)

	}
	return GetItem(item.Id)
}

func GetItem(id int64) (*StoreItem, error) {
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

func CreateItem(i StoreItem) (*StoreItem, error) {
	rows, err := db.Query("INSERT INTO store_items (data, price, instock, tags) VALUES ($1, $2, $3, $4) returning *",
		i.Data.String(), i.Price, i.InStock, pq.Array(i.Tags))
	if err != nil {
		log.Println(err)
		return nil, err
	}
	items, err := rowsToStoreItems(rows)
	return items[0], err
}

func GetAllItems(params map[string][]string) (*StoreItemsPage, error) {
	query, whereQuery, offset := paramsToDbRequest(params)
	rows, err := db.Query("SELECT * FROM store_items" +
		query)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer rows.Close()
	sip := StoreItemsPage{}
	sip.Items, err = rowsToStoreItems(rows)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	countRow, err := db.Query("SELECT count(*) FROM store_items" +
		whereQuery)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	countRow.Next()
	err = countRow.Scan(&sip.TotalCount)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	sip.NextOffset = offset
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return &sip, nil
}

func paramsToDbRequest(params map[string][]string) (string, string, int64) {
	where := ""
	limit := "100"
	size := params["size"]
	if len(size) > 0 {
		limit = size[0]
	}
	offset := "0"
	offsetQuery := params["offset"]
	if len(offsetQuery) > 0 {
		offset = offsetQuery[0]
	}
	tags := params["tags"]
	if len(tags) > 0 {
		taglist := ""
		for _, tag := range tags {
			taglist += ", " + tag
		}
		method := "&&"
		if m := params["method"]; len(m) > 0 && m[0] == "and" {
			method = "@>"
		}
		where = fmt.Sprintf("\nWHERE tags %s '{%s}'", method, taglist[2:])
	}
	offsetInt, err := strconv.ParseInt(offset, 10, 64)
	if err != nil {
		panic(err)
	}
	sizeInt, err := strconv.ParseInt(limit, 10, 64)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s\nORDER BY id\nLIMIT %s\nOFFSET %s", where, limit, offset),
		where,
		offsetInt + sizeInt
}

func rowsToStoreItems(rows *sql.Rows) ([]*StoreItem, error) {
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
	if err := rows.Err(); err != nil {
		log.Println(err)
		return nil, err
	}
	return items, nil
}
