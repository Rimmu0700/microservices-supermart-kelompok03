package order

import (
	"github.com/gofiber/fiber/v3"
	"github.com/nusantara-supermart/backend/pkg/config"
	"github.com/nusantara-supermart/backend/pkg/middleware"
)

func RegisterRoutes(router fiber.Router, handler *Handler, cfg *config.Config) {
	orderGroup := router.Group("/orders")

	authRequired := middleware.AuthRequired(cfg)
	staffOrAdmin := middleware.RolesRequired("SUPER_ADMIN", "WAREHOUSE_STAFF")
	customerOnly := middleware.StrictRolesRequired("CUSTOMER")

	// Cart routes (Khusus Pelanggan)
	orderGroup.Get("/cart", handler.GetCart, authRequired, customerOnly)
	orderGroup.Post("/cart", handler.AddToCart, authRequired, customerOnly)
	orderGroup.Put("/cart/:id", handler.UpdateCartItem, authRequired, customerOnly)
	orderGroup.Delete("/cart/:id", handler.RemoveCartItem, authRequired, customerOnly)
	orderGroup.Post("/checkout", handler.Checkout, authRequired, customerOnly)

	// Order routes
	orderGroup.Get("/statuses", handler.GetStatuses, authRequired)
	orderGroup.Get("/", handler.GetOrders, authRequired)
	orderGroup.Get("/:id", handler.GetOrderByID, authRequired)
	orderGroup.Post("/:id/cancel", handler.CancelOrder, authRequired)
	orderGroup.Post("/:id/notes", handler.AddOrderNote, authRequired)

	// Staff/Admin status update
	orderGroup.Put("/:id/status", handler.UpdateOrderStatus, authRequired, staffOrAdmin)
}
