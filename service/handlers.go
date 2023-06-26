package service

import (
	"fmt"
	"html/template"
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

// Контроллер для получения заказа по uid в формате json
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

type IndexHandler struct {
	Cache map[OrderUID]Order
}

// Контроллер для получения заказа по uid
func (handler *IndexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, "Method not allowed")
		return
	}
	tmpl := template.Must(template.ParseFiles("templates/index.html"))
	uid := r.FormValue("uid")
	var data struct {
		Success bool
		Order   Order
		Message string
	}
	data.Message = "Введите корректный uid"
	order, ok := handler.Cache[OrderUID(uid)]
	if ok {
		data.Order = order
		data.Success = true
		data.Message = "Заказ найден"
	} else {
		data.Success = false
		data.Message = "Заказ не найден"
	}
	tmpl.Execute(w, data)
}

type MockHandler struct{}

func (handler *MockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Here will be form")
}

type TestStanHandler struct{}

func (handler *TestStanHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Here will be publishing to stan")
}
