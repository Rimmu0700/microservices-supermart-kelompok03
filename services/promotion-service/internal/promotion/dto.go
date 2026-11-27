package promotion

type ValidateVoucherRequest struct {
	VoucherCode string  `json:"voucher_code"`
	Subtotal    float64 `json:"subtotal"`
}

type ValidateVoucherResponse struct {
	Valid          bool    `json:"valid"`
	VoucherCode    string  `json:"voucher_code"`
	DiscountAmount float64 `json:"discount_amount"`
	Message        string  `json:"message"`
}

type CreateVoucherRequest struct {
	VoucherCode  string  `json:"voucher_code"`
	VoucherValue float64 `json:"voucher_value"`
	QuotaLimit   int     `json:"quota_limit"`
	ExpiresAt    string  `json:"expires_at"` // YYYY-MM-DD
}

type CreatePromotionRequest struct {
	PromoCode string `json:"promo_code"`
	PromoName string `json:"promo_name"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type RedeemPointsRequest struct {
	RewardName     string `json:"reward_name"`
	PointsDeducted int    `json:"points_deducted"`
}
