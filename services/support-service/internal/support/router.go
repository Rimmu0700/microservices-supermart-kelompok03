package support

import (
	"github.com/gofiber/fiber/v3"
	"github.com/nusantara-supermart/backend/pkg/config"
	"github.com/nusantara-supermart/backend/pkg/middleware"
)

func RegisterRoutes(router fiber.Router, handler *Handler, cfg *config.Config) {
	supportGroup := router.Group("/support")

	// Public
	supportGroup.Get("/faq", handler.GetFAQs)
	supportGroup.Get("/categories", handler.GetCategories)

	// Authenticated
	authRequired := middleware.AuthRequired(cfg)
	supportGroup.Get("/tickets", handler.GetTickets, authRequired)
	supportGroup.Get("/tickets/:id", handler.GetTicketByID, authRequired)
	supportGroup.Post("/tickets", handler.CreateTicket, authRequired)
	supportGroup.Post("/tickets/:id/messages", handler.AddMessage, authRequired)
	supportGroup.Post("/tickets/:id/rate", handler.RateTicket, authRequired)
	supportGroup.Get("/disputes", handler.GetDisputes, authRequired)
	supportGroup.Post("/disputes", handler.CreateDispute, authRequired)

	// Support Agent / Admin
	agentOrAdmin := middleware.RolesRequired("SUPER_ADMIN", "CS_AGENT")
	supportGroup.Put("/tickets/:id/status", handler.UpdateStatus, authRequired, agentOrAdmin)
	supportGroup.Put("/tickets/:id/assign", handler.AssignTicket, authRequired, agentOrAdmin)
	supportGroup.Post("/faq", handler.CreateFAQ, authRequired, agentOrAdmin)
}
