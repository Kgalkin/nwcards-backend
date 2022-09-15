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
		Preview  string   `json:"preview"`
		Other    []string `json:"other"`
	} `json:"links"`
	//Price       int    `json:"price"`
}

func (id *StoreItemData) Scan(src interface{}) error {
	uintVal, ok := src.([]uint8)
	if !ok {
		return fmt.Errorf("Data field must be a string, got #{src} instead")
	}
	return json.Unmarshal(uintVal, id)
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

func (id *StoreItemData) String() string {
	s, er := json.Marshal(id)
	if er != nil {
		log.Print(er)
	}
	return string(s)
}

func UpdateItemCount(item OrderItem) error {
	data, _ := GetItem(item.Id)
	if data.InStock < item.Count {
		er := errors.New("Instock < count")
		log.Println(er)
		return er
	}
	_, er := db.Exec("UPDATE store_items SET instock = instock - $1 WHERE id = $2", item.Count, item.Id)
	if er != nil {
		log.Println(er)
		return er
	}
	return nil
}

func UpdateItem(item StoreItem) (*StoreItem, error) {
	er := checkTags(item.Tags)
	if er != nil {
		return nil, er
	}
	_, er = db.Exec(`UPDATE store_items SET data = $2,
 	price = $3,
  	instock = $4,
  	tags = $5 
  	WHERE id = $1`,
		item.Id, item.Data.String(), item.Price, item.InStock, pq.Array(item.Tags))
	if er != nil {
		return nil, er
	}
	return GetItem(item.Id)
}

func checkTags(tags []int64) error {
	var result bool
	allTags, er := GetTags()
	if er != nil {
		return er
	}
	for _, actTag := range tags {
		result = false
		for _, tag := range allTags {
			if actTag == tag.Id {
				result = true
				break
			}
		}
		if !result {
			return errors.New(fmt.Sprintf("No tag found, id: %d", actTag))
		}
	}
	return nil
}

func GetItem(id int64) (*StoreItem, error) {
	rows, err := db.Query("SELECT * FROM store_items WHERE id = $1", id)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer rows.Close()
	item := new(StoreItem)
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
	err := checkTags(i.Tags)
	rows, err := db.Query("INSERT INTO store_items (data, price, instock, tags) VALUES ($1, $2, $3, $4) returning *",
		i.Data.String(), i.Price, i.InStock, pq.Array(i.Tags))
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer rows.Close()
	items, err := rowsToStoreItems(rows)
	return items[0], err
}

func GetItems(params map[string][]string) (*StoreItemsPage, error) {
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
	defer countRow.Close()
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
		if m := params["tags.method"]; len(m) > 0 && m[0] == "and" {
			method = "@>"
		}
		where = fmt.Sprintf("\nWHERE tags %s '{%s}'", method, taglist[2:])
	}
	ids := params["ids"]
	if len(ids) > 0 {
		word := "WHERE"
		if where != "" {
			word = "AND"
		}
		where += fmt.Sprintf("\n%s id in (%s)", word, ids[0])
	}
	offsetInt, err := strconv.ParseInt(offset, 10, 64)
	if err != nil {
		panic(err)
	}
	sizeInt, err := strconv.ParseInt(limit, 10, 64)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s\nORDER BY (instock > 0) desc, id\nLIMIT %s\nOFFSET %s", where, limit, offset),
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
