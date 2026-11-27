package payment

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

func (h *Handler) GetMethods(c fiber.Ctx) error {
	methods, err := h.service.GetPaymentMethods()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch payment methods", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Payment methods retrieved", methods)
}

func (h *Handler) GetInvoices(c fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	userRole, _ := c.Locals("userRole").(string)

	queryUser := userID
	if userRole == "SUPER_ADMIN" {
		queryUser = ""
	}

	invoices, err := h.service.GetInvoices(queryUser)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch invoices", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Invoices retrieved", invoices)
}

func (h *Handler) GetInvoiceByID(c fiber.Ctx) error {
	id := c.Params("id")
	invoice, txs, err := h.service.GetInvoiceByID(id)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "Invoice not found", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Invoice details retrieved", fiber.Map{
		"invoice":      invoice,
		"transactions": txs,
	})
}

func (h *Handler) GetInvoiceByOrderID(c fiber.Ctx) error {
	orderID := c.Params("order_id")
	invoice, err := h.service.GetInvoiceByOrderID(orderID)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "Invoice not found for order", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Invoice details retrieved", invoice)
}

func (h *Handler) PayInvoice(c fiber.Ctx) error {
	var req PayInvoiceRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	tx, err := h.service.PayInvoice(req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Payment processing failed", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Payment processed successfully", tx)
}

func (h *Handler) GetStoreCredit(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized", nil)
	}

	sc, txs, err := h.service.GetStoreCredit(userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch store credit", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Store credit retrieved", fiber.Map{
		"credit":       sc,
		"transactions": txs,
	})
}

func (h *Handler) TopupStoreCredit(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized", nil)
	}

	var req TopupCreditRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	if err := h.service.TopupStoreCredit(userID, req.Amount); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to topup store credit", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Topup successful", nil)
}

func (h *Handler) RequestRefund(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized", nil)
	}

	var req RefundRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	ref, err := h.service.RequestRefund(userID, req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to request refund", err.Error())
	}

	return response.Success(c, fiber.StatusCreated, "Refund request submitted", ref)
}

func (h *Handler) GetRefunds(c fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	userRole, _ := c.Locals("userRole").(string)

	refunds, err := h.service.GetRefunds(userID, userRole)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch refunds", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Refunds retrieved", refunds)
}

func (h *Handler) ApproveRefund(c fiber.Ctx) error {
	id := c.Params("id")
	var req ApproveRefundRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	approved := req.Status == "APPROVED"
	if err := h.service.ApproveRefund(id, approved); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to update refund", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Refund status updated", nil)
}
