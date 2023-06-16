package main

import (
	"fmt"
	"lptnkv/orders/service"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	router := mux.NewRouter()
	router.Handle("/", &service.AddOrderHandler{})
	router.Handle("/order/{id}", &service.GetOrderHandler{})
	router.Handle("/form", &service.OrderFormHandler{})
	http.Handle("/", router)
	err := http.ListenAndServe("127.0.0.1:8080", router)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Listening at 127.0.0.1:8080")
}
