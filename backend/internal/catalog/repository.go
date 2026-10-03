package catalog

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repository interface {
	ListProducts(q ProductListQuery) ([]Product, int, error)
	GetProductByID(id string) (*Product, error)
	CreateProduct(p *Product, categoryID, imageURL *string) error
	UpdateProduct(p *Product, categoryID, imageURL *string) error
	DeleteProduct(id string) error
	GetCategories() ([]Category, error)
	CreateCategory(c *Category) error
	GetBrands() ([]Brand, error)
	CreateBrand(b *Brand) error
	GetProductReviews(productID string) ([]ProductReview, error)
	CreateProductReview(r *ProductReview) error
	GetProductVariants(productID string) ([]ProductVariant, error)
	GetProductImages(productID string) ([]ProductImage, error)
	GetProductSummaries(ids []string) ([]ProductSummary, error)
}

type mysqlRepository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &mysqlRepository{db: db}
}

func (r *mysqlRepository) ListProducts(q ProductListQuery) ([]Product, int, error) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.Limit <= 0 {
		q.Limit = 12
	}
	offset := (q.Page - 1) * q.Limit

	whereClauses := []string{"p.is_published = 1"}
	args := []interface{}{}

	if q.Search != "" {
		whereClauses = append(whereClauses, "(p.title LIKE ? OR p.sku LIKE ? OR p.description LIKE ?)")
		searchTerm := "%" + q.Search + "%"
		args = append(args, searchTerm, searchTerm, searchTerm)
	}

	if q.CategoryID != "" {
		whereClauses = append(whereClauses, "pc.category_id = ?")
		args = append(args, q.CategoryID)
	}

	if q.BrandID != "" {
		whereClauses = append(whereClauses, "p.brand_id = ?")
		args = append(args, q.BrandID)
	}

	if q.MinPrice > 0 {
		whereClauses = append(whereClauses, "p.base_price >= ?")
		args = append(args, q.MinPrice)
	}

	if q.MaxPrice > 0 {
		whereClauses = append(whereClauses, "p.base_price <= ?")
		args = append(args, q.MaxPrice)
	}

	whereSQL := "WHERE " + strings.Join(whereClauses, " AND ")

	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT p.id)
		FROM products p
		LEFT JOIN product_categories pc ON p.id = pc.product_id
		%s
	`, whereSQL)

	var total int
	err := r.db.Get(&total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	orderBy := "p.created_at DESC"
	switch q.SortBy {
	case "price_asc":
		orderBy = "p.base_price ASC"
	case "price_desc":
		orderBy = "p.base_price DESC"
	case "title_asc":
		orderBy = "p.title ASC"
	}

	dataQuery := fmt.Sprintf(`
		SELECT p.id, p.sku, p.title, p.description, p.base_price, p.brand_id, p.weight_gram, p.is_published, p.created_at,
		       b.brand_name, c.category_name, pc.category_id,
		       COALESCE(pi.image_url, 'https://images.unsplash.com/photo-1586201375761-83865001e31c?w=500') AS image_url,
		       COALESCE(AVG(pr.rating), 5.0) AS avg_rating,
		       COUNT(pr.id) AS review_count
		FROM products p
		LEFT JOIN brands b ON p.brand_id = b.id
		LEFT JOIN product_categories pc ON p.id = pc.product_id
		LEFT JOIN categories c ON pc.category_id = c.id
		LEFT JOIN product_images pi ON p.id = pi.product_id AND pi.is_primary = 1
		LEFT JOIN product_reviews pr ON p.id = pr.product_id
		%s
		GROUP BY p.id, b.brand_name, c.category_name, pc.category_id, pi.image_url
		ORDER BY %s
		LIMIT ? OFFSET ?
	`, whereSQL, orderBy)

	args = append(args, q.Limit, offset)
	var products []Product
	err = r.db.Select(&products, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return products, total, nil
}

func (r *mysqlRepository) GetProductByID(id string) (*Product, error) {
	query := `
		SELECT p.id, p.sku, p.title, p.description, p.base_price, p.brand_id, p.weight_gram, p.is_published, p.created_at,
		       b.brand_name, c.category_name, pc.category_id,
		       COALESCE(pi.image_url, 'https://images.unsplash.com/photo-1586201375761-83865001e31c?w=500') AS image_url,
		       COALESCE(AVG(pr.rating), 5.0) AS avg_rating,
		       COUNT(pr.id) AS review_count
		FROM products p
		LEFT JOIN brands b ON p.brand_id = b.id
		LEFT JOIN product_categories pc ON p.id = pc.product_id
		LEFT JOIN categories c ON pc.category_id = c.id
		LEFT JOIN product_images pi ON p.id = pi.product_id AND pi.is_primary = 1
		LEFT JOIN product_reviews pr ON p.id = pr.product_id
		WHERE p.id = ?
		GROUP BY p.id, b.brand_name, c.category_name, pc.category_id, pi.image_url
		LIMIT 1
	`
	var p Product
	err := r.db.Get(&p, query, id)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *mysqlRepository) CreateProduct(p *Product, categoryID, imageURL *string) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	p.CreatedAt = time.Now()

	insertProduct := `
		INSERT INTO products (id, sku, title, description, base_price, brand_id, weight_gram, is_published, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = tx.Exec(insertProduct, p.ID, p.SKU, p.Title, p.Description, p.BasePrice, p.BrandID, p.WeightGram, p.IsPublished, p.CreatedAt)
	if err != nil {
		return err
	}

	if categoryID != nil && *categoryID != "" {
		_, err = tx.Exec("INSERT INTO product_categories (product_id, category_id) VALUES (?, ?)", p.ID, *categoryID)
		if err != nil {
			return err
		}
	}

	if imageURL != nil && *imageURL != "" {
		imgID := uuid.NewString()
		_, err = tx.Exec("INSERT INTO product_images (id, product_id, image_url, is_primary, display_order) VALUES (?, ?, ?, 1, 1)", imgID, p.ID, *imageURL)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *mysqlRepository) UpdateProduct(p *Product, categoryID, imageURL *string) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		UPDATE products
		SET title = ?, description = ?, base_price = ?, brand_id = ?, weight_gram = ?, is_published = ?
		WHERE id = ?
	`
	_, err = tx.Exec(query, p.Title, p.Description, p.BasePrice, p.BrandID, p.WeightGram, p.IsPublished, p.ID)
	if err != nil {
		return err
	}

	if categoryID != nil && *categoryID != "" {
		_, _ = tx.Exec("DELETE FROM product_categories WHERE product_id = ?", p.ID)
		_, err = tx.Exec("INSERT INTO product_categories (product_id, category_id) VALUES (?, ?)", p.ID, *categoryID)
		if err != nil {
			return err
		}
	}

	if imageURL != nil && *imageURL != "" {
		_, _ = tx.Exec("DELETE FROM product_images WHERE product_id = ? AND is_primary = 1", p.ID)
		imgID := uuid.NewString()
		_, err = tx.Exec("INSERT INTO product_images (id, product_id, image_url, is_primary, display_order) VALUES (?, ?, ?, 1, 1)", imgID, p.ID, *imageURL)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *mysqlRepository) DeleteProduct(id string) error {
	_, err := r.db.Exec("DELETE FROM products WHERE id = ?", id)
	return err
}

func (r *mysqlRepository) GetCategories() ([]Category, error) {
	var categories []Category
	err := r.db.Select(&categories, "SELECT * FROM categories ORDER BY category_name ASC")
	return categories, err
}

func (r *mysqlRepository) CreateCategory(c *Category) error {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	query := `INSERT INTO categories (id, category_name, slug, icon_url) VALUES (?, ?, ?, ?)`
	_, err := r.db.Exec(query, c.ID, c.CategoryName, c.Slug, c.IconURL)
	return err
}

func (r *mysqlRepository) GetBrands() ([]Brand, error) {
	var brands []Brand
	err := r.db.Select(&brands, "SELECT * FROM brands ORDER BY brand_name ASC")
	return brands, err
}

func (r *mysqlRepository) CreateBrand(b *Brand) error {
	if b.ID == "" {
		b.ID = uuid.NewString()
	}
	query := `INSERT INTO brands (id, brand_name, logo_url, website_url) VALUES (?, ?, ?, ?)`
	_, err := r.db.Exec(query, b.ID, b.BrandName, b.LogoURL, b.WebsiteURL)
	return err
}

func (r *mysqlRepository) GetProductReviews(productID string) ([]ProductReview, error) {
	query := `
		SELECT pr.id, pr.product_id, pr.user_id, pr.rating, pr.review_text, pr.created_at,
		       COALESCE(u.full_name, 'Pelanggan SuperMart') AS full_name
		FROM product_reviews pr
		LEFT JOIN users u ON pr.user_id = u.id
		WHERE pr.product_id = ?
		ORDER BY pr.created_at DESC
	`
	var reviews []ProductReview
	err := r.db.Select(&reviews, query, productID)
	return reviews, err
}

func (r *mysqlRepository) CreateProductReview(rev *ProductReview) error {
	if rev.ID == "" {
		rev.ID = uuid.NewString()
	}
	rev.CreatedAt = time.Now()
	query := `INSERT INTO product_reviews (id, product_id, user_id, rating, review_text, created_at) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := r.db.Exec(query, rev.ID, rev.ProductID, rev.UserID, rev.Rating, rev.ReviewText, rev.CreatedAt)
	return err
}

func (r *mysqlRepository) GetProductVariants(productID string) ([]ProductVariant, error) {
	var variants []ProductVariant
	err := r.db.Select(&variants, "SELECT * FROM product_variants WHERE product_id = ?", productID)
	return variants, err
}

func (r *mysqlRepository) GetProductImages(productID string) ([]ProductImage, error) {
	var images []ProductImage
	err := r.db.Select(&images, "SELECT * FROM product_images WHERE product_id = ? ORDER BY is_primary DESC, display_order ASC", productID)
	return images, err
}
