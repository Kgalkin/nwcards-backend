package main

import (
	"NorthwindREST/src/go/models/db"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"io"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
)

//public routes
func getItems(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	a, er := db.GetAllItems()
	if er != nil {
		fmt.Print(er)
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

func createOrder(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
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
	if er = db.CreateOrder(order); er != nil {
		http.Error(w, er.Error(), http.StatusInternalServerError)
	}
	fmt.Fprintf(w, "Order created")
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
		filePath := fmt.Sprintf("./static/img/%[1]d/%[1]d_cover.%[2]s", i.Id, strings.Split(file.Filename, ".")[1])
		er = makeDirAndSaveFile(f, filePath)
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

func editItem(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	storeItem := db.StoreItem{}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	er := decoder.Decode(&storeItem)
	if er != nil {
		http.Error(w, er.Error(), http.StatusBadRequest)
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

//handlers
func enableCors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:9090")
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

func send(body string) {
	from := "a@gmail.com"
	pass := "12345"
	to := "foobarbazz@mailinator.com"

	msg := "From: " + from + "\n" +
		"To: " + to + "\n" +
		"Subject: Hello there\n\n" +
		body

	err := smtp.SendMail("smtp.gmail.com:587",
		smtp.PlainAuth("", from, pass, "smtp.gmail.com"),
		from, []string{to}, []byte(msg))

	if err != nil {
		log.Printf("smtp error: %s", err)
		return
	}

	log.Print("sent, visit http://foobarbazz.mailinator.com")
}

func sendEmail(recipient, text string) error {
	from := mail.Address{"", "galkin_kirill@mail.ru"}
	to := mail.Address{"", recipient}
	subj := "This is the email subject"
	body := "This is an example body.\n" + text

	// Setup headers
	headers := make(map[string]string)
	headers["From"] = from.String()
	headers["To"] = to.String()
	headers["Subject"] = subj

	// Setup message
	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	// Connect to the SMTP Server
	servername := "smtp.mail.ru:465"

	host, _, _ := net.SplitHostPort(servername)

	auth := smtp.PlainAuth("", "galkin_kirill@mail.ru", "stop", host)

	// TLS config
	tlsconfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         host,
	}

	// Here is the key, you need to call tls.Dial instead of smtp.Dial
	// for smtp servers running on 465 that require an ssl connection
	// from the very beginning (no starttls)
	conn, err := tls.Dial("tcp", servername, tlsconfig)
	if err != nil {
		log.Panic(err)
	}

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		log.Panic(err)
	}

	defer c.Quit()

	// Auth
	if err = c.Auth(auth); err != nil {
		log.Panic(err)
	}

	// To && From
	if err = c.Mail(from.Address); err != nil {
		log.Panic(err)
	}

	if err = c.Rcpt(to.Address); err != nil {
		log.Panic(err)
	}

	// Data
	w, err := c.Data()
	if err != nil {
		log.Panic(err)
	}

	_, err = w.Write([]byte(message))
	if err != nil {
		log.Panic(err)
	}

	err = w.Close()
	if err != nil {
		log.Panic(err)
	}

	return nil
}

func main() {
	//sendEmail("galkin_kirill@mail.ru", "hello")
	db.InitDB("user=postgres password=N0coments dbname=northwindstoredb sslmode=disable")
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	router := mux.NewRouter()
	router.Use(func(next http.Handler) http.Handler { return handlers.LoggingHandler(os.Stdout, next) })
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	router.HandleFunc("/api/items", getItems).Methods(http.MethodGet)
	router.HandleFunc("/api/items", createItem).Methods(http.MethodPost)
	router.HandleFunc("/api/orders", createOrder).Methods(http.MethodPost)
	router.HandleFunc("/api/items/{id}/uploadCoverImage", uploadFile).Methods(http.MethodPost)
	router.HandleFunc("/api/tags", createTag).Methods(http.MethodPost)
	router.HandleFunc("/api/tags", getTags).Methods(http.MethodGet)
	router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "./view/index.html") })
	log.Fatal(http.ListenAndServe("localhost:8080", router))
}
