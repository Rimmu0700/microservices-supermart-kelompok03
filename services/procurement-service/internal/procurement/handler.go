package procurement

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

func (h *Handler) GetSuppliers(c fiber.Ctx) error {
	suppliers, err := h.service.GetSuppliers()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch suppliers", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Suppliers retrieved", suppliers)
}

func (h *Handler) CreateSupplier(c fiber.Ctx) error {
	var req CreateSupplierRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	sup, err := h.service.CreateSupplier(req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to create supplier", err.Error())
	}

	return response.Success(c, fiber.StatusCreated, "Supplier created", sup)
}

func (h *Handler) GetPurchaseOrders(c fiber.Ctx) error {
	pos, err := h.service.GetPurchaseOrders()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch purchase orders", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Purchase orders retrieved", pos)
}

func (h *Handler) GetPurchaseOrderByID(c fiber.Ctx) error {
	id := c.Params("id")
	po, items, err := h.service.GetPurchaseOrderByID(id)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "Purchase order not found", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Purchase order details retrieved", fiber.Map{
		"purchase_order": po,
		"items":          items,
	})
}

func (h *Handler) CreatePurchaseOrder(c fiber.Ctx) error {
	var req CreatePORequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	po, err := h.service.CreatePurchaseOrder(req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to create purchase order", err.Error())
	}

	return response.Success(c, fiber.StatusCreated, "Purchase order created", po)
}

func (h *Handler) ApprovePO(c fiber.Ctx) error {
	id := c.Params("id")
	if err := h.service.ApprovePO(id); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to approve purchase order", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Purchase order approved", nil)
}

func (h *Handler) CreateGRN(c fiber.Ctx) error {
	var req CreateGRNRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	grn, err := h.service.CreateGRN(req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to create goods receipt note", err.Error())
	}

	return response.Success(c, fiber.StatusCreated, "Goods receipt recorded, inventory updated", grn)
}

func (h *Handler) GetGRNs(c fiber.Ctx) error {
	poID := c.Query("po_id")
	grns, err := h.service.GetGRNs(poID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch GRNs", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Goods receipt notes retrieved", grns)
}

func (h *Handler) GetInvoices(c fiber.Ctx) error {
	invoices, err := h.service.GetInvoices()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch purchase invoices", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Purchase invoices retrieved", invoices)
}
