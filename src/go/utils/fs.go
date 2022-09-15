package utils

import (
	"io"
	"log"
	"os"
	"path/filepath"
)

func MakeDirAndSaveFile(file io.Reader, path string) error {
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		log.Println(err)
		return err
	}
	osFile, _ := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0666)
	defer osFile.Close()
	_, err = io.Copy(osFile, file)
	if err != nil {
		return err
	}
	return nil
}
