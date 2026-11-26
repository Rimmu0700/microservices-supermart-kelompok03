package inventory

import (
	"github.com/gofiber/fiber/v3"
	"github.com/nusantara-supermart/backend/pkg/config"
	"github.com/nusantara-supermart/backend/pkg/middleware"
)

func RegisterRoutes(router fiber.Router, handler *Handler, cfg *config.Config) {
	inventoryGroup := router.Group("/inventory")

	authRequired := middleware.AuthRequired(cfg)
	staffOrAdmin := middleware.RolesRequired("SUPER_ADMIN", "WAREHOUSE_STAFF")

	inventoryGroup.Get("/warehouses", handler.GetWarehouses, authRequired)
	inventoryGroup.Get("/warehouses/:id/zones", handler.GetWarehouseZones, authRequired)

	inventoryGroup.Get("/stocks", handler.GetStocks, authRequired, staffOrAdmin)
	inventoryGroup.Put("/stocks/:id/adjust", handler.AdjustStock, authRequired, staffOrAdmin)
	inventoryGroup.Post("/mutations", handler.CreateMutation, authRequired, staffOrAdmin)
	inventoryGroup.Get("/mutations", handler.GetMutations, authRequired, staffOrAdmin)
	inventoryGroup.Get("/alerts", handler.GetLowStockAlerts, authRequired, staffOrAdmin)
	inventoryGroup.Get("/opnames", handler.GetOpnames, authRequired, staffOrAdmin)
	inventoryGroup.Post("/opnames", handler.CreateOpname, authRequired, staffOrAdmin)

	adminOnly := middleware.RolesRequired("SUPER_ADMIN")
	inventoryGroup.Post("/warehouses", handler.CreateWarehouse, authRequired, adminOnly)
}
