package main

import (
	"log"

	"github.com/gofiber/fiber/v3"

	"github.com/nusantara-supermart/backend/internal/messaging"
	"github.com/nusantara-supermart/backend/internal/order"
	"github.com/nusantara-supermart/backend/pkg/bootstrap"
	"github.com/nusantara-supermart/backend/pkg/internalclient"
	"github.com/nusantara-supermart/backend/pkg/replica"
	"github.com/nusantara-supermart/backend/pkg/response"
)

func main() {
	svc := bootstrap.New("order-service")

	// Order -> Catalog lewat gRPC; Order -> Inventory lewat HTTP internal
	catalogClient, closeCatalog, err := order.NewGRPCCatalogClient(bootstrap.Env("CATALOG_GRPC_ADDR", "catalog-service:50051"))
	if err != nil {
		log.Fatalf("[order-service] gagal membuat klien gRPC Catalog: %v", err)
	}
	defer closeCatalog()
	stockClient := internalclient.NewInventoryClient(bootstrap.Env("INVENTORY_URL", "http://inventory-service:3000"))

	repo := order.NewRepository(svc.DB)
	service := order.NewService(repo, catalogClient, stockClient)
	order.RegisterRoutes(svc.API, order.NewHandler(service), svc.Cfg)

	// Endpoint internal: dipanggil Payment dan Logistics untuk mengubah status order (tidak lewat gateway)
	svc.App.Put("/internal/orders/:id/status", func(c fiber.Ctx) error {
		var body struct {
			Status string `json:"status"`
			Notes  string `json:"notes"`
		}
		if err := c.Bind().Body(&body); err != nil {
			return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
		}
		if err := service.UpdateStatus(c.Params("id"), body.Status, body.Notes); err != nil {
			return response.Error(c, fiber.StatusBadRequest, "Failed to update order status", err.Error())
		}
		return response.Success(c, fiber.StatusOK, "Order status updated", nil)
	}, replica.RequireToken())

	// Data milik Order yang dipakai Payment, Logistics, dan Support
	svc.ExportReplica([]replica.Export{
		{Table: "orders"},
		{Table: "order_shipping_details"},
	})

	// Data referensi milik service lain
	catalogURL := bootstrap.Env("CATALOG_URL", "http://catalog-service:3000")
	authURL := bootstrap.Env("AUTH_URL", "http://auth-service:3000")
	inventoryURL := bootstrap.Env("INVENTORY_URL", "http://inventory-service:3000")
	svc.StartReplica([]replica.Source{
		{Table: "products", URL: catalogURL},
		{Table: "product_images", URL: catalogURL},
		{Table: "users", URL: authURL},
		{Table: "user_addresses", URL: authURL},
		{Table: "warehouses", URL: inventoryURL},
	})

	// Transactional outbox (OrderPlaced) + consumer kompensasi (StockReservationFailed)
	svc.StartOutboxRelay()
	svc.StartConsumer(messaging.QueueOrderCompensation, "order-compensation", order.CompensationHandler(svc.DB, service))

	svc.Run()
}
