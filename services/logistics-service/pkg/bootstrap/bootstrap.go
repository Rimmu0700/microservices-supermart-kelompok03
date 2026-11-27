// Package bootstrap berisi kerangka yang sama untuk semua service:
// konfigurasi, koneksi database, Fiber, health check, consumer, outbox relay, dan graceful shutdown.
package bootstrap

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/jmoiron/sqlx"

	"github.com/nusantara-supermart/backend/internal/messaging"
	"github.com/nusantara-supermart/backend/pkg/config"
	"github.com/nusantara-supermart/backend/pkg/database"
	"github.com/nusantara-supermart/backend/pkg/middleware"
	"github.com/nusantara-supermart/backend/pkg/replica"
	"github.com/nusantara-supermart/backend/pkg/response"
)

// Service membungkus semua yang dibutuhkan satu proses microservice.
type Service struct {
	Name string
	Cfg  *config.Config
	DB   *sqlx.DB
	App  *fiber.App
	API  fiber.Router
	Ctx  context.Context

	stop context.CancelFunc
}

// Env membaca environment variable dengan nilai bawaan.
func Env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// New menyiapkan konfigurasi, database (dengan retry), Fiber, dan endpoint /api/v1/health.
func New(name string) *Service {
	cfg := config.LoadConfig()
	log.Printf("[%s] Memulai service (mode %s, database %s)...", name, cfg.AppEnv, cfg.MySQLDatabase)

	var db *sqlx.DB
	var err error
	for attempt := 1; attempt <= 30; attempt++ {
		db, err = database.ConnectMySQL(cfg)
		if err == nil {
			break
		}
		log.Printf("[%s] Menunggu MySQL (percobaan %d/30): %v", name, attempt, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("[%s] Gagal konek database: %v", name, err)
	}

	app := fiber.New(fiber.Config{AppName: name, ServerHeader: name})
	if os.Getenv("BENCH_MODE") != "1" {
		app.Use(logger.New())
	}
	app.Use(recover.New())
	app.Use(middleware.SetupCORS(cfg))

	api := app.Group("/api/v1")
	api.Get("/health", func(c fiber.Ctx) error {
		dbStatus := "healthy"
		if db.Ping() != nil {
			dbStatus = "unreachable"
		}
		return response.Success(c, fiber.StatusOK, name+" is operational", fiber.Map{
			"service":   name,
			"database":  dbStatus,
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	return &Service{Name: name, Cfg: cfg, DB: db, App: app, API: api, Ctx: ctx, stop: stop}
}

// ExportReplica membuka endpoint /internal/replica/:table untuk tabel milik service ini.
func (s *Service) ExportReplica(exports []replica.Export) {
	replica.RegisterExporter(s.App, s.DB, exports)
}

// StartReplica menyinkronkan tabel replika dari service pemilik secara berkala.
func (s *Service) StartReplica(sources []replica.Source) {
	seconds, err := strconv.Atoi(Env("REPLICA_INTERVAL_SECONDS", "5"))
	if err != nil || seconds <= 0 {
		seconds = 5
	}
	sy := &replica.Syncer{DB: s.DB, Sources: sources, Interval: time.Duration(seconds) * time.Second}
	go sy.Run(s.Ctx)
}

// StartConsumer menjalankan consumer RabbitMQ di goroutine, dengan koneksi ulang otomatis.
func (s *Service) StartConsumer(queue, tag string, handler messaging.Handler) {
	go func() {
		uri := messaging.AMQPURI()
		for s.Ctx.Err() == nil {
			if err := messaging.Consume(s.Ctx, uri, queue, tag, 10, handler); err != nil {
				log.Printf("[%s] %v. Mencoba lagi dalam 3 detik...", tag, err)
				select {
				case <-s.Ctx.Done():
					return
				case <-time.After(3 * time.Second):
				}
			}
		}
	}()
}

// StartOutboxRelay menerbitkan event dari tabel outbox_events ke RabbitMQ.
func (s *Service) StartOutboxRelay() {
	relay := &messaging.OutboxRelay{DB: s.DB, URI: messaging.AMQPURI(), Interval: time.Second, BatchSize: 50}
	go relay.Run(s.Ctx)
}

// Run menjalankan server HTTP sampai menerima SIGINT/SIGTERM.
func (s *Service) Run() {
	go func() {
		<-s.Ctx.Done()
		log.Printf("[%s] Mematikan server dengan rapi...", s.Name)
		_ = s.App.Shutdown()
	}()

	addr := fmt.Sprintf(":%s", s.Cfg.AppPort)
	log.Printf("[%s] Mendengarkan di http://0.0.0.0%s", s.Name, addr)
	if err := s.App.Listen(addr); err != nil {
		log.Printf("[%s] Server berhenti: %v", s.Name, err)
	}
	s.stop()
	_ = s.DB.Close()
}
