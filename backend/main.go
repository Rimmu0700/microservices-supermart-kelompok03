package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"

	"github.com/nusantara-supermart/backend/internal/auth"
	"github.com/nusantara-supermart/backend/internal/catalog"
	"github.com/nusantara-supermart/backend/internal/inventory"
	"github.com/nusantara-supermart/backend/internal/logistics"
	"github.com/nusantara-supermart/backend/internal/order"
	"github.com/nusantara-supermart/backend/internal/payment"
	"github.com/nusantara-supermart/backend/internal/procurement"
	"github.com/nusantara-supermart/backend/internal/promotion"
	"github.com/nusantara-supermart/backend/internal/support"
	"github.com/nusantara-supermart/backend/pkg/config"
	"github.com/nusantara-supermart/backend/pkg/database"
	"github.com/nusantara-supermart/backend/pkg/middleware"
	"github.com/nusantara-supermart/backend/pkg/response"
)

func main() {
	cfg := config.LoadConfig()

	log.Printf("Starting PT Nusantara SuperMart Monolith Backend in [%s] mode...", cfg.AppEnv)

	// Database Connection
	db, err := database.ConnectMySQL(cfg)
	if err != nil {
		log.Printf("WARNING: MySQL connection failed: %v", err)
		log.Println("Backend will retry connecting when serving requests or initializing...")
	} else {
		log.Println("Connected to MySQL database successfully!")
		defer db.Close()
	}

	app := fiber.New(fiber.Config{
		AppName:      "PT Nusantara SuperMart Indonesia API v1.0",
		ServerHeader: "NusantaraSuperMart-Monolith",
	})

	// Global Middlewares
	// Logger dimatikan saat benchmark agar REST tidak dibebani log per request.
	// Jalankan dengan BENCH_MODE=1 untuk mematikannya.
	if os.Getenv("BENCH_MODE") != "1" {
		app.Use(logger.New())
	}
	app.Use(recover.New())
	app.Use(middleware.SetupCORS(cfg))

	// Base API Group
	api := app.Group("/api/v1")

	// Healthcheck
	api.Get("/health", func(c fiber.Ctx) error {
		dbStatus := "healthy"
		if db == nil || db.Ping() != nil {
			dbStatus = "unreachable"
		}
		return response.Success(c, fiber.StatusOK, "Nusantara SuperMart Monolith API is operational", fiber.Map{
			"timestamp": time.Now().Format(time.RFC3339),
			"database":  dbStatus,
			"version":   "1.0.0",
		})
	})

	// Domain Repositories
	authRepo := auth.NewRepository(db)
	catalogRepo := catalog.NewRepository(db)
	inventoryRepo := inventory.NewRepository(db)
	orderRepo := order.NewRepository(db)
	paymentRepo := payment.NewRepository(db)
	promotionRepo := promotion.NewRepository(db)
	logisticsRepo := logistics.NewRepository(db)
	procurementRepo := procurement.NewRepository(db)
	supportRepo := support.NewRepository(db)

	// Domain Services (Monolith cross-domain coupling)
	authSvc := auth.NewService(authRepo, cfg)
	catalogSvc := catalog.NewService(catalogRepo)

	// gRPC Catalog (East-West Traffic) berjalan paralel dengan REST
	go func() {
		if err := catalog.ServeGRPC(":50051", catalogSvc); err != nil {
			log.Printf("gRPC server stopped: %v", err)
		}
	}()

	inventorySvc := inventory.NewService(inventoryRepo)
	orderSvc := order.NewService(orderRepo, catalogSvc, inventorySvc)
	paymentSvc := payment.NewService(paymentRepo, orderSvc)
	promotionSvc := promotion.NewService(promotionRepo)
	logisticsSvc := logistics.NewService(logisticsRepo, orderSvc)
	procurementSvc := procurement.NewService(procurementRepo)
	supportSvc := support.NewService(supportRepo)

	// Domain Handlers
	authHandler := auth.NewHandler(authSvc)
	catalogHandler := catalog.NewHandler(catalogSvc)
	inventoryHandler := inventory.NewHandler(inventorySvc)
	orderHandler := order.NewHandler(orderSvc)
	paymentHandler := payment.NewHandler(paymentSvc)
	promotionHandler := promotion.NewHandler(promotionSvc)
	logisticsHandler := logistics.NewHandler(logisticsSvc)
	procurementHandler := procurement.NewHandler(procurementSvc)
	supportHandler := support.NewHandler(supportSvc)

	// Register Domain Routers
	auth.RegisterRoutes(api, authHandler, cfg)
	catalog.RegisterRoutes(api, catalogHandler, cfg)
	inventory.RegisterRoutes(api, inventoryHandler, cfg)
	order.RegisterRoutes(api, orderHandler, cfg)
	payment.RegisterRoutes(api, paymentHandler, cfg)
	promotion.RegisterRoutes(api, promotionHandler, cfg)
	logistics.RegisterRoutes(api, logisticsHandler, cfg)
	procurement.RegisterRoutes(api, procurementHandler, cfg)
	support.RegisterRoutes(api, supportHandler, cfg)

	// Graceful Shutdown Setup
	serverShutdown := make(chan os.Signal, 1)
	signal.Notify(serverShutdown, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-serverShutdown
		log.Println("Shutting down Nusantara SuperMart server gracefully...")
		_ = app.Shutdown()
	}()

	addr := fmt.Sprintf(":%s", cfg.AppPort)
	log.Printf("Nusantara SuperMart server listening on http://0.0.0.0%s", addr)
	if err := app.Listen(addr); err != nil {
		log.Printf("Server closed: %v", err)
	}
}
