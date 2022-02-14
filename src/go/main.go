package main

import (
	"NorthwindREST/src/go/email"
	"NorthwindREST/src/go/imageprocessing"
	"NorthwindREST/src/go/models/db"
	"NorthwindREST/src/go/props"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

//public routes
func getItems(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	a, er := db.GetItems(r.URL.Query())
	if er != nil {
		http.Error(w, er.Error(), http.StatusInternalServerError)
		return
	}
	resp, _ := json.Marshal(a)
	fmt.Fprintf(w, string(resp))
}

func getTags(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	enableCors(w)
	allTags, er := db.GetTags()
	if er != nil {
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
	resp, _ := json.Marshal(allTags)
	fmt.Fprintf(w, string(resp))
}

func getMenu(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	menuItems, er := db.GetMenuItems()
	if er != nil {
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
	resp, er := json.Marshal(menuItems)
	if er != nil {
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
	fmt.Fprintf(w, string(resp))
}

func createOrder(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	appJson(w)
	order := db.Order{}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	er := decoder.Decode(&order)
	if er != nil {
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
	for _, item := range order.Data.Items {
		if er = db.UpdateItemCount(item); er != nil {
			http.Error(w, er.Error(), http.StatusConflict)
		}
	}
	created, er := db.CreateOrder(order)
	if er != nil {
		http.Error(w, er.Error(), http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "{\"uuid\": \"%s\"}", created.Uuid)
	table, er := createTable(
		"Заказ №", strconv.Itoa(created.Id),
		"Ф.И.О", created.Data.Name,
		"Адрес", created.Data.Address,
		"Индекс", created.Data.Index,
		"Способ доставки", created.Data.DeliveryOption.Description,
		"Ссылка на заказ", fmt.Sprintf("<a href='http://%s/orders/%s'>NorthwindCards</a>", props.Get()["site.host"], created.Uuid))
	if er != nil {
		log.Println(er)
		return
	}
	er = email.Send(created.Data.Email, "Новый заказ в магазине NorthwindCards", table)
	if er != nil {
		log.Printf("Error during sending created email %s\n", er.Error())
		return
	}
	created.State = db.CREATED_EMAIL_SENT
	_ = db.UpdateOrder(*created)
}

func createTable(args ...string) (string, error) {
	if len(args)%2 != 0 {
		return "", fmt.Errorf("Can not create table provided not paired args\n")
	}
	table := ""
	for i := 0; i < len(args); i += 2 {
		table += fmt.Sprintf("<tr><td>%s</td><td>%s</td></tr>", args[i], args[i+1])
	}
	return fmt.Sprintf("<table width=\"600\" style=\"border:1px solid #333\">%s</table>", table), nil
}

func viewOrderByUUID(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	appJson(w)
	params := mux.Vars(r)
	id := params["uuid"]
	order, er := db.GetOrderByUUID(id)
	if er != nil {
		http.Error(w, er.Error(), http.StatusInternalServerError)
		return
	}
	order.Data.Name = ""
	order.Data.Address = ""
	order.Data.Email = ""
	order.Data.Index = ""
	resp, er := json.Marshal(order)
	if er != nil {
		http.Error(w, er.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(resp)
}

//private
func createItem(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	enableCors(w)
	r.Header.Get("content-type")
	er := r.ParseMultipartForm(32 << 20) // limit your max input length!
	if er != nil {
		fmt.Println(er.Error())
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
	item := db.StoreItem{}
	er = json.Unmarshal([]byte(r.FormValue("data")), &item)
	if er != nil {
		fmt.Println(er.Error())
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
	files := r.MultipartForm.File["image"]
	var items []db.StoreItem
	for _, file := range files {
		f, er := file.Open()
		if er != nil {
			fmt.Println(er.Error())
			http.Error(w, er.Error(), http.StatusBadRequest)
			return
		}
		i, er := db.CreateItem(item)
		if er != nil {
			fmt.Println(er.Error())
			http.Error(w, er.Error(), http.StatusBadRequest)
			return
		}
		filePathOriginal := fmt.Sprintf("./static/img/%[1]d/%[1]d_original.%[2]s", i.Id, strings.Split(file.Filename, ".")[1])
		filePathShort := fmt.Sprintf("./static/img/%[1]d/%[1]d_short.webp", i.Id, strings.Split(file.Filename, ".")[1])
		er = makeDirAndSaveFile(f, filePathOriginal)
		if er != nil {
			fmt.Println(er.Error())
			http.Error(w, er.Error(), http.StatusBadRequest)
			return
		}
		i.Data.Links.Original = filePathOriginal
		i.Data.Links.Short = filePathShort
		er = imageprocessing.Compress(filePathOriginal, 40, filePathShort)
		if er != nil {
			fmt.Println(er.Error())
			http.Error(w, er.Error(), http.StatusBadRequest)
			return
		}
		_, er = db.UpdateItem(*i)
		if er != nil {
			fmt.Println(er.Error())
			http.Error(w, er.Error(), http.StatusBadRequest)
			return
		}
		items = append(items, *i)
	}
	resp, _ := json.Marshal(items)
	fmt.Fprintf(w, string(resp))
}

func updateItem(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	params := mux.Vars(r)
	id := params["id"]
	if len(id) != 1 {
		http.Error(w, "There should be 1 item id in request "+r.URL.Path, http.StatusBadRequest)
		return
	}
	idInt, er := strconv.ParseInt(id, 0, 64)
	if er != nil {
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
	storeItem := db.StoreItem{}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	er = decoder.Decode(&storeItem)
	if er != nil {
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
	if storeItem.Id != idInt {
		http.Error(w, "Item id must be same as path id", http.StatusBadRequest)
		return
	}
	updated, er := db.UpdateItem(storeItem)
	if er != nil {
		http.Error(w, er.Error(), http.StatusInternalServerError)
	}
	resp, _ := json.Marshal(updated)
	fmt.Fprintf(w, string(resp))
}

func createTag(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	enableCors(w)
	tags := new([]string)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	er := decoder.Decode(&tags)
	if er != nil {
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
	allTags, er := db.CreateTags(*tags)
	if er != nil {
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
	resp, _ := json.Marshal(allTags)
	_, er = fmt.Fprintf(w, string(resp))
	if er != nil {
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]
	err := r.ParseMultipartForm(32 << 20) // limit your max input length!
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()
	name := strings.Split(header.Filename, ".")
	err = makeDirAndSaveFile(file,
		filepath.Join("static", "img", id, fmt.Sprintf("%s_cover.%s", id, name[1])))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getOrders(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	appJson(w)
	orders, err := db.GetOrders()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp, err := json.Marshal(orders)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(resp)
}

//handlers
func enableCors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:9090")
	w.Header().Set("Access-Control-Allow-Methods", "PUT,POST,GET,DELETE,OPTIONS,PATCH")
	appJson(w)
}

func appJson(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
}

//functions
func makeDirAndSaveFile(file multipart.File, path string) error {
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
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

func sha256(str string) {
	hasher := sha1.New()
	hasher.Write([]byte(str))
	hex.EncodeToString(hasher.Sum(nil))
}

func main() {
	/*email.Send("galkin_kirill@mail.ru", "test",
	"<table><tr><td>Hello world</td>"+fmt.Sprintf("<td><a href='http://%s/orders/%s'>NorthwindCards</a></td></tr></table>", props.Get()["site.host"], "1"))*/
	go func() {
		for i := 0; i < 10; i++ {
			fmt.Println("time ticked")
			time.Sleep(5 * time.Second)
		}
	}()
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	router := mux.NewRouter()
	router.Use(func(next http.Handler) http.Handler { return handlers.LoggingHandler(os.Stdout, next) })
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	router.HandleFunc("/api/items", getItems).Methods(http.MethodGet)
	router.HandleFunc("/api/items", createItem).Methods(http.MethodPost)
	router.HandleFunc("/api/items/{id}", updateItem).Methods(http.MethodPatch)
	router.HandleFunc("/api/orders", createOrder).Methods(http.MethodPost)
	router.HandleFunc("/api/orders/{uuid}", viewOrderByUUID).Methods(http.MethodGet)
	router.HandleFunc("/api/orders", getOrders).Methods(http.MethodGet)
	router.HandleFunc("/api/items/{id}/uploadCoverImage", uploadFile).Methods(http.MethodPost)
	router.HandleFunc("/api/tags", createTag).Methods(http.MethodPost)
	router.HandleFunc("/api/tags", getTags).Methods(http.MethodGet)
	router.HandleFunc("/api/menu", getMenu).Methods(http.MethodGet)
	router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) { enableCors(w) }).Methods(http.MethodOptions)
	router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "./view/index.html") }).Methods(http.MethodGet)
	log.Fatal(http.ListenAndServe("localhost:8080", router))
}
