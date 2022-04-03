package payments

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func RequestLink(form InitRequest) (*InitResponse, error) {
	bs, er := json.Marshal(form)
	if er != nil {
		log.Println(er)
		return nil, er
	}
	post, er := http.Post("https://securepay.tinkoff.ru/v2/Init", "application/json", bytes.NewBuffer(bs))
	if er != nil {
		log.Println(er)
		return nil, er
	}
	if post.StatusCode != 200 {
		er = fmt.Errorf("Paument request for order %d returned code %d\n", form.OrderId, post.StatusCode)
		log.Println(er)
		return nil, er
	}
	initResponse := InitResponse{}
	decoder := json.NewDecoder(post.Body)
	decoder.DisallowUnknownFields()
	er = decoder.Decode(&initResponse)
	if er != nil {
		log.Println(er)
		return nil, er
	}
	return &initResponse, nil
}
