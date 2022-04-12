package payments

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"nwcards-backend/src/go/models/db"
	"nwcards-backend/src/go/props"
	"strconv"
	"time"
)

type Timestamp struct {
	time.Time
}

func (t Timestamp) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", t.Format("2006-01-02T15:04:05-07:00"))), nil
}

type InitResponse struct {
	Success     bool   `json:"Success"`
	ErrorCode   string `json:"ErrorCode"`
	TerminalKey string `json:"TerminalKey"`
	Status      string `json:"Status"`
	PaymentId   string `json:"PaymentId"`
	OrderId     string `json:"OrderId"`
	Amount      int    `json:"Amount"`
	PaymentURL  string `json:"PaymentURL"`
	Message     string `json:"Message"`
	Details     string `json:"Details"`
}

type InitRequest struct {
	TerminalKey     string    `json:"TerminalKey"`
	Amount          int       `json:"Amount"`
	OrderId         int       `json:"OrderId"`
	Description     string    `json:"Description"` //not mandatory
	Token           string    `json:"Token"`
	RedirectDueDate Timestamp `json:"RedirectDueDate"`
	//Cрок жизни ссылки (не более 90 дней)
	//Временная метка по стандарту ISO8601 в формате YYYY-MM-DDThh:mm:ss±hh:mm

	//Адрес для получения http нотификаций
	//NotificationURL string `json:"NotificationURL"`

	//SuccessURL string `json:"SuccessURL"`

	//FailURL string `json:"FailURL"`

	Receipt Receipt `json:"Receipt"`
}

type Receipt struct {
	Email string `json:"Email"`
	//Телефон покупателя
	//В формате +{Ц}
	Phone string `json:"Phone"`

	//not mandatory
	EmailCompany string `json:"EmailCompany"`

	//Система налогообложения:
	//osn — общая
	//usn_income — упрощенная (доходы)
	//usn_income_outcome — упрощенная (доходы минус расходы)
	//patent — патентная
	//envd — единый налог на вмененный доход
	//esn — единый сельскохозяйственный налог
	Taxation string `json:"Taxation"`

	Items []Item `json:"Items"`
}

type Item struct {
	Name     string `json:"Name"`
	Quantity int    `json:"Quantity"`
	Amount   int    `json:"Amount"`
	Price    int    `json:"Price"`

	//Признак способа расчета:
	//full_payment — полный расчет
	//full_prepayment — предоплата 100%
	//prepayment — предоплата
	//advance — аванс
	//partial_payment — частичный расчет и кредит
	//credit — передача в кредит
	//credit_payment — оплата кредита
	//not mandatory
	PaymentMethod string `json:"PaymentMethod"`

	//Признак предмета расчета:
	//commodity — товар
	//excise — подакцизный товар
	//job — работа
	//service — услуга
	//gambling_bet — ставка азартной игры
	//gambling_prize — выигрыш азартной игры
	//lottery — лотерейный билет
	//lottery_prize — выигрыш лотереи
	//intellectual_activity — предоставление результатов интеллектуальной деятельности
	//payment — платеж
	//agent_commission — агентское вознаграждение
	//composite — составной предмет расчета
	//another — иной предмет расчета
	//not mandatory; if nil commodity -> передается в кассу
	PaymentObject string `json:"PaymentObject"`

	//Ставка НДС:
	//none — без НДС
	//vat0 — 0%
	//vat10 — 10%
	//vat20 — 20%
	//vat110 — 10/110
	//vat120 — 20/120
	Tax string `json:"Tax"`
}

type PaymentResponse struct {
	TerminalKey string `json:"TerminalKey"`
	OrderID     string `json:"OrderId"`
	Success     bool   `json:"Success"`
	Status      string `json:"Status"`
	PaymentID   int    `json:"PaymentId"`
	ErrorCode   string `json:"ErrorCode"`
	Amount      int    `json:"Amount"`
	CardID      int    `json:"CardId"`
	Pan         string `json:"Pan"`
	ExpDate     string `json:"ExpDate"`
	Token       string `json:"Token"`
	Message     string `json:"Message"`
}

func NewInitForm(order db.Order) *InitRequest {
	iForm := new(InitRequest)
	iForm.TerminalKey = props.Get()["payment.terminal.key"].(string)
	iForm.OrderId = order.Id
	iForm.Receipt = *new(Receipt)
	iForm.Receipt.Email = order.Data.Email
	iForm.Receipt.Taxation = "usn_income"
	iForm.Receipt.EmailCompany = props.Get()["company.email"].(string)
	numberOfItems := len(order.Data.Items)
	if order.Data.DeliveryOption.Price > 0 {
		numberOfItems += 1
	}
	iForm.Receipt.Items = make([]Item, numberOfItems)
	amount := 0
	for i, item := range order.Data.Items {
		amount += item.Price * item.Count
		iForm.Receipt.Items[i] = *toItem(item)
	}
	iForm.Receipt.Items[numberOfItems-1] = Item{
		Name:          order.Data.DeliveryOption.Description,
		Quantity:      1,
		Amount:        order.Data.DeliveryOption.Price * 100,
		Price:         order.Data.DeliveryOption.Price * 100,
		PaymentMethod: "full_prepayment",
		PaymentObject: "commodity",
		Tax:           "none"}
	amount += order.Data.DeliveryOption.Price
	iForm.Amount = amount * 100
	iForm.generateToken()
	iForm.RedirectDueDate = Timestamp{time.Now().Add(time.Minute * time.Duration(20))}
	return iForm
}

func toItem(orderItem db.OrderItem) *Item {
	item := new(Item)
	item.Tax = "none"
	item.PaymentObject = "commodity"
	item.PaymentMethod = "full_prepayment"
	item.Name = orderItem.Data.Title
	item.Price = orderItem.Price * 100
	item.Quantity = orderItem.Count
	item.Amount = orderItem.Count * item.Price
	return item
}

func (i *InitRequest) generateToken() {
	str := strconv.Itoa(i.Amount) +
		i.Description +
		strconv.Itoa(i.OrderId) +
		props.Get()["payment.token.pss"].(string) +
		i.TerminalKey
	i.Token = sha256(str)
}

func sha256(str string) string {
	hasher := sha1.New()
	hasher.Write([]byte(str))
	return hex.EncodeToString(hasher.Sum(nil))
}
