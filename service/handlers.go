package service

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

type AddOrderHandler struct{}

func (handler *AddOrderHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Here will be POST method to handle adding of order to database")
}

type GetOrderHandler struct {
	Cache map[OrderUID]Order
}

// Контроллер для получения заказа по uid
func (handler *GetOrderHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fmt.Println(vars)
	res, ok := handler.Cache[OrderUID(vars["id"])]
	if !ok {
		fmt.Fprintf(w, "Not found")
		return
	}
	fmt.Fprintf(w, "%+v\n", res)
}

type MockHandler struct{}

func (handler *MockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Here will be form")
}

type TestStanHandler struct{}

func (handler *TestStanHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Here will be publishing to stan")
}
