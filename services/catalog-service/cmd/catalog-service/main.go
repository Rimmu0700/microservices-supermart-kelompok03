package main

import (
	"log"

	"github.com/nusantara-supermart/backend/internal/catalog"
	"github.com/nusantara-supermart/backend/pkg/bootstrap"
	"github.com/nusantara-supermart/backend/pkg/replica"
)

func main() {
	svc := bootstrap.New("catalog-service")

	repo := catalog.NewRepository(svc.DB)
	service := catalog.NewService(repo)
	catalog.RegisterRoutes(svc.API, catalog.NewHandler(service), svc.Cfg)

	// gRPC Catalog (East-West) untuk Order dan service lain
	go func() {
		if err := catalog.ServeGRPC(":50051", service); err != nil {
			log.Printf("[catalog-service] gRPC server berhenti: %v", err)
		}
	}()

	authURL := bootstrap.Env("AUTH_URL", "http://auth-service:3000")
	svc.StartReplica([]replica.Source{
		{Table: "users", URL: authURL}, // nama reviewer
	})
	svc.ExportReplica([]replica.Export{
		{Table: "products"},
		{Table: "product_images"},
	})

	svc.Run()
}
