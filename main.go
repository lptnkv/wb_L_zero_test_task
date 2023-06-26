package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"log"
	"lptnkv/orders/service"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	stan "github.com/nats-io/stan.go"
)

var cache map[service.OrderUID]service.Order

func main() {
	// Router and endpoints
	router := mux.NewRouter()
	router.Handle("/", &service.IndexHandler{Cache: cache})
	router.Handle("/order/{id}", &service.GetOrderHandler{Cache: cache})
	router.Handle("/mock", &service.MockHandler{})
	server := http.Server{
		Addr:    "127.0.0.1:8080",
		Handler: router,
	}

	sc, err := stan.Connect("test-cluster", "wb-l0-client2")
	if err != nil {
		log.Fatalf("Could not connect to nats cluster: %+v", err)
	}
	log.Println("Connected to cluster")
	sub, err := sc.Subscribe("orders", func(m *stan.Msg) {
		log.Println("Received a message from topic")
		received, err := DecodeToOrder(m.Data)
		if err != nil {
			log.Printf("Could not decode received data: %+v\n", err)
			return
		}
		err = AddToDatabase(received)
		if err != nil {
			log.Printf("Could not add to database: %+v\n", err)
			return
		}
		log.Println("Added new order to database")
	})
	if err != nil {
		log.Fatalf("Could not subscribe to topic: %+v", err)
	}
	log.Println("Subscribed to topic")

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
	log.Println("Unsubscribed from topic")
	sc.Close()
	log.Println("Closed connection to cluster")
}

func init() {
	cache = make(map[service.OrderUID]service.Order)
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Some error occured loading env: %s", err)
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

func DecodeToOrder(s []byte) (service.Order, error) {

	o := service.Order{}
	dec := gob.NewDecoder(bytes.NewReader(s))
	err := dec.Decode(&o)
	log.Println(o)
	if err != nil {
		return o, err
	}
	return o, nil
}

func AddToDatabase(order service.Order) error {
	err := godotenv.Load(".env")
	if err != nil {
		log.Printf("Some error occured loading env: %s\n", err)
		return err
	}
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	defer conn.Close(context.Background())
	if err != nil {
		log.Printf("Unable to connect to database: %v\n", err)
		return err
	}
	// Начинаем транзакцию
	tx, err := conn.BeginTx(context.Background(), pgx.TxOptions{})
	if err != nil {
		return err
	}
	tx.Begin(context.Background())
	// Добавляем товары и сохраняем их id для вставки в таблицу order_item
	itemIds := make([]int, len(order.Items))
	for i, val := range order.Items {
		var itemId int
		insertItemQuery := `INSERT INTO public.items
		(chrt_id, track_number, price, rid, "name", sale, "size", total_price, nm_id, brand, status)
		VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING id;
		`
		row := conn.QueryRow(context.Background(), insertItemQuery, val.ChrtID, val.TrackNumber,
			val.Price, val.Rid, val.Name, val.Sale, val.Size, val.TotalPrice, val.NmID, val.Brand, val.Status)
		err := row.Scan(&itemId)
		if err != nil {
			tx.Rollback(context.Background())
			log.Println("error adding items")
			return err
		}
		itemIds[i] = itemId
	}
	// Добавляем информацию о платеже
	var paymentId int
	insertPaymentQuery := `INSERT INTO payment
	("transaction", request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
	VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id;
	`
	row := tx.QueryRow(context.Background(), insertPaymentQuery, order.Payment.Transaction,
		order.Payment.RequestID, order.Payment.Currency, order.Payment.Provider, order.Payment.Amount, order.Payment.PaymentDt,
		order.Payment.Bank, order.Payment.DeliveryCost, order.Payment.GoodsTotal, order.Payment.CustomFee)
	err = row.Scan(&paymentId)
	if err != nil {
		log.Println("error adding payment")
		tx.Rollback(context.Background())
		return err
	}
	// Добавляем информацию о доставке
	var deliveryId int
	insertDeliveryQuery := `INSERT INTO delivery
	("name", phone, zip, city, address, region, email)
	VALUES($1, $2, $3, $4, $5, $6, $7) RETURNING id;
	`
	row = tx.QueryRow(context.Background(), insertDeliveryQuery, order.Delivery.Name, order.Delivery.Phone,
		order.Delivery.Zip, order.Delivery.City, order.Delivery.Address, order.Delivery.Region, order.Delivery.Email)
	err = row.Scan(&deliveryId)
	if err != nil {
		log.Println("error adding delivery")
		tx.Rollback(context.Background())
		return err
	}
	insertOrderQuery := `INSERT INTO orders
	(order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard, payment_id, delivery_id)
	VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) RETURNING id;`
	row = tx.QueryRow(context.Background(), insertOrderQuery, order.OrderUID, order.TrackNumber, order.Entry, order.Locale, order.InternalSignature,
		order.CustomerID, order.DeliveryService, order.Shardkey, order.SmID, order.DateCreated, order.OofShard, paymentId, deliveryId)
	var orderId int
	row.Scan(&orderId)
	if err != nil {
		log.Println("error adding order")
		tx.Rollback(context.Background())
		return err
	}
	for _, val := range itemIds {
		insertItemOrderQuery := `INSERT INTO public.order_item
		(order_id, item_id)
		VALUES($1, $2);
		`
		_, err := tx.Exec(context.Background(), insertItemOrderQuery, orderId, val)
		if err != nil {
			log.Println("error adding order_item")
			tx.Rollback(context.Background())
			return err
		}
	}
	tx.Commit(context.Background())
	cache[service.OrderUID(order.OrderUID)] = order
	return nil
}
