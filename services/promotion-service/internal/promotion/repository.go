package promotion

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repository interface {
	GetPromotions() ([]Promotion, error)
	CreatePromotion(p *Promotion) error
	GetVouchers() ([]Voucher, error)
	GetVoucherByCode(code string) (*Voucher, error)
	CreateVoucher(v *Voucher) error
	GetOrCreateLoyaltyPoints(userID string) (*LoyaltyPoint, error)
	AddLoyaltyPoints(userID string, pts int, refOrderID string) error
	RedeemPoints(userID string, rewardName string, pts int) error
	GetPointTransactions(loyaltyID string) ([]PointTransaction, error)
	GetFlashSales() ([]FlashSale, error)
}

type mysqlRepository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &mysqlRepository{db: db}
}

func (r *mysqlRepository) GetPromotions() ([]Promotion, error) {
	var promos []Promotion
	err := r.db.Select(&promos, "SELECT * FROM promotions ORDER BY start_date DESC")
	return promos, err
}

func (r *mysqlRepository) CreatePromotion(p *Promotion) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	query := `INSERT INTO promotions (id, promo_code, promo_name, start_date, end_date, is_active) VALUES (?, ?, ?, ?, ?, 1)`
	_, err := r.db.Exec(query, p.ID, p.PromoCode, p.PromoName, p.StartDate, p.EndDate)
	return err
}

func (r *mysqlRepository) GetVouchers() ([]Voucher, error) {
	var vouchers []Voucher
	err := r.db.Select(&vouchers, "SELECT * FROM vouchers ORDER BY expires_at DESC")
	return vouchers, err
}

func (r *mysqlRepository) GetVoucherByCode(code string) (*Voucher, error) {
	var v Voucher
	err := r.db.Get(&v, "SELECT * FROM vouchers WHERE voucher_code = ? LIMIT 1", code)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *mysqlRepository) CreateVoucher(v *Voucher) error {
	if v.ID == "" {
		v.ID = uuid.NewString()
	}
	query := `INSERT INTO vouchers (id, voucher_code, voucher_value, quota_limit, quota_used, expires_at) VALUES (?, ?, ?, ?, 0, ?)`
	_, err := r.db.Exec(query, v.ID, v.VoucherCode, v.VoucherValue, v.QuotaLimit, v.ExpiresAt)
	return err
}

func (r *mysqlRepository) GetOrCreateLoyaltyPoints(userID string) (*LoyaltyPoint, error) {
	var lp LoyaltyPoint
	err := r.db.Get(&lp, "SELECT * FROM loyalty_points WHERE user_id = ? LIMIT 1", userID)
	if err == nil {
		return &lp, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	newID := uuid.NewString()
	_, err = r.db.Exec("INSERT INTO loyalty_points (id, user_id, current_points) VALUES (?, ?, 0)", newID, userID)
	if err != nil {
		return nil, err
	}
	return &LoyaltyPoint{ID: newID, UserID: userID, CurrentPoints: 0}, nil
}

func (r *mysqlRepository) AddLoyaltyPoints(userID string, pts int, refOrderID string) error {
	lp, err := r.GetOrCreateLoyaltyPoints(userID)
	if err != nil {
		return err
	}

	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("UPDATE loyalty_points SET current_points = current_points + ? WHERE id = ?", pts, lp.ID)
	if err != nil {
		return err
	}

	txID := uuid.NewString()
	_, err = tx.Exec(`
		INSERT INTO point_transactions (id, loyalty_point_id, points, point_type, reference_order_id, created_at)
		VALUES (?, ?, ?, 'EARNED', ?, NOW())
	`, txID, lp.ID, pts, refOrderID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *mysqlRepository) RedeemPoints(userID string, rewardName string, pts int) error {
	lp, err := r.GetOrCreateLoyaltyPoints(userID)
	if err != nil {
		return err
	}

	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec("UPDATE loyalty_points SET current_points = current_points - ? WHERE id = ? AND current_points >= ?", pts, lp.ID, pts)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("insufficient loyalty points")
	}

	txID := uuid.NewString()
	_, err = tx.Exec(`
		INSERT INTO point_transactions (id, loyalty_point_id, points, point_type, created_at)
		VALUES (?, ?, ?, 'REDEEMED', NOW())
	`, txID, lp.ID, pts)
	if err != nil {
		return err
	}

	redemptionID := uuid.NewString()
	_, err = tx.Exec(`
		INSERT INTO point_redemptions (id, user_id, reward_name, points_deducted, redeemed_at)
		VALUES (?, ?, ?, ?, NOW())
	`, redemptionID, userID, rewardName, pts)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *mysqlRepository) GetPointTransactions(loyaltyID string) ([]PointTransaction, error) {
	var txs []PointTransaction
	err := r.db.Select(&txs, "SELECT * FROM point_transactions WHERE loyalty_point_id = ? ORDER BY created_at DESC", loyaltyID)
	return txs, err
}

func (r *mysqlRepository) GetFlashSales() ([]FlashSale, error) {
	query := `
		SELECT fs.id, fs.product_id, fs.flash_price, fs.allocated_stock, fs.starts_at, fs.ends_at,
		       p.title AS product_title, p.base_price,
		       COALESCE(pi.image_url, 'https://images.unsplash.com/photo-1527864550417-7fd91fc51a46?w=500') AS image_url
		FROM flash_sales fs
		JOIN products p ON fs.product_id = p.id
		LEFT JOIN product_images pi ON p.id = pi.product_id AND pi.is_primary = 1
		WHERE NOW() BETWEEN fs.starts_at AND fs.ends_at
	`
	var sales []FlashSale
	err := r.db.Select(&sales, query)
	return sales, err
}
