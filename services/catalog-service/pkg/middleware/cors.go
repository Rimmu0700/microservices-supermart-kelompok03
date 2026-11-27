package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/nusantara-supermart/backend/pkg/config"
)

func SetupCORS(cfg *config.Config) fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins: cfg.CORSOrigins,
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowMethods: []string{"GET", "POST", "HEAD", "PUT", "DELETE", "PATCH", "OPTIONS"},
	})
}
