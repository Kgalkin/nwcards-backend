package ya

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"nwcards-backend/src/go/models/db"
	"nwcards-backend/src/go/props"
	"strconv"
	"strings"
)

func GetPrice(request PriceRequest) (*PriceResponse, error) {
	bs, er := json.Marshal(request)
	if er != nil {
		log.Println(er)
		return nil, er
	}
	post, er := http.NewRequest("POST", props.Get()["ya.delivery.checkprice.url"].(string), bytes.NewBuffer(bs))
	if er != nil {
		log.Println(er)
		return nil, er
	}
	post.Header.Add("Content-Type", "application/json")
	addAuthHeader(post)
	response, er := (&http.Client{}).Do(post)
	if er != nil {
		log.Println(er)
		return nil, er
	}
	if response.StatusCode != 200 {
		er = fmt.Errorf(
			"error checking yandex delivery price. code: %d, destination id: %s\n",
			response.StatusCode, request.Destination.PlatformStationId)
		log.Println(er)
		return nil, er
	}
	priceResponse := PriceResponse{}
	decoder := json.NewDecoder(response.Body)
	decoder.DisallowUnknownFields()
	er = decoder.Decode(&priceResponse)
	if er != nil {
		log.Println(er)
		return nil, er
	}
	return &priceResponse, nil
}

func addAuthHeader(request *http.Request) {
	request.Header.Add("Authorization", "Bearer "+props.Get()["ya.delivery.token"].(string))
}

type PriceRequest struct {
	Source        Destination            `json:"source"`
	Destination   Destination            `json:"destination"`
	Tariff        string                 `json:"tariff"`
	TotalWeight   int                    `json:"total_weight"`
	PaymentMethod string                 `json:"payment_method"`
	Places        []PricingResourcePlace `json:"places"`
}

type Destination struct {
	PlatformStationId string `json:"platform_station_id"`
	Address           string `json:"address"`
}

type PricingResourcePlace struct {
	PhysicalDims PhysicalDims `json:"physical_dims"`
}

type PhysicalDims struct {
	Dx          int `json:"dx"` // centimeters
	Dy          int `json:"dy"`
	Dz          int `json:"dz"`
	WeightGross int `json:"weight_gross"` // grams
}

type PriceResponse struct {
	PricingTotal string `json:"pricing_total"`
	DeliveryDays int    `json:"delivery_days"`
}

func (p PriceResponse) IntPrice() (int, error) {
	fl, er := strconv.ParseFloat(strings.Split(p.PricingTotal, " ")[0], 32)
	if er != nil {
		log.Println(er)
		return 0, er
	}
	return int(math.Ceil(fl)), nil
}

func NewPriceRequest(order db.Order) (*PriceRequest, error) {
	if order.Data.PvzId == "" {
		return nil, fmt.Errorf("PvzId can not be null")
	}
	priceRequest := new(PriceRequest)
	source := new(Destination)
	source.PlatformStationId = props.Get()["ya.delivery.source.pvz.id"].(string)
	priceRequest.Source = *source
	destination := new(Destination)
	destination.PlatformStationId = order.Data.PvzId
	destination.Address = order.Data.PvzAddress
	priceRequest.Destination = *destination
	priceRequest.Tariff = "time_interval"
	priceRequest.PaymentMethod = "already_paid"
	priceRequest.TotalWeight = 250
	place := new(PricingResourcePlace)
	dims := new(PhysicalDims)
	dims.Dx = 25
	dims.Dy = 15
	dims.Dz = 10
	dims.WeightGross = 250
	place.PhysicalDims = *dims
	priceRequest.Places = []PricingResourcePlace{*place}
	return priceRequest, nil
}
