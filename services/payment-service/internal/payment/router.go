package payment

import (
	"github.com/gofiber/fiber/v3"
	"github.com/nusantara-supermart/backend/pkg/config"
	"github.com/nusantara-supermart/backend/pkg/middleware"
)

func RegisterRoutes(router fiber.Router, handler *Handler, cfg *config.Config) {
	paymentGroup := router.Group("/payments")

	// Public
	paymentGroup.Get("/methods", handler.GetMethods)

	// Authenticated
	authRequired := middleware.AuthRequired(cfg)
	paymentGroup.Get("/invoices", handler.GetInvoices, authRequired)
	paymentGroup.Get("/invoices/:id", handler.GetInvoiceByID, authRequired)
	paymentGroup.Get("/invoices/order/:order_id", handler.GetInvoiceByOrderID, authRequired)
	paymentGroup.Post("/pay", handler.PayInvoice, authRequired)
	paymentGroup.Get("/credits", handler.GetStoreCredit, authRequired)
	paymentGroup.Post("/credits/topup", handler.TopupStoreCredit, authRequired)
	paymentGroup.Get("/refunds", handler.GetRefunds, authRequired)
	paymentGroup.Post("/refunds", handler.RequestRefund, authRequired)

	// Admin
	adminOnly := middleware.RolesRequired("SUPER_ADMIN")
	paymentGroup.Put("/refunds/:id/approve", handler.ApproveRefund, authRequired, adminOnly)
}
