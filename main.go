package main

import (
	"context"
	"fmt"
	"log"
	"lptnkv/orders/service"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	stan "github.com/nats-io/stan.go"
)

var cache map[service.OrderUID]service.Order

func main() {
	// Router and endpoints
	router := mux.NewRouter()
	router.Handle("/", &service.AddOrderHandler{})
	router.Handle("/order/{id}", &service.GetOrderHandler{Cache: cache})
	router.Handle("/mock", &service.MockHandler{})
	server := http.Server{
		Addr:    "127.0.0.1:8080",
		Handler: router,
	}

	sc, _ := stan.Connect("test-cluster", "wb-l0-client")
	sub, _ := sc.Subscribe("orders", func(m *stan.Msg) {
		fmt.Println("Received a message: ", string(m.Data))
	})

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
	sub.Unsubscribe()
	sc.Close()
}

func init() {
	cache = make(map[service.OrderUID]service.Order)
	cache["absd"] = service.Order{Entry: "a"}
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Some error occured. Err: %s", err)
	}
	conn, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	queryOrderString := `select order_uid, track_number, entry, locale, 
	internal_signature, customer_id, delivery_service, shardkey, sm_id,
	date_created, oof_shard, payment_id, delivery_id, id from orders`
	rows, err := conn.Query(context.Background(), queryOrderString)
	if err != nil {
		log.Fatalf("Could not query rows: %+v\n", err)
	}
	for rows.Next() {
		var order service.Order
		var order_id int
		var payment_id int
		var delivery_id int

		err = rows.Scan(&order.OrderUID, &order.TrackNumber, &order.Entry, &order.Locale,
			&order.InternalSignature, &order.CustomerID, &order.DeliveryService,
			&order.Shardkey, &order.SmID, &order.DateCreated, &order.OofShard, &payment_id, &delivery_id, &order_id)
		if err != nil {
			log.Fatalf("Could not scan row: %+v\n", err)
		}

		queryPaymentString := fmt.Sprintf(`select transaction, request_id, currency, provider, amount, 
		payment_dt, bank, delivery_cost, goods_total, custom_fee from payment where id = %d`, payment_id)
		row := conn.QueryRow(context.Background(), queryPaymentString)
		err = row.Scan(&order.Payment.Transaction,
			&order.Payment.RequestID, &order.Payment.Currency, &order.Payment.Provider, &order.Payment.Amount,
			&order.Payment.PaymentDt, &order.Payment.Bank,
			&order.Payment.DeliveryCost,
			&order.Payment.GoodsTotal,
			&order.Payment.CustomFee)
		if err != nil {
			log.Fatalf("Could not scan payment row: %+v\n", err)
		}

		queryDeliveryString := fmt.Sprintf(`select name, phone, zip, city, address, region, email from delivery where id=%d`, delivery_id)
		row = conn.QueryRow(context.Background(), queryDeliveryString)
		err = row.Scan(&order.Delivery.Name, &order.Delivery.Phone,
			&order.Delivery.Zip, &order.Delivery.City, &order.Delivery.Address,
			&order.Delivery.Region, &order.Delivery.Email)
		if err != nil {
			log.Fatalf("Could not scan delivery row: %+v\n", err)
		}

		// Получаем количество товаров в заказе
		queryCountItems := fmt.Sprintf(`select count(*)
		from items 
		join order_item 
		on items.id = order_item.item_id
		join orders
		on orders.id=order_item.order_id 
		where orders.id = %d`, order_id)
		cnt := 0
		err = conn.QueryRow(context.Background(), queryCountItems).Scan(&cnt)
		if err != nil {
			log.Fatalf("Could not count items: %+v\n", err)
		}
		// Получаем сами товары
		queryItemsString := fmt.Sprintf(`select items.chrt_id, items.track_number, items.price, items.rid, items.name, items.sale, items.size, items.total_price, items.nm_id, items.brand, items.status
			from items 
			join order_item 
			on items.id = order_item.item_id
			join orders
			on orders.id=order_item.order_id 
			where orders.id = %d`, order_id)
		rows, err := conn.Query(context.Background(), queryItemsString)
		if err != nil {
			log.Fatalf("Could not query items: %+v\n", err)
		}
		// Выделяем память под товары
		order.Items = make([]service.Item, cnt)
		i := 0
		for rows.Next() {
			err = rows.Scan(&order.Items[i].ChrtID, &order.Items[i].TrackNumber,
				&order.Items[i].Price, &order.Items[i].Rid, &order.Items[i].Name,
				&order.Items[i].Sale, &order.Items[i].Size, &order.Items[i].TotalPrice,
				&order.Items[i].NmID, &order.Items[i].Brand, &order.Items[i].Status)
			if err != nil {
				log.Fatalf("Could not scan item: %+v\n", err)
			}
		}

		//fmt.Printf("%+v\n", order)

		// Добавляем в кэш
		cache[service.OrderUID(order.OrderUID)] = order
	}
}
