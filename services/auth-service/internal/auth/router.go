package auth

import (
	"github.com/gofiber/fiber/v3"
	"github.com/nusantara-supermart/backend/pkg/config"
	"github.com/nusantara-supermart/backend/pkg/middleware"
)

func RegisterRoutes(router fiber.Router, handler *Handler, cfg *config.Config) {
	authGroup := router.Group("/auth")

	// Public routes
	authGroup.Post("/login", handler.Login)
	authGroup.Post("/register", handler.Register)

	// Authenticated routes
	authRequired := middleware.AuthRequired(cfg)
	authGroup.Get("/me", handler.GetMe, authRequired)
	authGroup.Put("/me", handler.UpdateProfile, authRequired)
	authGroup.Get("/addresses", handler.GetAddresses, authRequired)
	authGroup.Post("/addresses", handler.AddAddress, authRequired)
	authGroup.Delete("/addresses/:id", handler.DeleteAddress, authRequired)
	authGroup.Get("/kyc", handler.GetKYC, authRequired)
	authGroup.Post("/kyc", handler.SubmitKYC, authRequired)

	// Admin routes
	adminOnly := middleware.RolesRequired("SUPER_ADMIN")
	authGroup.Get("/users", handler.GetAllUsers, authRequired, adminOnly)
	authGroup.Post("/users", handler.CreateUser, authRequired, adminOnly)
	authGroup.Put("/users/:id", handler.UpdateUser, authRequired, adminOnly)
	authGroup.Put("/users/:id/status", handler.ToggleUserStatus, authRequired, adminOnly)
	authGroup.Delete("/users/:id", handler.DeleteUser, authRequired, adminOnly)
	authGroup.Get("/roles", handler.GetRoles, authRequired, adminOnly)
	authGroup.Get("/permissions", handler.GetPermissions, authRequired, adminOnly)
	authGroup.Put("/kyc/:id/verify", handler.VerifyKYC, authRequired, adminOnly)
}
