package imageprocessing

import (
	"fmt"
	"github.com/h2non/bimg"
	"io"
	"log"
	"math/rand"
	"nwcards-backend/src/go/models/db"
	"nwcards-backend/src/go/utils"
	"os"
)

func CreateImagesForItem(originalImage io.Reader, item *db.StoreItem, originalExtension string) error {
	filePathOriginal, filePathShort, filePathPreview := CoverImageLinks(item.Id, originalExtension)
	er := utils.MakeDirAndSaveFile(originalImage, filePathOriginal)
	if er != nil {
		log.Println(er.Error())
		return er
	}
	item.Data.Links.Original = filePathOriginal
	item.Data.Links.Short = filePathShort
	item.Data.Links.Preview = filePathPreview
	er = CompressFileToWebp(filePathOriginal, 10, filePathShort)
	if er != nil {
		log.Println(er.Error())
		return er
	}
	er = CompressFileToType(filePathOriginal, 60, filePathPreview, bimg.JPEG)
	_, er = db.UpdateItem(*item)
	if er != nil {
		log.Println(er.Error())
		return er
	}
	return nil
}

func CoverImageLinks(id int64, extension string) (string, string, string) {
	random := rand.Intn(10000)
	return fmt.Sprintf("./static/img/%[1]d/%[1]d_original_%[3]d.%[2]s", id, extension, random),
		fmt.Sprintf("./static/img/%[1]d/%[1]d_short_%[2]d.webp", id, random),
		fmt.Sprintf("./static/img/%[1]d/%[1]d_preview_%[2]d.jpeg", id, random)
}

func CompressFileToWebp(originalPath string, quality int, newFilePath string) error {
	return CompressFileToType(originalPath, quality, newFilePath, bimg.WEBP)
}

func CompressFileToType(originalPath string, quality int, newFilePath string, t bimg.ImageType) error {
	buffer, err := os.ReadFile(originalPath)
	if err != nil {
		log.Println(err)
		return err
	}
	return CompressToType(buffer, quality, newFilePath, t)
}

func Compress(buffer []byte, quality int, newFilePath string) error {
	return CompressToType(buffer, quality, newFilePath, bimg.WEBP)
}
func CompressToType(buffer []byte, quality int, newFilePath string, t bimg.ImageType) error {
	converted, err := bimg.NewImage(buffer).Convert(t)
	if err != nil {
		log.Println(err)
		return err
	}
	processed, err := bimg.NewImage(converted).Process(bimg.Options{Quality: quality})
	if err != nil {
		log.Println(err)
		return err
	}
	/*rotated, err := bimg.NewImage(processed).Rotate(270)
	if err != nil {
		return err
	}*/
	err = bimg.Write(newFilePath, processed)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}
