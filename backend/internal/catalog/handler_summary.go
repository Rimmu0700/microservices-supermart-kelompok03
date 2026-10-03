package catalog

import (
	"github.com/gofiber/fiber/v3"
	"github.com/nusantara-supermart/backend/pkg/response"
)

// GetProductSummary: padanan REST dari gRPC GetProduct (query dan data identik).
func (h *Handler) GetProductSummary(c fiber.Ctx) error {
	list, err := h.service.GetProductSummaries([]string{c.Params("id")})
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch product", err.Error())
	}
	if len(list) == 0 {
		return response.Error(c, fiber.StatusNotFound, "Product not found", nil)
	}
	return response.Success(c, fiber.StatusOK, "Product summary retrieved", list[0])
}
