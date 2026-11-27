package main

import (
	"github.com/nusantara-supermart/backend/internal/procurement"
	"github.com/nusantara-supermart/backend/pkg/bootstrap"
	"github.com/nusantara-supermart/backend/pkg/replica"
)

func main() {
	svc := bootstrap.New("procurement-service")

	repo := procurement.NewRepository(svc.DB)
	service := procurement.NewService(repo)
	procurement.RegisterRoutes(svc.API, procurement.NewHandler(service), svc.Cfg)

	svc.StartReplica([]replica.Source{
		{Table: "products", URL: bootstrap.Env("CATALOG_URL", "http://catalog-service:3000")},
		{Table: "warehouses", URL: bootstrap.Env("INVENTORY_URL", "http://inventory-service:3000")},
	})

	// Transactional outbox: event GoodsReceived dari CreateGRN
	svc.StartOutboxRelay()

	svc.Run()
}
