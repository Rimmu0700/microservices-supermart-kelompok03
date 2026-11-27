package inventory

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

func (h *Handler) GetStocks(c fiber.Ctx) error {
	warehouseID := c.Query("warehouse_id")
	stocks, err := h.service.GetStocks(warehouseID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch stocks", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Stocks retrieved", stocks)
}

func (h *Handler) AdjustStock(c fiber.Ctx) error {
	stockID := c.Params("id")
	var req AdjustStockRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	if err := h.service.AdjustStock(stockID, req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to adjust stock", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Stock adjusted successfully", nil)
}

func (h *Handler) CreateMutation(c fiber.Ctx) error {
	var req StockMutationRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	if err := h.service.CreateMutation(req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to create mutation", err.Error())
	}

	return response.Success(c, fiber.StatusCreated, "Stock mutation executed", nil)
}

func (h *Handler) GetMutations(c fiber.Ctx) error {
	mutations, err := h.service.GetMutations()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch mutations", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Mutations retrieved", mutations)
}

func (h *Handler) GetWarehouses(c fiber.Ctx) error {
	warehouses, err := h.service.GetWarehouses()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch warehouses", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Warehouses retrieved", warehouses)
}

func (h *Handler) CreateWarehouse(c fiber.Ctx) error {
	var req WarehouseRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	w, err := h.service.CreateWarehouse(req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to create warehouse", err.Error())
	}

	return response.Success(c, fiber.StatusCreated, "Warehouse created", w)
}

func (h *Handler) GetWarehouseZones(c fiber.Ctx) error {
	warehouseID := c.Params("id")
	zones, err := h.service.GetWarehouseZones(warehouseID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch zones", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Zones retrieved", zones)
}

func (h *Handler) GetLowStockAlerts(c fiber.Ctx) error {
	alerts, err := h.service.GetLowStockAlerts()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch alerts", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Alerts retrieved", alerts)
}

func (h *Handler) GetOpnames(c fiber.Ctx) error {
	warehouseID := c.Query("warehouse_id")
	opnames, err := h.service.GetOpnames(warehouseID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch opnames", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Opnames retrieved", opnames)
}

func (h *Handler) CreateOpname(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized", nil)
	}

	var req CreateOpnameRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	op, err := h.service.CreateOpname(userID, req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to create opname", err.Error())
	}

	return response.Success(c, fiber.StatusCreated, "Opname scheduled", op)
}
