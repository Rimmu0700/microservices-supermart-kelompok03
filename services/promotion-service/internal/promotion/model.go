package promotion

import (
	"time"
)

type Promotion struct {
	ID        string    `db:"id" json:"id"`
	PromoCode string    `db:"promo_code" json:"promo_code"`
	PromoName string    `db:"promo_name" json:"promo_name"`
	StartDate time.Time `db:"start_date" json:"start_date"`
	EndDate   time.Time `db:"end_date" json:"end_date"`
	IsActive  bool      `db:"is_active" json:"is_active"`
}

type PromotionRule struct {
	ID                 string   `db:"id" json:"id"`
	PromotionID        string   `db:"promotion_id" json:"promotion_id"`
	MinPurchaseAmount  float64  `db:"min_purchase_amount" json:"min_purchase_amount"`
	DiscountPercentage float64  `db:"discount_percentage" json:"discount_percentage"`
	MaxDiscountCap     *float64 `db:"max_discount_cap" json:"max_discount_cap"`
}

type Voucher struct {
	ID           string    `db:"id" json:"id"`
	VoucherCode  string    `db:"voucher_code" json:"voucher_code"`
	VoucherValue float64   `db:"voucher_value" json:"voucher_value"`
	QuotaLimit   int       `db:"quota_limit" json:"quota_limit"`
	QuotaUsed    int       `db:"quota_used" json:"quota_used"`
	ExpiresAt    time.Time `db:"expires_at" json:"expires_at"`
}

type VoucherUsage struct {
	ID        string    `db:"id" json:"id"`
	VoucherID string    `db:"voucher_id" json:"voucher_id"`
	UserID    string    `db:"user_id" json:"user_id"`
	OrderID   string    `db:"order_id" json:"order_id"`
	UsedAt    time.Time `db:"used_at" json:"used_at"`
}

type LoyaltyPoint struct {
	ID            string `db:"id" json:"id"`
	UserID        string `db:"user_id" json:"user_id"`
	CurrentPoints int    `db:"current_points" json:"current_points"`
}

type PointTransaction struct {
	ID               string    `db:"id" json:"id"`
	LoyaltyPointID   string    `db:"loyalty_point_id" json:"loyalty_point_id"`
	Points           int       `db:"points" json:"points"`
	PointType        string    `db:"point_type" json:"point_type"` // EARNED, REDEEMED
	ReferenceOrderID *string   `db:"reference_order_id" json:"reference_order_id"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
}

type FlashSale struct {
	ID             string    `db:"id" json:"id"`
	ProductID      string    `db:"product_id" json:"product_id"`
	FlashPrice     float64   `db:"flash_price" json:"flash_price"`
	AllocatedStock int       `db:"allocated_stock" json:"allocated_stock"`
	StartsAt       time.Time `db:"starts_at" json:"starts_at"`
	EndsAt         time.Time `db:"ends_at" json:"ends_at"`

	// Joined
	ProductTitle string  `db:"product_title" json:"product_title,omitempty"`
	BasePrice    float64 `db:"base_price" json:"base_price,omitempty"`
	ImageURL     string  `db:"image_url" json:"image_url,omitempty"`
}
