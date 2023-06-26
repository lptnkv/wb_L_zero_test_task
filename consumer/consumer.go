package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"lptnkv/orders/service"
	"os"
	"os/signal"

	stan "github.com/nats-io/stan.go"
)

func main() {
	sc, err := stan.Connect("test-cluster", "wb-l0-client2")
	if err != nil {
		log.Fatalf("Could not connect to cluster: %v", err)
	}
	log.Println("Connected to test-cluster")
	sub, err := sc.Subscribe("orders", func(m *stan.Msg) {
		received := DecodeToOrder(m.Data)
		fmt.Printf("Received a struct: %+v\n", received)
	})
	if err != nil {
		log.Fatalf("Could not subscribe to topic: %v", err)
	}
	log.Println("Subscribed to topic")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	signal.Notify(sigChan, os.Kill)

	sig := <-sigChan
	log.Println("Got signal: ", sig)
	sub.Unsubscribe()
	sc.Close()
}

func DecodeToOrder(s []byte) service.Order {

	o := service.Order{}
	dec := gob.NewDecoder(bytes.NewReader(s))
	err := dec.Decode(&o)
	if err != nil {
		log.Fatal(err)
	}
	return o
}
