package catalog

import (
	"time"
)

type Brand struct {
	ID         string  `db:"id" json:"id"`
	BrandName  string  `db:"brand_name" json:"brand_name"`
	LogoURL    *string `db:"logo_url" json:"logo_url"`
	WebsiteURL *string `db:"website_url" json:"website_url"`
}

type Category struct {
	ID           string  `db:"id" json:"id"`
	CategoryName string  `db:"category_name" json:"category_name"`
	Slug         string  `db:"slug" json:"slug"`
	IconURL      *string `db:"icon_url" json:"icon_url"`
}

type Product struct {
	ID          string    `db:"id" json:"id"`
	SKU         string    `db:"sku" json:"sku"`
	Title       string    `db:"title" json:"title"`
	Description *string   `db:"description" json:"description"`
	BasePrice   float64   `db:"base_price" json:"base_price"`
	BrandID     *string   `db:"brand_id" json:"brand_id"`
	WeightGram  int       `db:"weight_gram" json:"weight_gram"`
	IsPublished bool      `db:"is_published" json:"is_published"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`

	// Joined fields
	BrandName    *string `db:"brand_name" json:"brand_name,omitempty"`
	CategoryName *string `db:"category_name" json:"category_name,omitempty"`
	CategoryID   *string `db:"category_id" json:"category_id,omitempty"`
	ImageURL     *string `db:"image_url" json:"image_url,omitempty"`
	Rating       float64 `db:"avg_rating" json:"rating"`
	ReviewCount  int     `db:"review_count" json:"review_count"`
}

type ProductImage struct {
	ID           string `db:"id" json:"id"`
	ProductID    string `db:"product_id" json:"product_id"`
	ImageURL     string `db:"image_url" json:"image_url"`
	IsPrimary    bool   `db:"is_primary" json:"is_primary"`
	DisplayOrder int    `db:"display_order" json:"display_order"`
}

type ProductVariant struct {
	ID              string  `db:"id" json:"id"`
	ProductID       string  `db:"product_id" json:"product_id"`
	VariantSKU      string  `db:"variant_sku" json:"variant_sku"`
	AdditionalPrice float64 `db:"additional_price" json:"additional_price"`
}

type ProductReview struct {
	ID         string    `db:"id" json:"id"`
	ProductID  string    `db:"product_id" json:"product_id"`
	UserID     string    `db:"user_id" json:"user_id"`
	Rating     int       `db:"rating" json:"rating"`
	ReviewText *string   `db:"review_text" json:"review_text"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`

	// Joined
	UserFullName string `db:"full_name" json:"user_name,omitempty"`
}

type Tag struct {
	ID      string `db:"id" json:"id"`
	TagName string `db:"tag_name" json:"tag_name"`
}
