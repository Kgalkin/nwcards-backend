package service

import (
	"errors"
	"fmt"
	"log"
	"nwcards-backend/src/go/client/ya"
	"nwcards-backend/src/go/models/db"
	"strconv"
)

func AdjustOrder(order *db.Order) error {
	er := FillOrder(order)
	if er != nil {
		log.Println(er)
		return er
	}

	//bonuses, er := db.GetBonusesByIds([]string{""})
	bonuses, er := db.GetBonuses()

	if er != nil {
		log.Println(er)
		return er
	}

	activeBonuses := []*db.Bonus{}
	for _, bonus := range bonuses {
		if bonus.IsGlobal {
			activeBonuses = append(activeBonuses, bonus)
		}
	}

	if activeBonuses != nil {
		for _, bonus := range activeBonuses {
			if bonus.Type == db.ITEM_WRAPPER_TYPE {
				processItemWrapperBonus(order, bonus.Data)
			}
		}
	}

	deliveryOption, er := db.ResolveDeliveryOption(order.Data.DeliveryOption.Id)
	if er != nil {
		log.Println(er)
		return er
	}
	order.Data.DeliveryOption = *deliveryOption
	if deliveryOption.Id == 7 { //if yandex delivery
		pr, er := ya.NewPriceRequest(*order)
		if er != nil {
			log.Println(er)
			return er
		}
		priceResponse, er := ya.GetPrice(*pr)
		if er != nil {
			log.Println(er)
			return er
		}
		order.Data.DeliveryPrice, er = priceResponse.IntPrice()
		if er != nil {
			log.Println(er)
			return er
		}
	} else {
		order.Data.DeliveryPrice = deliveryOption.Price
	}
	order.ApplyDeliveryPriceModifiers()
	return nil
}

func processItemWrapperBonus(order *db.Order, bonus map[string]interface{}) {
	countToWrap := int(bonus["countToWrap"].(float64))
	bonusTags := anyToIntArray(bonus["tags"].([]any))
	newPriceOf := int(bonus["newPriceOf"].(float64))
	for i := 0; i < len(order.Data.Items); i++ {
		item := &order.Data.Items[i]
		if containsOneOf(item.Tags, bonusTags) && item.Count >= countToWrap {
			wrapperCount := item.Count / countToWrap
			item.Count = item.Count % countToWrap
			newItem := db.OrderItem{
				Id:    item.Id,
				Type:  db.ITEM_WRAPPER_TYPE,
				Count: wrapperCount,
				Price: newPriceOf * item.Price,
				Data: db.StoreItemData{
					Title: fmt.Sprintf("10 открыток: '%s'", item.Data.Title),
					Links: item.Data.Links},
				Props: map[string]any{"countToWrap": bonus["countToWrap"]}}
			if bonus["hexColor"] != nil {
				newItem.Props["hexColor"] = bonus["hexColor"]
			}
			order.Data.Items = append(order.Data.Items, newItem)
		}
	}
	noZeroCount := make([]db.OrderItem, 0)
	j := 0
	for i, item := range order.Data.Items {
		if item.Count == 0 {
			noZeroCount = append(noZeroCount, order.Data.Items[j:i]...)
			j = i + 1
		} else if i == (len(order.Data.Items) - 1) {
			noZeroCount = append(noZeroCount, order.Data.Items[j:]...)
		}
	}
	order.Data.Items = noZeroCount
}

func anyToIntArray(in []any) (out []int64) {
	out = make([]int64, 0, len(in))
	for _, v := range in {
		out = append(out, int64(v.(float64)))
	}
	return
}

func containsOneOf(slice []int64, otherSlice []int64) bool {
	for _, first := range slice {
		for _, another := range otherSlice {
			if first == another {
				return true
			}
		}
	}
	return false
}

func FillOrder(order *db.Order) error {
	ids := ""
	for i, item := range order.Data.Items {
		if i > 0 {
			ids += ","
		}
		ids += strconv.FormatInt(item.Id, 10)
	}
	items, er := db.GetItems(map[string][]string{"ids": {ids}, "size": {strconv.Itoa(len(order.Data.Items) + 1)}})
	if er != nil {
		log.Println(er)
		return er
	}
	for i, _ := range order.Data.Items {
		item := &order.Data.Items[i]
		for i, it := range items.Items {
			if it.Id == item.Id {
				if it.InStock < item.Count {
					message := fmt.Sprintf("Instock < count for item {id: %d,title: %s, count: %d, instock: %d}",
						item.Id, it.Data.Title, item.Count, it.InStock)
					log.Println(message)
					return errors.New(message)
				}
				item.Data = it.Data
				item.Price = it.Price
				item.Tags = it.Tags
				break
			}
			if i+1 == len(items.Items) {
				return errors.New(fmt.Sprintf("\nItem with ID: %d, not found in the database\n", item.Id))
			}
		}
	}

	return nil
}
