package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
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
	Uuid    string    `json:"uuid"`
	Token   []byte    `json:"token"`
	Error   Error     `json:"error"`
}

type OrderData struct {
	Email          string         `json:"email"`
	PhoneNumber    string         `json:"phoneNumber,omitempty"`
	Index          string         `json:"index"`
	Address        string         `json:"address"`
	Name           string         `json:"name"`
	PvzId          string         `json:"pvzId,omitempty"`
	PvzAddress     string         `json:"pvzAddress,omitempty"`
	DeliveryPrice  int            `json:"deliveryPrice,omitempty"`
	DeliveryOption DeliveryOption `json:"deliveryOption"`
	PostalCode     string         `json:"postalCode"`
	Items          []OrderItem    `json:"items"`
	PaymentLink    string         `json:"payment_link"`
	Comments       string         `json:"comments"`
	AdminComments  string         `json:"adminComments"`
	Links          struct {
		Proof string   `json:"proof"`
		Other []string `json:"other"`
	} `json:"links"`
}

func (o *OrderData) GetDeliveryPrice() int {
	deliveryPrice := o.DeliveryPrice
	if deliveryPrice == 0 {
		deliveryPrice = o.DeliveryOption.Price
	}
	return deliveryPrice
}

type Error struct {
	/*
		codes:
		1 - Create order email not sent;
	*/
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (od *Error) Scan(src interface{}) error {
	val, ok := src.([]uint8)
	if !ok {
		return nil
	}
	return json.Unmarshal(val, od)
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
	Id    int64                  `json:"id"`
	Type  string                 `json:"type"`
	Count int                    `json:"count"`
	Price int                    `json:"price"`
	Tags  []int64                `json:"tags"`
	Data  StoreItemData          `json:"data"`
	Props map[string]interface{} `json:"props,omitempty"`
}

const (
	CREATED            = "created"
	PAYMENT_RECEIVED   = "payment_received"
	READY_FOR_SHIPPING = "ready_for_shipping"
	SENT_TO_CUSTOMER   = "sent_to_customer"
	COMPLETED          = "completed"
	CANCELED           = "canceled"
	ITEM_WRAPPER_TYPE  = "itemWrapper"
)

func getStates() []string {
	return []string{CREATED,
		PAYMENT_RECEIVED,
		SENT_TO_CUSTOMER,
		COMPLETED,
		CANCELED}
}

func GetOrders(params map[string][]string) ([]*Order, error) {
	query, _, _ := ParamsToDbRequest(params, "ORDER BY created desc")
	rows, er := db.Query("SELECT * FROM orders " + query)
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
	upd := make(map[int64]int)
	updateItems := ""
	for _, it := range order.Data.Items {
		var count int
		if it.Type == ITEM_WRAPPER_TYPE {
			count = int(it.Props["countToWrap"].(float64)) * it.Count
		} else {
			count = it.Count
		}
		upd[it.Id] += count
	}
	for id, count := range upd {
		updateItems += fmt.Sprintf("(%d, %d)", id, count)
		delete(upd, id)
		if len(upd) > 0 {
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
		update += fmt.Sprintf("data = '%s'",
			strings.Replace(data.String(), "'", "''", -1))
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
		er := rows.Scan(&order.Id, &order.Data, &order.State, &order.Created, &order.Token, &order.Uuid, &order.Error)
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
		er := fmt.Errorf("Found %d orders with id: %d\n", len(orders), id)
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
	data, er := json.Marshal(order.Data)
	if er != nil {
		log.Println(er)
		return nil, er
	}
	if order.Data.DeliveryPrice == 0 {
		if order.Data.DeliveryOption.Price != 0 && order.Data.DeliveryOption.Price > 0 {
			order.Data.DeliveryPrice = order.Data.DeliveryOption.Price
		}
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

func (o *Order) ApplyDeliveryPriceModifiers() {
	price := o.Data.GetDeliveryPrice()
	if len(o.Data.DeliveryOption.Data.PriceModifiers) > 0 {
		for _, pm := range o.Data.DeliveryOption.Data.PriceModifiers {
			if o.isApplicable(pm.Constraints) {
				if pm.Type == "plus" {
					plus, _ := strconv.Atoi(pm.Value)
					price += plus
				}
				if pm.Type == "%" {
					percent, _ := strconv.ParseFloat(pm.Value, 32)
					price = int(math.Floor(float64(price)*percent) / float64(100))
				}
			}
		}
	}
	o.Data.DeliveryPrice = price
}

func (o *Order) isApplicable(cs []Constraint) bool {
	if len(cs) == 0 {
		return true
	}
	result := true
	for _, c := range cs {
		if c.Type == "orderPriceMin" {
			value, _ := strconv.Atoi(c.Value)
			result = result && (value <= o.getItemsPrice())
		}
	}
	return result
}

func (o *Order) getItemsPrice() int {
	price := 0
	if len(o.Data.Items) > 0 {
		for _, item := range o.Data.Items {
			price = price + item.Price*item.Count
		}
	}
	return price
}
