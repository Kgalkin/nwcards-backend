package main

import (
	"NorthwindREST/src/go/models/db"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"log"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
)

func getItems(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	a, er := db.GetAllItems()
	if er != nil {
		fmt.Print(er)
	}
	resp, _ := json.Marshal(a)
	fmt.Fprintf(w, string(resp))
}

func enableCors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:9090")
	w.Header().Set("Content-Type", "application/json")
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

func createItem(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	enableCors(w)
	er := r.ParseMultipartForm(32 << 20) // limit your max input length!
	if er != nil {
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
	item := db.StoreItem{}
	er = json.Unmarshal([]byte(r.FormValue("data")), &item)
	if er != nil {
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
	files := r.MultipartForm.File["image"]
	for _, file := range files {
		f, er := file.Open()
		if er != nil {
			http.Error(w, er.Error(), http.StatusBadRequest)
			return
		}
		i, er := db.CreateItem(item)
		if er != nil {
			http.Error(w, er.Error(), http.StatusBadRequest)
			return
		}
		dirPath := fmt.Sprintf("./static/img/%1d", i.Id)
		os.MkdirAll(dirPath, os.ModePerm)
		fileName := fmt.Sprintf("%d_cover.%s", i.Id, strings.Split(file.Filename, ".")[1])
		osFile, _ := os.OpenFile(filepath.Join(dirPath, fileName), os.O_WRONLY|os.O_CREATE, 0666)
		defer osFile.Close()
		io.Copy(osFile, f)
	}
	a, er := db.GetAllItems()
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
	fmt.Fprintf(w, string(resp))
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]
	r.ParseMultipartForm(32 << 20) // limit your max input length!
	file, header, err := r.FormFile("image")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	name := strings.Split(header.Filename, ".")
	dirPath := fmt.Sprintf("./static/img/%s", id)
	os.MkdirAll(dirPath, os.ModePerm)
	fileName := fmt.Sprintf("%s_cover.%s", id, name[1])
	osFile, _ := os.OpenFile(filepath.Join(dirPath, fileName), os.O_WRONLY|os.O_CREATE, 0666)
	defer file.Close()
	io.Copy(osFile, file)
	return
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
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	router.HandleFunc("/api/items", getItems).Methods(http.MethodGet)
	router.HandleFunc("/api/items", createItem).Methods(http.MethodPost)
	router.HandleFunc("/api/createOrder", createOrder).Methods(http.MethodPost)
	router.HandleFunc("/api/items/{id}/uploadCoverImage", uploadFile).Methods(http.MethodPost)
	router.HandleFunc("/api/tags", createTag).Methods(http.MethodPost)
	router.HandleFunc("/api/tags", getTags).Methods(http.MethodGet)
	router.Handle("/", http.FileServer(http.Dir("./view/")))
	log.Fatal(http.ListenAndServe("192.168.0.101:8081", router))
}
