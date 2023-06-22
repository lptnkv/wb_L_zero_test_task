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

type GetOrderHandler struct{}

func (handler *GetOrderHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fmt.Println(vars)
	fmt.Fprintln(w, "Here will be Get method to get Order by id")
}

type OrderFormHandler struct{}

func (handler *OrderFormHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Here will be form")
}

type TestStanHandler struct{}

func (handler *TestStanHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Here will be publishing to stan")
}
