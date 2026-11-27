package logistics

import (
	"github.com/gofiber/fiber/v3"
	"github.com/nusantara-supermart/backend/pkg/config"
	"github.com/nusantara-supermart/backend/pkg/middleware"
)

func RegisterRoutes(router fiber.Router, handler *Handler, cfg *config.Config) {
	logisticsGroup := router.Group("/logistics")

	// Public
	logisticsGroup.Get("/couriers", handler.GetCouriers)
	logisticsGroup.Get("/services", handler.GetServices)
	logisticsGroup.Get("/rates", handler.GetRates)

	// Authenticated
	authRequired := middleware.AuthRequired(cfg)
	logisticsGroup.Get("/shipments/:id", handler.GetShipmentByID, authRequired)
	logisticsGroup.Get("/order/:order_id", handler.GetShipmentByOrderID, authRequired)

	// Staff / Courier / Admin
	opsOrAdmin := middleware.RolesRequired("SUPER_ADMIN", "WAREHOUSE_STAFF", "COURIER")
	logisticsGroup.Get("/shipments", handler.GetShipments, authRequired, opsOrAdmin)
	logisticsGroup.Post("/shipments", handler.CreateShipment, authRequired, opsOrAdmin)
	logisticsGroup.Put("/shipments/:id/status", handler.UpdateStatus, authRequired, opsOrAdmin)
	logisticsGroup.Post("/shipments/:id/pod", handler.SubmitPOD, authRequired, opsOrAdmin)
	logisticsGroup.Get("/runs", handler.GetDeliveryRuns, authRequired, opsOrAdmin)
	logisticsGroup.Get("/drivers", handler.GetDrivers, authRequired, opsOrAdmin)
	logisticsGroup.Get("/vehicles", handler.GetVehicles, authRequired, opsOrAdmin)
}
