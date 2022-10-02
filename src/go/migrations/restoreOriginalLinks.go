package migrations

import (
	"fmt"
	"io/ioutil"
	"log"
	"nwcards-backend/src/go/models/db"
	"strings"
)

func RestoreOriginalLinks() error {
	items, er := db.GetItems(map[string][]string{"size": {"1000"}})
	if er != nil {
		log.Println(er)
		return er
	}
	for _, it := range items.Items {
		files, er := ioutil.ReadDir(fmt.Sprintf("./static/img/%[1]d/", it.Id))
		if er != nil {
			log.Println(er)
		}

		for _, file := range files {
			if strings.Contains(file.Name(), "original") {
				it.Data.Links.Original = fmt.Sprintf("./static/img/%[1]d/%s", it.Id, file.Name())
				fmt.Println(it.Data.Links.Original)
				_, er = db.UpdateItem(*it)
				if er != nil {
					log.Fatal(er)
				}
				break
			}
		}
	}
	return nil
}
