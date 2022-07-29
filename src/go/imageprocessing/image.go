package imageprocessing

import (
	"github.com/h2non/bimg"
	"log"
	"os"
)

func CompressFile(originalPath string, quality int, newFilePath string) error {
	buffer, err := os.ReadFile(originalPath)
	if err != nil {
		log.Println(err)
		return err
	}
	return Compress(buffer, quality, newFilePath)
}

func Compress(buffer []byte, quality int, newFilePath string) error {
	converted, err := bimg.NewImage(buffer).Convert(bimg.WEBP)
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
