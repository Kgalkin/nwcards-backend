package main

import (
	"NorthwindREST/model"
	"fmt"
	"net/http"
)

func homeLink(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome home!")
}

func test(w http.ResponseWriter, r *http.Request){
	fmt.Fprintf(w, "Hello again")
}

func main() {
	model.InitDB("user=postgres password=N0coments dbname=northwindstoredb sslmode=disable")
	fmt.Println(model.AllItems())
/*	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", homeLink).Methods("GET")
	router.HandleFunc("/hello", test).Methods("GET")
	log.Fatal(http.ListenAndServe(":8080", router))*/
}