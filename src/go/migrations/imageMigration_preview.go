package migrations

import (
	"fmt"
	"github.com/h2non/bimg"
	"log"
	"nwcards-backend/src/go/imageprocessing"
	"nwcards-backend/src/go/models/db"
	"os"
)

func Migrate_images_to_preview() error {
	items, er := db.GetItems(map[string][]string{"size": {"1000"}})
	if er != nil {
		log.Println(er)
		return er
	}

	for _, it := range items.Items {
		buffer, err := os.ReadFile(it.Data.Links.Original)
		if err != nil {
			log.Println(err)
			return err
		}
		err = os.Remove(it.Data.Links.Short)
		if err != nil {
			log.Println(err)
			return err
		}

		_, _, preview := imageprocessing.CoverImageLinks(it.Id, "")
		err = imageprocessing.CompressToType(buffer, 10, it.Data.Links.Short, bimg.WEBP)
		if err != nil {
			log.Println(err)
			return err
		}
		err = imageprocessing.CompressToType(buffer, 50, preview, bimg.JPEG)
		if err != nil {
			log.Println(err)
			return err
		}
		it.Data.Links.Preview = preview
		_, err = db.UpdateItem(*it)
		if err != nil {
			log.Println(err)
			return err
		}
		log.Print(fmt.Sprintf("%d done", it.Id))
	}
	return nil
}
