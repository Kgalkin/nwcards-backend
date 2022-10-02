package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/h2non/bimg"
	"io/ioutil"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"nwcards-backend/src/go/email"
	"nwcards-backend/src/go/imageprocessing"
	"nwcards-backend/src/go/migrations"
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
	for _, it := range a.Items {
		it.Data.Links.Original = ""
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
	order.Data.Email = strings.TrimSpace(order.Data.Email)
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
		"Ссылка на заказ", generateOrderLink(created.Uuid, "NorthwindCards"))
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

func checkSimplePost(order db.Order) error {
	/*if order.Data.DeliveryOption.Id == 1 {
		tags := ""
		for _, item := range order.Data.Items {

		}
		items, err := db.GetItems(map[string][]string{"tags": {}})
	}*/
	return nil
}

func generateOrderLink(uuid string, text string) string {
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
		log.Println(er.Error())
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
	item := db.StoreItem{}
	er = json.Unmarshal([]byte(r.FormValue("data")), &item)
	if er != nil {
		log.Println(er.Error())
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
	files := r.MultipartForm.File["image"]
	var items []db.StoreItem
	for _, file := range files {
		i, er := db.CreateItem(item)
		if er != nil {
			log.Println(er.Error())
			http.Error(w, er.Error(), http.StatusBadRequest)
			return
		}
		er = saveCoverImage(file, i)
		if er != nil {
			log.Println(er.Error())
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
	idInt, er := parseId(r)
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
	updated, er := db.UpdateItemWithoutLinks(storeItem)
	if er != nil {
		log.Println(er)
		http.Error(w, er.Error(), http.StatusInternalServerError)
	}
	resp, _ := json.Marshal(updated)
	fmt.Fprintf(w, string(resp))
}

func updateCoverImage(w http.ResponseWriter, r *http.Request) {
	if !authChecker(w, r) {
		return
	}
	defer r.Body.Close()
	enableCors(w)
	er := r.ParseMultipartForm(32 << 20) // limit your max input length!
	if er != nil {
		log.Println(er)
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
	id, er := parseId(r)
	if er != nil {
		log.Println(er)
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
	item, er := db.GetItem(id)
	if er != nil {
		log.Println(er)
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
	files := r.MultipartForm.File["image"]
	er = os.Remove(item.Data.Links.Original)
	if er != nil {
		log.Println(er)
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
	er = os.Remove(item.Data.Links.Short)
	if er != nil {
		log.Println(er)
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
	er = saveCoverImage(files[0], item)
	if er != nil {
		log.Println(er)
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
	resp, er := json.Marshal(item)
	if er != nil {
		log.Println(er)
		http.Error(w, er.Error(), http.StatusBadRequest)
		return
	}
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
	er = handlePatchOrder(idInt, r)
	if er != nil {
		log.Println(er)
		http.Error(w, er.Error(), http.StatusInternalServerError)
		return
	}
}

func handlePatchOrder(id int, r *http.Request) error {
	er := r.ParseMultipartForm(32 << 20) // limit your max input length!
	if er != nil {
		log.Println(er)
		return er
	}
	patch := make(map[string]string)
	er = json.Unmarshal([]byte(r.FormValue("data")), &patch)
	if er != nil {
		log.Println(er)
		return er
	}
	/*patch := make(map[string]string)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	er = decoder.Decode(&patch)*/
	if er != nil {
		log.Println(er)
		return er
	}
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
				generateOrderLink(order.Uuid, "NorthwindCards"))
			if er != nil {
				log.Println(er)
				return er
			}
		case "READY_FOR_SHIPPING":
			er := db.UpdateOrder(id, fmt.Sprintf("state = '%s'", db.READY_FOR_SHIPPING))
			if er != nil {
				log.Println(er)
				return er
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
			if order.State != db.READY_FOR_SHIPPING {
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
					`Здравствуйте! Ваш %[1]s из магазина North Wind Cards отправлен.<br/>
Вы можете отслеживать отправку с помощью трекера на <a href="https://www.pochta.ru/tracking#%[2]s">сайте Почты России</a> или в приложении Почты России.<br/> 
Попутного ветра!<br/>
%[2]s<br/><br/>

Будем благодарны за оставленный отзыв &#128144;<br/>
<a href="https://vk.com/topic-182017496_46570052">Оставить отзыв</a>`,
					generateOrderLink(order.Uuid, "заказ"),
					postalCode))
		case "SENT_IMAGE":
			order, er := db.GetOrderById(id)
			if er != nil {
				log.Println(er)
				return er
			}
			if order.State != db.READY_FOR_SHIPPING {
				er = fmt.Errorf("SENT advance for order: %d but order state != %s\n", id, db.PAYMENT_RECEIVED)
				return er
			}
			files := r.MultipartForm.File["image"]
			file, er := files[0].Open()
			if er != nil {
				log.Println(er)
				return er
			}
			defer file.Close()
			buff, er := ioutil.ReadAll(file)
			if er != nil {
				log.Println(er)
				return er
			}
			imageLink := fmt.Sprintf("./orders/%[1]d/img/%[1]d_proof_%[2]d.jpeg",
				id, rand.Intn(10000))
			dir := filepath.Dir(imageLink)
			er = os.MkdirAll(dir, os.ModePerm)
			if er != nil {
				log.Println(er)
				return er
			}
			er = imageprocessing.CompressToType(buff, 10, imageLink, bimg.JPEG)
			if er != nil {
				log.Println(er)
				return er
			}
			order.Data.Links.Proof = imageLink
			er = db.UpdateOrderStateData(id, db.SENT_TO_CUSTOMER, &order.Data)
			if er != nil {
				log.Println(er)
				return er
			}
			er = email.SendWithFile(order.Data.Email,
				fmt.Sprintf("ваш заказ отправлен из магазина %s", props.Get()["site.host"].(string)),
				`Ваш заказ отправлен простым писмом, в приложении к письму вы найдете фото-подтверждение.<br/><br/>
Будем благодарны за оставленный отзыв &#128144;<br/>
<a href="https://vk.com/topic-182017496_46570052">Оставить отзыв</a>`,
				imageLink)
			if er != nil {
				log.Println(er)
				return er
			}
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
func parseId(r *http.Request) (int64, error) {
	params := mux.Vars(r)
	id := params["id"]
	if len(id) == 0 {
		er := fmt.Errorf("There should be 1 item id in request %s\n", r.URL.Path)
		log.Println(er)
		return 0, er
	}
	idInt, er := strconv.ParseInt(id, 0, 64)
	if er != nil {
		log.Println(er)
		return 0, er
	}
	return idInt, nil
}

func saveCoverImage(file *multipart.FileHeader, item *db.StoreItem) error {
	f, er := file.Open()
	if er != nil {
		log.Println(er)
		return er
	}
	defer f.Close()
	return imageprocessing.CreateImagesForItem(f, item, strings.Split(file.Filename, ".")[1])
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
		order, er := db.GetOrderById(orderId)
		if er != nil {
			log.Println(er)
			return
		}
		if order.State != db.SENT_TO_CUSTOMER {
			er = db.UpdateOrderStateData(orderId, db.PAYMENT_RECEIVED, nil)
			if er != nil {
				log.Println(er)
			}
		}
		w.Write([]byte("OK"))
	}
}

func authMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.RemoteAddr, "79.120.11.119") ||
			strings.Contains(r.RemoteAddr, "127.0.0.1") ||
			strings.Contains(r.URL.Path, "/api/handlepaymentresult") {
			next.ServeHTTP(w, r)
		} else {
			http.Error(w, "No permissions", http.StatusUnauthorized)
		}
	}
	return http.HandlerFunc(fn)
}

func main() {
	migrations.RestoreOriginalLinks()
	go func() {
		for i := 0; i < 10; i++ {
			fmt.Println("time ticked")
			time.Sleep(5 * time.Second)
		}
	}()
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	router := mux.NewRouter()
	//router.Use(authMiddleware)
	router.Use(func(next http.Handler) http.Handler { return handlers.LoggingHandler(os.Stdout, next) })
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	router.HandleFunc("/api/items", getItems).Methods(http.MethodGet)
	router.HandleFunc("/api/items", createItem).Methods(http.MethodPost)
	router.HandleFunc("/api/items/{id}", updateItem).Methods(http.MethodPatch)
	router.HandleFunc("/api/items/{id}/image/cover", updateCoverImage).Methods(http.MethodPatch)
	router.HandleFunc("/api/orders", createOrder).Methods(http.MethodPost)
	router.HandleFunc("/api/orders/{uuid}", viewOrderByUUID).Methods(http.MethodGet)
	router.HandleFunc("/api/orders/{id}", updateOrder).Methods(http.MethodPatch)
	router.HandleFunc("/api/orders/{id}/requestPayment", requestPayment).Methods(http.MethodGet)
	router.HandleFunc("/api/orders", getOrders).Methods(http.MethodGet)
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
