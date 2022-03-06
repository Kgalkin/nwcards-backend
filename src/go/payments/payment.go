package payments

import (
	"NorthwindREST/src/go/models/db"
	"NorthwindREST/src/go/props"
)

type InitForm struct {
	TerminalKey string `json:"TerminalKey"`
	Amount      int    `json:"Amount"`
	OrderId     int    `json:"OrderId"`
	//not mandatory
	Description string `json:"Description"`
	Token       string `json:"Token"`

	//Cрок жизни ссылки (не более 90 дней)
	//Временная метка по стандарту ISO8601 в формате YYYY-MM-DDThh:mm:ss±hh:mm
	//RedirectDueDate `json:"RedirectDueDate"`

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
	//PaymentMethod string `json:"PaymentMethod"`

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

func NewInitForm(order db.Order) *InitForm {
	iForm := new(InitForm)
	iForm.TerminalKey = props.Get()["payment.terminal.key"]
	iForm.OrderId = order.Id
	amount := 0
	for _, item := range order.Data.Items {
		amount += item.Price * item.Count
	}
	amount += order.Data.DeliveryOption.Price
	iForm.Amount = amount
	return iForm
}
