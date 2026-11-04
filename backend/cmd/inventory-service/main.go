package main

import (
	"github.com/gofiber/fiber/v3"

	"github.com/nusantara-supermart/backend/internal/inventory"
	"github.com/nusantara-supermart/backend/internal/messaging"
	"github.com/nusantara-supermart/backend/pkg/bootstrap"
	"github.com/nusantara-supermart/backend/pkg/replica"
	"github.com/nusantara-supermart/backend/pkg/response"
)

func main() {
	svc := bootstrap.New("inventory-service")

	repo := inventory.NewRepository(svc.DB)
	service := inventory.NewService(repo)
	inventory.RegisterRoutes(svc.API, inventory.NewHandler(service), svc.Cfg)

	// Endpoint internal: dipanggil Order saat pesanan dibatalkan (tidak lewat gateway)
	svc.App.Post("/internal/stock/release/:orderId", func(c fiber.Ctx) error {
		if err := service.ReleaseStock(c.Params("orderId")); err != nil {
			return response.Error(c, fiber.StatusInternalServerError, "Failed to release stock", err.Error())
		}
		return response.Success(c, fiber.StatusOK, "Stock released", nil)
	}, replica.RequireToken())

	svc.ExportReplica([]replica.Export{{Table: "warehouses"}})
	svc.StartReplica([]replica.Source{
		{Table: "products", URL: bootstrap.Env("CATALOG_URL", "http://catalog-service:3000")},
		{Table: "users", URL: bootstrap.Env("AUTH_URL", "http://auth-service:3000")},
	})

	// Consumer: reservasi stok (OrderPlaced) dan penambahan stok (GoodsReceived)
	svc.StartConsumer(messaging.QueueInventoryReservation, "inventory-reservation", inventory.ReservationHandler(svc.DB, messaging.AMQPURI()))
	svc.StartConsumer(messaging.QueueInventoryGoodsIn, "inventory-goods-received", inventory.GoodsReceivedHandler(svc.DB))

	svc.Run()
}
