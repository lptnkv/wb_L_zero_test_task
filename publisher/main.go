package main

import (
	"bytes"
	"encoding/gob"
	"flag"
	"log"
	"lptnkv/orders/service"

	"github.com/go-faker/faker/v4"
	stan "github.com/nats-io/stan.go"
)

func main() {
	isCorruptedData := flag.Bool("corrupt", false, "Choose to publish fake order or corrupted data")
	flag.Parse()

	sc, err := stan.Connect("test-cluster", "wb-l0-client1")
	if err != nil {
		log.Fatalf("Could not connect to cluster: %+v\n", err)
	}

	if *isCorruptedData {
		log.Println("Publishing wrong data")
		err = sc.Publish("orders", []byte("wrong data"))
		if err != nil {
			log.Fatalf("Could not publish to topic: %v", err)
		}
		sc.Close()
		return
	}

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

	log.Println("Publishing fake order")
	err = sc.Publish("orders", EncodeToBytes(fakeOrder))
	if err != nil {
		log.Fatalf("Could not publish to topic: %v", err)
	}
	sc.Close()
}

func EncodeToBytes(p interface{}) []byte {

	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(p)
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Println("uncompressed size (bytes): ", len(buf.Bytes()))
	return buf.Bytes()
}
