package main

import (
	"github.com/nusantara-supermart/backend/internal/logistics"
	"github.com/nusantara-supermart/backend/internal/messaging"
	"github.com/nusantara-supermart/backend/pkg/bootstrap"
	"github.com/nusantara-supermart/backend/pkg/internalclient"
	"github.com/nusantara-supermart/backend/pkg/replica"
)

func main() {
	svc := bootstrap.New("logistics-service")

	// Logistics -> Order (HTTP internal): status SHIPPED dan COMPLETED
	orderClient := internalclient.NewOrderClient(bootstrap.Env("ORDER_URL", "http://order-service:3000"))

	repo := logistics.NewRepository(svc.DB)
	service := logistics.NewService(repo, orderClient)
	logistics.RegisterRoutes(svc.API, logistics.NewHandler(service), svc.Cfg)

	orderURL := bootstrap.Env("ORDER_URL", "http://order-service:3000")
	authURL := bootstrap.Env("AUTH_URL", "http://auth-service:3000")
	svc.StartReplica([]replica.Source{
		{Table: "orders", URL: orderURL},
		{Table: "order_shipping_details", URL: orderURL},
		{Table: "users", URL: authURL},
		{Table: "user_addresses", URL: authURL},
	})

	// Consumer: buat pengiriman saat OrderPlaced
	svc.StartConsumer(messaging.QueueLogisticsShipment, "logistics-shipment", logistics.OrderPlacedHandler(svc.DB))

	svc.Run()
}
