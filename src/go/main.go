package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"nwcards-backend/src/go/email"
	"nwcards-backend/src/go/imageprocessing"
	"nwcards-backend/src/go/models/db"
	"nwcards-backend/src/go/payments"
	"nwcards-backend/src/go/props"
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
		log.Println(er)
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
		log.Println(er)
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
		log.Println(er)
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
	resp, er := json.Marshal(menuItems)
	if er != nil {
		log.Println(er)
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
		log.Println(er)
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
	for _, item := range order.Data.Items {
		if er = db.UpdateItemCount(item); er != nil {
			log.Println(er)
			http.Error(w, er.Error(), http.StatusConflict)
			return
		}
	}
	created, er := db.CreateOrder(order)
	if er != nil {
		log.Println(er)
		http.Error(w, er.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "{\"uuid\": \"%s\"}", created.Uuid)
	text := `Здравствуйте! Вы сделали заказ в магазине North Wind Cards. Срок сборки и отправки заказа после оплаты 3-5 дней. Мы пришлем Вам трекер либо фото конверта, готового к отправке, если выбрана доставка простым письмом.<br><br>
Посмотреть и оплатить заказ вы можете по ссылке.<br><br>
Спасибо за заказ!<br><br>
Ева Северный Ветер`
	table, er := createTable(
		"Заказ №", strconv.Itoa(created.Id),
		"Ф.И.О", created.Data.Name,
		"Адрес", created.Data.Address,
		"Индекс", created.Data.Index,
		"Способ доставки", created.Data.DeliveryOption.Description,
		"Ссылка на заказ", generateOrderLinc(created.Uuid, "NorthwindCards"))
	if er == nil {
		er = email.Send(created.Data.Email, "Новый заказ в магазине NorthwindCards", text+table)
	}
	if er != nil {
		log.Printf("Error during sending CREATED email %s\n", er)
		error := db.Error{Code: 1, Message: fmt.Sprintln(er)}
		stringErr, err := json.Marshal(error)
		if err != nil {
			log.Println(err)
			return
		}
		err = db.UpdateOrder(order.Id, fmt.Sprintf("error = '%s'", string(stringErr)))
		if err != nil {
			log.Println(err)
		}
		return
	}
}

func generateOrderLinc(uuid string, text string) string {
	return fmt.Sprintf("<a href='http://%s/orders/%s'>%s</a>", props.Get()["site.host"].(string), uuid, text)
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
		log.Println(er)
		if strings.Contains(er.Error(), "Found 0") {
			http.Error(w, er.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, er.Error(), http.StatusInternalServerError)
		return
	}
	order.Data.Name = ""
	order.Data.Address = ""
	order.Data.Email = ""
	order.Data.Index = ""
	resp, er := json.Marshal(order)
	if er != nil {
		log.Println(er)
		http.Error(w, er.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(resp)
}

func getDeliveryOptions(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	enableCors(w)
	w.WriteHeader(200)
	deliveryOptions, er := db.GetDeliveryOptions()
	if er != nil {
		log.Println(er)
		http.Error(w, er.Error(), http.StatusInternalServerError)
		return
	}
	resp, er := json.Marshal(deliveryOptions)
	if er != nil {
		log.Println(er)
		http.Error(w, er.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(resp)
}

func requestPayment(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	params := mux.Vars(r)
	id := params["id"]
	if len(id) == 0 {
		er := fmt.Errorf("No {id} in path parameter: ")
		log.Println(er)
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
	intId, er := strconv.Atoi(id)
	if er != nil {
		log.Println(er)
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
	order, er := db.GetOrderById(intId)
	if er != nil {
		log.Println(er)
		http.Error(w, er.Error(), http.StatusNotFound)
		return
	}
	initRequest := payments.NewInitForm(*order)
	response, er := payments.RequestLink(*initRequest)
	if er != nil {
		log.Println(er)
		http.Error(w, er.Error(), http.StatusInternalServerError)
		return
	}
	if !response.Success {
		er = fmt.Errorf("Something went wrong with link request, order %d, errorCode %s\n", intId, response.ErrorCode)
		log.Println(er, response.Message, response.Details)
		http.Error(w, er.Error(), http.StatusInternalServerError)
		return
	}
	order.Data.PaymentLink = response.PaymentURL
	orderDataJson, er := json.Marshal(order.Data)
	if er != nil {
		log.Println(er)
		log.Println("Error saving payment details to DB, Token:  " + initRequest.Token +
			" payment_link: " + response.PaymentURL)
	}
	er = db.UpdateOrder(order.Id, fmt.Sprintf("token = '%s', data = '%s'",
		initRequest.Token, string(orderDataJson)))
	if er != nil {
		log.Println(er)
		log.Println("Error saving payment details to DB, Token:  " + initRequest.Token +
			" payment_link: " + response.PaymentURL)
	}
	w.Write([]byte(fmt.Sprintf("{\"link\": \"%s\"}", response.PaymentURL)))
}

//private
func createItem(w http.ResponseWriter, r *http.Request) {
	if !authChecker(w, r) {
		return
	}
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
	if !authChecker(w, r) {
		return
	}
	enableCors(w)
	params := mux.Vars(r)
	id := params["id"]
	if len(id) == 0 {
		log.Println("There should be 1 item id in request " + r.URL.Path)
		http.Error(w, "There should be 1 item id in request "+r.URL.Path, http.StatusBadRequest)
		return
	}
	idInt, er := strconv.ParseInt(id, 0, 64)
	if er != nil {
		log.Println(er)
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
	storeItem := db.StoreItem{}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	er = decoder.Decode(&storeItem)
	if er != nil {
		log.Println(er)
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
	if storeItem.Id != idInt {
		log.Println(er)
		http.Error(w, "Item id must be same as path id", http.StatusBadRequest)
		return
	}
	updated, er := db.UpdateItem(storeItem)
	if er != nil {
		log.Println(er)
		http.Error(w, er.Error(), http.StatusInternalServerError)
	}
	resp, _ := json.Marshal(updated)
	fmt.Fprintf(w, string(resp))
}

func createTag(w http.ResponseWriter, r *http.Request) {
	if !authChecker(w, r) {
		return
	}
	defer r.Body.Close()
	enableCors(w)
	tags := new([]string)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	er := decoder.Decode(&tags)
	if er != nil {
		log.Println(er)
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
	allTags, er := db.CreateTags(*tags)
	if er != nil {
		log.Println(er)
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
	resp, _ := json.Marshal(allTags)
	_, er = fmt.Fprintf(w, string(resp))
	if er != nil {
		log.Println(er)
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	if !authChecker(w, r) {
		return
	}
	params := mux.Vars(r)
	id := params["id"]
	err := r.ParseMultipartForm(32 << 20) // limit your max input length!
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()
	name := strings.Split(header.Filename, ".")
	err = makeDirAndSaveFile(file,
		filepath.Join("static", "img", id, fmt.Sprintf("%s_cover.%s", id, name[1])))
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getOrders(w http.ResponseWriter, r *http.Request) {
	if !authChecker(w, r) {
		return
	}
	enableCors(w)
	appJson(w)
	orders, err := db.GetOrders()
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp, err := json.Marshal(orders)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(resp)
}

func updateOrder(w http.ResponseWriter, r *http.Request) {
	if !authChecker(w, r) {
		return
	}
	enableCors(w)
	params := mux.Vars(r)
	id := params["id"]
	if len(id) < 0 {
		log.Println("There no {id} parameter in request " + r.URL.Path)
		http.Error(w, "There no {id} parameter in request "+r.URL.Path, http.StatusBadRequest)
		return
	}
	idInt, er := strconv.Atoi(id)
	if er != nil {
		log.Println(er)
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
	patch := make(map[string]string)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	er = decoder.Decode(&patch)
	if er != nil {
		log.Println(er)
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
	er = handlePatchOrder(idInt, patch)
	if er != nil {
		log.Println(er)
		http.Error(w, er.Error(), http.StatusInternalServerError)
		return
	}
}

func handlePatchOrder(id int, patch map[string]string) error {
	advance := patch["advance"]
	if len(advance) > 0 {
		switch advance {
		case "REVOKE":
			er := db.RevokeOrder(id)
			if er != nil {
				log.Println(er)
				return er
			}
		case "SET_DELIVERY_PRICE":
			price, er := strconv.Atoi(patch["deliveryPrice"])
			if er != nil {
				log.Println(er)
				return er
			}
			order, er := db.GetOrderById(id)
			if er != nil {
				log.Println(er)
				return er
			}
			order.Data.DeliveryOption.Price = price
			er = db.UpdateOrderStateData(id, "", &order.Data)
			if er != nil {
				log.Println(er)
				return er
			}
			er = email.Send(order.Data.Email, "Стоимость доставки заказа обновлена",
				generateOrderLinc(order.Uuid, "NorthwindCards"))
			if er != nil {
				log.Println(er)
			}
		case "SENT_CODE":
			postalCode := patch["postalCode"]
			if len(postalCode) == 0 {
				er := fmt.Errorf("SENT advance for order: %d but no postalCode found!\n", id)
				log.Println(er)
				return er
			}
			order, er := db.GetOrderById(id)
			if er != nil {
				log.Println(er)
				return er
			}
			if order.State != db.PAYMENT_RECEIVED {
				er = fmt.Errorf("SENT advance for order: %d but order state != %s\n", id, db.PAYMENT_RECEIVED)
				return er
			}
			order.Data.PostalCode = postalCode
			er = db.UpdateOrderStateData(id, db.SENT_TO_CUSTOMER, &order.Data)
			if er != nil {
				log.Println(er)
				return er
			}
			er = email.Send(order.Data.Email,
				fmt.Sprintf("ваш заказ отправлен из магазина %s", props.Get()["site.host"].(string)),
				fmt.Sprintf(
					`Здравствуйте! Ваш %[1]s из магазина North Wind Cards отправлен. 
Вы можете отслеживать отправку с помощью трекера на <a href="https://www.pochta.ru/tracking#%[2]s">сайте Почты России</a> или в приложении Почты России. 
Попутного ветра!
%[2]s`,
					generateOrderLinc(order.Uuid, "заказ"),
					postalCode))
		}
	}
	return nil
}

func login(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	if !authChecker(w, r) {
		return
	}
	w.WriteHeader(200)
}

//handlers
func authChecker(w http.ResponseWriter, r *http.Request) bool {
	login, pss, ok := r.BasicAuth()
	if !ok {
		log.Println("Can not get basic auth")
		http.Error(w, "Can not get basic auth", http.StatusUnauthorized)
		return false
	}
	if login == "" || pss == "" {
		log.Println("Login or password is empty")
		http.Error(w, "Login or password is empty", http.StatusUnauthorized)
		return false
	}
	prop := props.Get()
	if login == prop["admin.login"] &&
		pss == prop["admin.pss"] {
		return true
	}
	log.Println("Login or password not correct")
	http.Error(w, "Login or password not correct", http.StatusUnauthorized)
	return false
}

func enableCors(w http.ResponseWriter) {
	if props.Get()["api.cors.enabled"].(bool) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:9090")
		w.Header().Set("Access-Control-Allow-Methods", "PUT,POST,GET,DELETE,OPTIONS,PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "*")
	}
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

func handlePayment(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	w.WriteHeader(200)
	paymentResponse := payments.PaymentResponse{}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	er := decoder.Decode(&paymentResponse)
	str, _ := json.Marshal(paymentResponse)
	log.Println("Received payment response: " + string(str))
	if er != nil {
		log.Println(er)
		return
	}
	if paymentResponse.Success && paymentResponse.Status == "CONFIRMED" {
		orderId, er := strconv.Atoi(paymentResponse.OrderID)
		if er != nil {
			log.Println(er)
			return
		}
		er = db.UpdateOrderStateData(orderId, db.PAYMENT_RECEIVED, nil)
		if er != nil {
			log.Println(er)
		}
	}
}

func authMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.RemoteAddr, "79.120.11.119") ||
			strings.Contains(r.RemoteAddr, "127.0.0.1") {
			next.ServeHTTP(w, r)
		} else {
			http.Error(w, "No permissions", http.StatusUnauthorized)
		}
	}
	return http.HandlerFunc(fn)
}

func main() {
	go func() {
		for i := 0; i < 10; i++ {
			fmt.Println("time ticked")
			time.Sleep(5 * time.Second)
		}
	}()
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	router := mux.NewRouter()
	router.Use(authMiddleware)
	router.Use(func(next http.Handler) http.Handler { return handlers.LoggingHandler(os.Stdout, next) })
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	router.HandleFunc("/api/items", getItems).Methods(http.MethodGet)
	router.HandleFunc("/api/items", createItem).Methods(http.MethodPost)
	router.HandleFunc("/api/items/{id}", updateItem).Methods(http.MethodPatch)
	router.HandleFunc("/api/orders", createOrder).Methods(http.MethodPost)
	router.HandleFunc("/api/orders/{uuid}", viewOrderByUUID).Methods(http.MethodGet)
	router.HandleFunc("/api/orders/{id}", updateOrder).Methods(http.MethodPatch)
	router.HandleFunc("/api/orders/{id}/requestPayment", requestPayment).Methods(http.MethodGet)
	router.HandleFunc("/api/orders", getOrders).Methods(http.MethodGet)
	router.HandleFunc("/api/items/{id}/uploadCoverImage", uploadFile).Methods(http.MethodPost)
	router.HandleFunc("/api/tags", createTag).Methods(http.MethodPost)
	router.HandleFunc("/api/tags", getTags).Methods(http.MethodGet)
	router.HandleFunc("/api/menu", getMenu).Methods(http.MethodGet)
	router.HandleFunc("/api/login", login).Methods(http.MethodGet)
	router.HandleFunc("/api/deliveryOptions", getDeliveryOptions).Methods(http.MethodGet)
	router.HandleFunc("/api/handlepaymentresult", handlePayment)
	router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) { enableCors(w) }).Methods(http.MethodOptions)
	router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "./view/index.html") }).Methods(http.MethodGet)
	log.Fatal(http.ListenAndServe(props.Get()["api.host.address"].(string), router))
}
