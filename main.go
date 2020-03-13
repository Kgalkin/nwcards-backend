package main

import (
	"NorthwindREST/db"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

func homeLink(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome home!")
}

func getItems(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	a, er := db.AllItems()
	if er != nil {
		fmt.Print(er)
	}
	resp, _ := json.Marshal(a)
	fmt.Fprintf(w, string(resp))
}

func enableCors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8080")
}

func createOrder(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	order := db.Order{}
	er := json.NewDecoder(r.Body).Decode(&order)
	if er != nil {
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
	for _, item := range order.Data.Items {
		if er = db.UpdateItemCount(item); er != nil {
			http.Error(w, er.Error(), http.StatusInternalServerError)
		}
	}
	if er = db.CreateOrder(order); er != nil {
		http.Error(w, er.Error(), http.StatusInternalServerError)
	}
	fmt.Fprintf(w, "Order created")
}

func main() {
	db.InitDB("user=postgres password=N0coments dbname=northwindstoredb sslmode=disable")

	router := mux.NewRouter()
	router.Handle("/", http.FileServer(http.Dir("./view/")))
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	router.HandleFunc("/api/getItems", getItems).Methods(http.MethodGet)
	router.HandleFunc("/api/createOrder", createOrder).Methods(http.MethodPost)
	log.Fatal(http.ListenAndServe(":8001", router))
}
