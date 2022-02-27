package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"log"
	"strconv"
	"time"
)

type Timestamp time.Time

func (t Timestamp) MarshalJSON() ([]byte, error) {
	stamp := fmt.Sprintf("%d", time.Time(t).UnixMilli())
	return []byte(stamp), nil
}

type Uuid struct {
	uuid.UUID
}

func (u *Uuid) Scan(src interface{}) error {
	if err := u.UnmarshalBinary(src.([]byte)); err != nil {
		return err
	}
	return nil
}

type Order struct {
	Id      int       `json:"id"`
	Created Timestamp `json:"created"`
	State   string    `json:"state"`
	Data    OrderData `json:"data"`
	Uuid    string    `json:"uuid"`
	Token   []byte    `json:"token"`
}

type OrderData struct {
	Email          string         `json:"email"`
	Index          string         `json:"index"`
	Address        string         `json:"address"`
	Name           string         `json:"name"`
	DeliveryOption DeliveryOption `json:"deliveryOption"`
	PostalTrack    string         `json:"postalTrack"`
	Items          []OrderItem    `json:"items"`
}

func (od *OrderData) Scan(src interface{}) error {
	val, ok := src.([]uint8)
	if !ok {
		return fmt.Errorf("Data field must be a string, got #{src} instead\n")
	}
	return json.Unmarshal(val, od)
}

func (od *OrderData) String() string {
	s, er := json.Marshal(od)
	if er != nil {
		log.Fatal(er)
	}
	return string(s)
}

type OrderItem struct {
	Id    int64         `json:"id"`
	Count int           `json:"count"`
	Price int           `json:"price"`
	Data  StoreItemData `json:"data"`
}

const (
	CREATED                     = "created"
	CREATED_EMAIL_SENT          = "created(email_sent)"
	PAYMENT_RECEIVED            = "payment_received"
	PAYMENT_RECEIVED_EMAIL_SENT = "payment_received(email_sent)"
	SENT_TO_CUSTOMER            = "sent_to_customer"
	COMPLETED                   = "completed"
	CANCELED                    = "canceled"
)

func getStates() []string {
	return []string{CREATED,
		CREATED_EMAIL_SENT,
		PAYMENT_RECEIVED,
		PAYMENT_RECEIVED_EMAIL_SENT,
		SENT_TO_CUSTOMER,
		COMPLETED,
		CANCELED}
}

func GetOrders() ([]*Order, error) {
	rows, er := db.Query("SELECT * FROM orders ORDER BY created desc")
	if er != nil {
		log.Println(er)
		return nil, er
	}
	defer rows.Close()
	return readOrders(rows)
}

func RevokeOrder(id int) error {
	order, er := GetOrderById(id)
	if order.State == CANCELED ||
		order.State == COMPLETED ||
		order.State == SENT_TO_CUSTOMER {
		return fmt.Errorf("Order in state %s can not be revoked\n", order.State)
	}
	if er != nil {
		log.Println(er)
		return er
	}
	updateItems := ""
	lastICount := len(order.Data.Items) - 1
	for i, it := range order.Data.Items {
		updateItems += fmt.Sprintf("(%d, %d)", it.Id, it.Count)
		if i != lastICount {
			updateItems += ",\n"
		}
	}
	_, er = db.Exec(fmt.Sprintf(`update store_items as si set
				inStock = inStock + c.addInStock
				from (values
          			%s
     			) as c(id, addInStock)
			where c.id = si.id;`, updateItems))
	if er != nil {
		log.Println(er)
		return er
	}
	er = UpdateOrderStateData(id, CANCELED, nil)
	if er != nil {
		log.Println(er)
		return er
	}
	return nil
}

func UpdateOrder(id int, query string) error {
	_, er := db.Exec("UPDATE orders set "+query+" WHERE id = $1", id)
	if er != nil {
		log.Println(er.Error())
		return er
	}
	return er
}

func UpdateOrderStateData(id int, state string, data *OrderData) error {
	update := ""
	if len(state) > 0 {
		if er := checkState(state); er != nil {
			log.Println(er)
			return er
		}
		update += fmt.Sprintf("state = '%s'", state)
	}
	if data != nil {
		if len(update) > 0 {
			update += ", "
		}
		update += fmt.Sprintf("data = '%s'", data.String())
	}
	if len(update) > 0 && id > 0 {
		_, er := db.Exec("UPDATE orders SET "+update+" WHERE id = $1", id)
		if er != nil {
			log.Println(er)
			return er
		}
	}
	return nil
}

func checkState(state string) error {
	for _, st := range getStates() {
		if st == state {
			return nil
		}
	}
	return fmt.Errorf("Unknown state: %s\n", state)
}

func readOrders(rows *sql.Rows) ([]*Order, error) {
	orders := make([]*Order, 0)
	for rows.Next() {
		order := new(Order)
		er := rows.Scan(&order.Id, &order.Data, &order.State, &order.Created, &order.Token, &order.Uuid)
		if er != nil {
			log.Println(er)
			return nil, er
		}
		orders = append(orders, order)
	}
	if er := rows.Err(); er != nil {
		log.Println(er)
		return nil, er
	}
	return orders, nil
}

func GetOrderById(id int) (*Order, error) {
	rows, er := db.Query("SELECT * FROM orders where id = $1", id)
	if er != nil {
		log.Println(er)
		return nil, er
	}
	defer rows.Close()
	orders, er := readOrders(rows)
	if er != nil {
		log.Println(er)
		return nil, er
	}
	if len(orders) != 1 {
		er := fmt.Errorf("Found %d orders with id: %s\n", len(orders), id)
		log.Println(er)
		return nil, er
	}
	return orders[0], nil
}

func GetOrderByUUID(uuid string) (*Order, error) {
	rows, er := db.Query("SELECT * FROM orders where uuid = $1", uuid)
	if er != nil {
		log.Println(er)
		return nil, er
	}
	defer rows.Close()
	orders, er := readOrders(rows)
	if er != nil {
		log.Println(er)
		return nil, er
	}
	if len(orders) != 1 {
		er := fmt.Errorf("Found %d orders with id: %s\n", len(orders), uuid)
		log.Println(er)
		return nil, er
	}
	return orders[0], nil
}

func CreateOrder(order Order) (*Order, error) {
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
		return nil, er
	}
	for i, _ := range order.Data.Items {
		item := &order.Data.Items[i]
		for _, it := range items.Items {
			if it.Id == item.Id {
				item.Data = it.Data
				item.Price = it.Price
				break
			}
		}
	}
	deliveryOption, er := resolveDeliveryOption(order.Data.DeliveryOption.Id)
	if er != nil {
		log.Println(er)
		return nil, er
	}
	order.Data.DeliveryOption = *deliveryOption
	data, er := json.Marshal(order.Data)
	if er != nil {
		log.Println(er)
		return nil, er
	}
	rows, er := db.Query("INSERT INTO orders (data) VALUES ($1) RETURNING *", data)
	if er != nil {
		log.Println(er)
		return nil, er
	}
	defer rows.Close()
	orders, er := readOrders(rows)
	if er != nil {
		log.Println(er)
		return nil, er
	}
	if len(orders) != 1 {
		er := fmt.Errorf("Returned more than expected orders\n")
		log.Println(er)
		return nil, er
	}
	return orders[0], nil
}
