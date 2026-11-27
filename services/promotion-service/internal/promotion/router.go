package promotion

import (
	"github.com/gofiber/fiber/v3"
	"github.com/nusantara-supermart/backend/pkg/config"
	"github.com/nusantara-supermart/backend/pkg/middleware"
)

func RegisterRoutes(router fiber.Router, handler *Handler, cfg *config.Config) {
	promoGroup := router.Group("/promotions")

	// Public
	promoGroup.Get("/", handler.GetPromotions)
	promoGroup.Get("/vouchers", handler.GetVouchers)
	promoGroup.Get("/flash-sales", handler.GetFlashSales)

	// Authenticated
	authRequired := middleware.AuthRequired(cfg)
	promoGroup.Post("/vouchers/validate", handler.ValidateVoucher, authRequired)
	promoGroup.Get("/loyalty", handler.GetLoyalty, authRequired)
	promoGroup.Post("/loyalty/redeem", handler.RedeemPoints, authRequired)

	// Admin
	adminOnly := middleware.RolesRequired("SUPER_ADMIN")
	promoGroup.Post("/", handler.CreatePromotion, authRequired, adminOnly)
	promoGroup.Post("/vouchers", handler.CreateVoucher, authRequired, adminOnly)
}
