package support

import (
	"github.com/gofiber/fiber/v3"
	"github.com/nusantara-supermart/backend/pkg/response"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetCategories(c fiber.Ctx) error {
	cats, err := h.service.GetCategories()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch ticket categories", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Categories retrieved", cats)
}

func (h *Handler) GetTickets(c fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	userRole, _ := c.Locals("userRole").(string)

	tickets, err := h.service.GetTickets(userID, userRole)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch tickets", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Tickets retrieved", tickets)
}

func (h *Handler) GetTicketByID(c fiber.Ctx) error {
	id := c.Params("id")
	ticket, msgs, rating, err := h.service.GetTicketByID(id)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "Ticket not found", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Ticket retrieved", fiber.Map{
		"ticket":   ticket,
		"messages": msgs,
		"rating":   rating,
	})
}

func (h *Handler) CreateTicket(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized", nil)
	}

	var req CreateTicketRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	ticket, err := h.service.CreateTicket(userID, req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to create ticket", err.Error())
	}

	return response.Success(c, fiber.StatusCreated, "Ticket created successfully", ticket)
}

func (h *Handler) AddMessage(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized", nil)
	}
	ticketID := c.Params("id")

	var req AddMessageRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	msg, err := h.service.AddMessage(userID, ticketID, req.MessageBody)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to add message", err.Error())
	}

	return response.Success(c, fiber.StatusCreated, "Message sent", msg)
}

func (h *Handler) UpdateStatus(c fiber.Ctx) error {
	ticketID := c.Params("id")
	var req UpdateTicketStatusRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	if err := h.service.UpdateStatus(ticketID, req.Status); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to update ticket status", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Ticket status updated", nil)
}

func (h *Handler) AssignTicket(c fiber.Ctx) error {
	ticketID := c.Params("id")
	var req AssignTicketRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	if err := h.service.AssignTicket(ticketID, req.AgentID); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to assign ticket", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Ticket assigned", nil)
}

func (h *Handler) RateTicket(c fiber.Ctx) error {
	ticketID := c.Params("id")
	var req RateTicketRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	if err := h.service.RateTicket(ticketID, req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to rate ticket", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Rating submitted", nil)
}

func (h *Handler) GetFAQs(c fiber.Ctx) error {
	categoryID := c.Query("category_id")
	faqs, err := h.service.GetFAQs(categoryID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch FAQs", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "FAQs retrieved", faqs)
}

func (h *Handler) CreateFAQ(c fiber.Ctx) error {
	var req CreateFAQRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	faq, err := h.service.CreateFAQ(req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to create FAQ", err.Error())
	}

	return response.Success(c, fiber.StatusCreated, "FAQ created", faq)
}

func (h *Handler) GetDisputes(c fiber.Ctx) error {
	disputes, err := h.service.GetDisputes()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch disputes", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Disputes retrieved", disputes)
}

func (h *Handler) CreateDispute(c fiber.Ctx) error {
	var req CreateDisputeRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	d, err := h.service.CreateDispute(req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to create dispute", err.Error())
	}

	return response.Success(c, fiber.StatusCreated, "Dispute created", d)
}
