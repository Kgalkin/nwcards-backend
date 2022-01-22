package imageprocessing

import (
	"github.com/h2non/bimg"
	"os"
)

func Compress(originalPath string, quality int, newFilePath string) error {
	buffer, err := os.ReadFile(originalPath)
	if err != nil {
		return err
	}
	converted, err := bimg.NewImage(buffer).Convert(bimg.WEBP)
	if err != nil {
		return err
	}
	processed, err := bimg.NewImage(converted).Process(bimg.Options{Quality: quality})
	if err != nil {
		return err
	}
	/*rotated, err := bimg.NewImage(processed).Rotate(270)
	if err != nil {
		return err
	}*/
	err = bimg.Write(newFilePath, processed)
	if err != nil {
		return err
	}

	return nil
}
