package procurement

import (
	"github.com/gofiber/fiber/v3"
	"github.com/nusantara-supermart/backend/pkg/config"
	"github.com/nusantara-supermart/backend/pkg/middleware"
)

func RegisterRoutes(router fiber.Router, handler *Handler, cfg *config.Config) {
	procGroup := router.Group("/procurement")

	authRequired := middleware.AuthRequired(cfg)
	staffOrAdmin := middleware.RolesRequired("SUPER_ADMIN", "WAREHOUSE_STAFF")

	procGroup.Get("/suppliers", handler.GetSuppliers, authRequired, staffOrAdmin)
	procGroup.Post("/suppliers", handler.CreateSupplier, authRequired, staffOrAdmin)
	procGroup.Get("/purchase-orders", handler.GetPurchaseOrders, authRequired, staffOrAdmin)
	procGroup.Get("/purchase-orders/:id", handler.GetPurchaseOrderByID, authRequired, staffOrAdmin)
	procGroup.Post("/purchase-orders", handler.CreatePurchaseOrder, authRequired, staffOrAdmin)
	procGroup.Post("/purchase-orders/:id/approve", handler.ApprovePO, authRequired, staffOrAdmin)
	procGroup.Get("/grn", handler.GetGRNs, authRequired, staffOrAdmin)
	procGroup.Post("/grn", handler.CreateGRN, authRequired, staffOrAdmin)
	procGroup.Get("/invoices", handler.GetInvoices, authRequired, staffOrAdmin)
}
