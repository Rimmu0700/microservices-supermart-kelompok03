package main

import (
	"github.com/nusantara-supermart/backend/internal/auth"
	"github.com/nusantara-supermart/backend/pkg/bootstrap"
	"github.com/nusantara-supermart/backend/pkg/replica"
)

func main() {
	svc := bootstrap.New("auth-service")

	repo := auth.NewRepository(svc.DB)
	service := auth.NewService(repo, svc.Cfg)
	auth.RegisterRoutes(svc.API, auth.NewHandler(service), svc.Cfg)

	// Data milik Auth yang dipakai service lain (password_hash TIDAK ikut diekspor)
	svc.ExportReplica([]replica.Export{
		{Table: "users", Exclude: []string{"password_hash"}},
		{Table: "user_addresses"},
		{Table: "roles"},
		{Table: "user_roles"},
	})

	svc.Run()
}
