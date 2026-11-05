package main

import (
	"github.com/nusantara-supermart/backend/internal/promotion"
	"github.com/nusantara-supermart/backend/pkg/bootstrap"
	"github.com/nusantara-supermart/backend/pkg/replica"
)

func main() {
	svc := bootstrap.New("promotion-service")

	repo := promotion.NewRepository(svc.DB)
	service := promotion.NewService(repo)
	promotion.RegisterRoutes(svc.API, promotion.NewHandler(service), svc.Cfg)

	catalogURL := bootstrap.Env("CATALOG_URL", "http://catalog-service:3000")
	svc.StartReplica([]replica.Source{
		{Table: "products", URL: catalogURL},
		{Table: "product_images", URL: catalogURL},
	})

	svc.Run()
}
