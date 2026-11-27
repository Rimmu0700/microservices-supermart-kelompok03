package promotion

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

func (h *Handler) ValidateVoucher(c fiber.Ctx) error {
	var req ValidateVoucherRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	res, err := h.service.ValidateVoucher(req.VoucherCode, req.Subtotal)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to validate voucher", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Voucher check complete", res)
}

func (h *Handler) GetPromotions(c fiber.Ctx) error {
	promos, err := h.service.GetPromotions()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch promotions", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Promotions retrieved", promos)
}

func (h *Handler) CreatePromotion(c fiber.Ctx) error {
	var req CreatePromotionRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	p, err := h.service.CreatePromotion(req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to create promotion", err.Error())
	}

	return response.Success(c, fiber.StatusCreated, "Promotion created", p)
}

func (h *Handler) GetVouchers(c fiber.Ctx) error {
	vouchers, err := h.service.GetVouchers()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch vouchers", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Vouchers retrieved", vouchers)
}

func (h *Handler) CreateVoucher(c fiber.Ctx) error {
	var req CreateVoucherRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	v, err := h.service.CreateVoucher(req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to create voucher", err.Error())
	}

	return response.Success(c, fiber.StatusCreated, "Voucher created", v)
}

func (h *Handler) GetLoyalty(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized", nil)
	}

	lp, txs, err := h.service.GetLoyalty(userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch loyalty points", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Loyalty points retrieved", fiber.Map{
		"points":       lp,
		"transactions": txs,
	})
}

func (h *Handler) RedeemPoints(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized", nil)
	}

	var req RedeemPointsRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	if err := h.service.RedeemPoints(userID, req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Points redemption failed", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Points redeemed successfully", nil)
}

func (h *Handler) GetFlashSales(c fiber.Ctx) error {
	sales, err := h.service.GetFlashSales()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch flash sales", err.Error())
	}
	return response.Success(c, fiber.StatusOK, "Flash sales retrieved", sales)
}
