package catalog

import (
	"github.com/gofiber/fiber/v3"
	"github.com/nusantara-supermart/backend/pkg/config"
	"github.com/nusantara-supermart/backend/pkg/middleware"
)

func RegisterRoutes(router fiber.Router, handler *Handler, cfg *config.Config) {
	catalogGroup := router.Group("/catalog")

	// Public routes
	catalogGroup.Get("/products", handler.ListProducts)
	catalogGroup.Get("/products/:id", handler.GetProduct)
	catalogGroup.Get("/products/:id/reviews", handler.GetReviews)
	catalogGroup.Get("/categories", handler.GetCategories)
	catalogGroup.Get("/brands", handler.GetBrands)
	catalogGroup.Get("/products/:id/summary", handler.GetProductSummary)

	// Authenticated routes
	authRequired := middleware.AuthRequired(cfg)
	catalogGroup.Post("/products/:id/reviews", handler.AddReview, authRequired)

	// Admin routes
	adminOnly := middleware.RolesRequired("SUPER_ADMIN")
	catalogGroup.Post("/products", handler.CreateProduct, authRequired, adminOnly)
	catalogGroup.Put("/products/:id", handler.UpdateProduct, authRequired, adminOnly)
	catalogGroup.Delete("/products/:id", handler.DeleteProduct, authRequired, adminOnly)
	catalogGroup.Post("/categories", handler.CreateCategory, authRequired, adminOnly)
	catalogGroup.Post("/brands", handler.CreateBrand, authRequired, adminOnly)
}
