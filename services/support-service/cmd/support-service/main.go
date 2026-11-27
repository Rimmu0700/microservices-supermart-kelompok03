package main

import (
	"github.com/nusantara-supermart/backend/internal/support"
	"github.com/nusantara-supermart/backend/pkg/bootstrap"
	"github.com/nusantara-supermart/backend/pkg/replica"
)

func main() {
	svc := bootstrap.New("support-service")

	repo := support.NewRepository(svc.DB)
	service := support.NewService(repo)
	support.RegisterRoutes(svc.API, support.NewHandler(service), svc.Cfg)

	authURL := bootstrap.Env("AUTH_URL", "http://auth-service:3000")
	svc.StartReplica([]replica.Source{
		{Table: "users", URL: authURL},
		{Table: "roles", URL: authURL},
		{Table: "user_roles", URL: authURL},
		{Table: "orders", URL: bootstrap.Env("ORDER_URL", "http://order-service:3000")},
	})

	svc.Run()
}
