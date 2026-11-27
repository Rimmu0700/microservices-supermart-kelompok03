package logistics

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

func (h *Handler) GetCouriers(c fiber.Ctx) error {
	couriers, err := h.service.GetCouriers()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch couriers", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Couriers retrieved", couriers)
}

func (h *Handler) GetServices(c fiber.Ctx) error {
	services, err := h.service.GetServices()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch services", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Services retrieved", services)
}

func (h *Handler) GetRates(c fiber.Ctx) error {
	rates, err := h.service.GetRates()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch rates", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Rates retrieved", rates)
}

func (h *Handler) GetShipments(c fiber.Ctx) error {
	status := c.Query("status")
	shipments, err := h.service.GetShipments(status)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch shipments", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Shipments retrieved", shipments)
}

func (h *Handler) GetShipmentByID(c fiber.Ctx) error {
	id := c.Params("id")
	so, logs, pod, err := h.service.GetShipmentByID(id)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "Shipment not found", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Shipment details retrieved", fiber.Map{
		"shipping_order": so,
		"tracking_logs":  logs,
		"proof":          pod,
	})
}

func (h *Handler) GetShipmentByOrderID(c fiber.Ctx) error {
	orderID := c.Params("order_id")
	so, logs, pod, err := h.service.GetShipmentByOrderID(orderID)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "Shipment not found for order", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Shipment details retrieved", fiber.Map{
		"shipping_order": so,
		"tracking_logs":  logs,
		"proof":          pod,
	})
}

func (h *Handler) CreateShipment(c fiber.Ctx) error {
	var req CreateShippingOrderRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	so, err := h.service.CreateShipment(req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to create shipment", err.Error())
	}

	return response.Success(c, fiber.StatusCreated, "Shipment created", so)
}

func (h *Handler) UpdateStatus(c fiber.Ctx) error {
	id := c.Params("id")
	var req UpdateShippingStatusRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	if err := h.service.UpdateStatus(id, req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to update shipment status", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Status updated successfully", nil)
}

func (h *Handler) SubmitPOD(c fiber.Ctx) error {
	id := c.Params("id")
	var req SubmitPODRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	if err := h.service.SubmitPOD(id, req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to submit POD", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Proof of delivery submitted successfully", nil)
}

func (h *Handler) GetDrivers(c fiber.Ctx) error {
	drivers, err := h.service.GetDrivers()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch drivers", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Drivers retrieved", drivers)
}

func (h *Handler) GetVehicles(c fiber.Ctx) error {
	vehicles, err := h.service.GetVehicles()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch vehicles", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Vehicles retrieved", vehicles)
}

func (h *Handler) GetDeliveryRuns(c fiber.Ctx) error {
	driverID := c.Query("driver_id")
	runs, err := h.service.GetDeliveryRuns(driverID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch delivery runs", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Delivery runs retrieved", runs)
}
