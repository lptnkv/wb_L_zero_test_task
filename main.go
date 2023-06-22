package main

import (
	"context"
	"fmt"
	"lptnkv/orders/service"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	// Router and endpoints
	router := mux.NewRouter()
	router.Handle("/", &service.AddOrderHandler{})
	router.Handle("/order/{id}", &service.GetOrderHandler{})
	router.Handle("/form", &service.OrderFormHandler{})
	server := http.Server{
		Addr:    "127.0.0.1:8080",
		Handler: router,
	}

	// Запуска сервера
	go func() {
		fmt.Println("Starting server at ", server.Addr)
		err := server.ListenAndServe()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	signal.Notify(sigChan, os.Kill)

	sig := <-sigChan
	fmt.Println("Got signal: ", sig)

	// Ждем завершения операций до 30 секунд и выключаем сервер
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	server.Shutdown(ctx)
}
