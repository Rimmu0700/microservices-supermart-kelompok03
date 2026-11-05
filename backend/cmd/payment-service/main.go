package main

import (
	"github.com/nusantara-supermart/backend/internal/messaging"
	"github.com/nusantara-supermart/backend/internal/payment"
	"github.com/nusantara-supermart/backend/pkg/bootstrap"
	"github.com/nusantara-supermart/backend/pkg/internalclient"
	"github.com/nusantara-supermart/backend/pkg/replica"
)

func main() {
	svc := bootstrap.New("payment-service")

	// Payment -> Order (HTTP internal): ubah status order menjadi PROCESSING setelah lunas
	orderClient := internalclient.NewOrderClient(bootstrap.Env("ORDER_URL", "http://order-service:3000"))

	repo := payment.NewRepository(svc.DB)
	service := payment.NewService(repo, orderClient)
	payment.RegisterRoutes(svc.API, payment.NewHandler(service), svc.Cfg)

	orderURL := bootstrap.Env("ORDER_URL", "http://order-service:3000")
	authURL := bootstrap.Env("AUTH_URL", "http://auth-service:3000")
	svc.StartReplica([]replica.Source{
		{Table: "orders", URL: orderURL},
		{Table: "users", URL: authURL},
	})

	// Consumer: buat faktur saat OrderPlaced
	svc.StartConsumer(messaging.QueuePaymentInvoice, "payment-invoice", payment.OrderPlacedHandler(svc.DB))

	svc.Run()
}
