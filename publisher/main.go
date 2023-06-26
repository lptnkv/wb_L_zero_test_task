package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"lptnkv/orders/service"

	"github.com/go-faker/faker/v4"
	stan "github.com/nats-io/stan.go"
)

func main() {
	sc, _ := stan.Connect("test-cluster", "wb-l0-client")

	var fakeOrder service.Order
	faker.FakeData(&fakeOrder)
	fakeOrder.Payment.Currency = "USD"
	fakeOrder.Payment.Provider = "wbpay"
	fakeOrder.Payment.Transaction = fakeOrder.Payment.Transaction + "test"
	fakeOrder.OrderUID = fakeOrder.OrderUID + "test"
	fakeOrder.Items = fakeOrder.Items[:1]
	fakeOrder.Entry = "WBIL"
	fakeOrder.Delivery.Zip = faker.GetRealAddress().PostalCode
	fakeOrder.Delivery.City = faker.GetRealAddress().City
	fakeOrder.Delivery.Address = faker.GetRealAddress().Address
	fakeOrder.Delivery.Region = faker.GetRealAddress().State
	fakeOrder.Payment.Transaction = fakeOrder.OrderUID
	fmt.Printf("%+v\n", fakeOrder)
	log.Println("Publishing fake order")
	err := sc.Publish("orders", EncodeToBytes(fakeOrder))
	if err != nil {
		log.Fatalf("Could not publish to topic: %v", err)
	}
}

func EncodeToBytes(p interface{}) []byte {

	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(p)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("uncompressed size (bytes): ", len(buf.Bytes()))
	return buf.Bytes()
}
