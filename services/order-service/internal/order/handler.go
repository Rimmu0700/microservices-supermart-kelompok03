package order

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

func (h *Handler) GetCart(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized", nil)
	}

	cart, err := h.service.GetCart(userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to get cart", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Cart retrieved", cart)
}

func (h *Handler) AddToCart(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized", nil)
	}

	var req AddToCartRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	if err := h.service.AddToCart(userID, req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to add to cart", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Item added to cart", nil)
}

func (h *Handler) UpdateCartItem(c fiber.Ctx) error {
	itemID := c.Params("id")
	var req UpdateCartItemRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	if err := h.service.UpdateCartItem(itemID, req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to update item", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Cart updated", nil)
}

func (h *Handler) RemoveCartItem(c fiber.Ctx) error {
	itemID := c.Params("id")
	if err := h.service.RemoveCartItem(itemID); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to remove item", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Item removed", nil)
}

func (h *Handler) Checkout(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized", nil)
	}

	var req CheckoutRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	order, err := h.service.Checkout(userID, req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Checkout failed", err.Error())
	}

	return response.Success(c, fiber.StatusCreated, "Order created successfully", order)
}

func (h *Handler) GetOrders(c fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	userRole, _ := c.Locals("userRole").(string)
	statusID := c.Query("status_id")

	orders, err := h.service.GetOrders(userID, userRole, statusID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch orders", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Orders retrieved", orders)
}

func (h *Handler) GetOrderByID(c fiber.Ctx) error {
	id := c.Params("id")
	order, items, shipping, histories, notes, err := h.service.GetOrderByID(id)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "Order not found", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Order details retrieved", fiber.Map{
		"order":     order,
		"items":     items,
		"shipping":  shipping,
		"histories": histories,
		"notes":     notes,
	})
}

func (h *Handler) UpdateOrderStatus(c fiber.Ctx) error {
	id := c.Params("id")
	var req UpdateOrderStatusRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	if err := h.service.UpdateStatus(id, req.StatusCode, req.Notes); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to update order status", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Order status updated", nil)
}

func (h *Handler) CancelOrder(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized", nil)
	}
	id := c.Params("id")

	var req CancelOrderRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	if err := h.service.CancelOrder(id, req.Reason, userID); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to cancel order", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Order cancelled successfully", nil)
}

func (h *Handler) AddOrderNote(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized", nil)
	}
	id := c.Params("id")

	var req AddOrderNoteRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	if err := h.service.AddOrderNote(id, userID, req.NoteContent); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to add note", err.Error())
	}

	return response.Success(c, fiber.StatusCreated, "Note added", nil)
}

func (h *Handler) GetStatuses(c fiber.Ctx) error {
	statuses, err := h.service.GetStatuses()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch statuses", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Statuses retrieved", statuses)
}
