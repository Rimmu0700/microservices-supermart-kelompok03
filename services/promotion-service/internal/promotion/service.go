package promotion

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Service interface {
	ValidateVoucher(code string, subtotal float64) (*ValidateVoucherResponse, error)
	GetPromotions() ([]Promotion, error)
	CreatePromotion(req CreatePromotionRequest) (*Promotion, error)
	GetVouchers() ([]Voucher, error)
	CreateVoucher(req CreateVoucherRequest) (*Voucher, error)
	GetLoyalty(userID string) (*LoyaltyPoint, []PointTransaction, error)
	RedeemPoints(userID string, req RedeemPointsRequest) error
	GetFlashSales() ([]FlashSale, error)
}

type promotionService struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &promotionService{repo: repo}
}

func (s *promotionService) ValidateVoucher(code string, subtotal float64) (*ValidateVoucherResponse, error) {
	v, err := s.repo.GetVoucherByCode(code)
	if err != nil {
		return &ValidateVoucherResponse{
			Valid:   false,
			Message: "Kode voucher tidak ditemukan atau tidak aktif",
		}, nil
	}

	if time.Now().After(v.ExpiresAt) {
		return &ValidateVoucherResponse{
			Valid:   false,
			Message: "Voucher telah kedaluwarsa",
		}, nil
	}

	if v.QuotaUsed >= v.QuotaLimit {
		return &ValidateVoucherResponse{
			Valid:   false,
			Message: "Kuota voucher telah habis",
		}, nil
	}

	discount := v.VoucherValue
	if discount > subtotal {
		discount = subtotal
	}

	return &ValidateVoucherResponse{
		Valid:          true,
		VoucherCode:    v.VoucherCode,
		DiscountAmount: discount,
		Message:        "Voucher berhasil diterapkan!",
	}, nil
}

func (s *promotionService) GetPromotions() ([]Promotion, error) {
	return s.repo.GetPromotions()
}

func (s *promotionService) CreatePromotion(req CreatePromotionRequest) (*Promotion, error) {
	start, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		start = time.Now()
	}
	end, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		end = time.Now().Add(30 * 24 * time.Hour)
	}

	p := &Promotion{
		ID:        uuid.NewString(),
		PromoCode: req.PromoCode,
		PromoName: req.PromoName,
		StartDate: start,
		EndDate:   end,
		IsActive:  true,
	}

	if err := s.repo.CreatePromotion(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *promotionService) GetVouchers() ([]Voucher, error) {
	return s.repo.GetVouchers()
}

func (s *promotionService) CreateVoucher(req CreateVoucherRequest) (*Voucher, error) {
	expiresAt, err := time.Parse("2006-01-02", req.ExpiresAt)
	if err != nil {
		expiresAt = time.Now().Add(365 * 24 * time.Hour)
	}

	v := &Voucher{
		ID:           uuid.NewString(),
		VoucherCode:  req.VoucherCode,
		VoucherValue: req.VoucherValue,
		QuotaLimit:   req.QuotaLimit,
		QuotaUsed:    0,
		ExpiresAt:    expiresAt,
	}

	if err := s.repo.CreateVoucher(v); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *promotionService) GetLoyalty(userID string) (*LoyaltyPoint, []PointTransaction, error) {
	lp, err := s.repo.GetOrCreateLoyaltyPoints(userID)
	if err != nil {
		return nil, nil, err
	}
	txs, _ := s.repo.GetPointTransactions(lp.ID)
	return lp, txs, nil
}

func (s *promotionService) RedeemPoints(userID string, req RedeemPointsRequest) error {
	if req.PointsDeducted <= 0 {
		return errors.New("points deducted must be greater than zero")
	}
	return s.repo.RedeemPoints(userID, req.RewardName, req.PointsDeducted)
}

func (s *promotionService) GetFlashSales() ([]FlashSale, error) {
	return s.repo.GetFlashSales()
}
