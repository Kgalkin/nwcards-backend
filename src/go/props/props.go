package props

import (
	"encoding/json"
	"log"
	"os"
)

func Get() map[string]interface{} {
	file, err := os.Open("./props.json")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	var data map[string]interface{}
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		log.Fatal(err)
	}
	return data
}
