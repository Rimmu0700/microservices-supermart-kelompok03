package catalog

import "github.com/jmoiron/sqlx"

// ProductSummary: hanya 4 field esensial untuk konsumen (Order).
type ProductSummary struct {
	ID          string  `db:"id" json:"id"`
	Title       string  `db:"title" json:"title"`
	BasePrice   float64 `db:"base_price" json:"base_price"`
	IsPublished bool    `db:"is_published" json:"is_published"`
}

// GetProductSummaries mengambil banyak produk dengan SATU query.
func (r *mysqlRepository) GetProductSummaries(ids []string) ([]ProductSummary, error) {
	if len(ids) == 0 {
		return []ProductSummary{}, nil
	}
	query, args, err := sqlx.In(
		`SELECT id, title, base_price, is_published FROM products WHERE id IN (?)`, ids)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)

	var out []ProductSummary
	if err := r.db.Select(&out, query, args...); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *catalogService) GetProductSummaries(ids []string) ([]ProductSummary, error) {
	return s.repo.GetProductSummaries(ids)
}
