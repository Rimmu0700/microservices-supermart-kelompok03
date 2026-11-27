package catalog

type ProductListQuery struct {
	Search     string  `query:"search"`
	CategoryID string  `query:"category_id"`
	BrandID    string  `query:"brand_id"`
	MinPrice   float64 `query:"min_price"`
	MaxPrice   float64 `query:"max_price"`
	Page       int     `query:"page"`
	Limit      int     `query:"limit"`
	SortBy     string  `query:"sort_by"`
}

type CreateProductRequest struct {
	SKU         string   `json:"sku"`
	Title       string   `json:"title"`
	Description *string  `json:"description"`
	BasePrice   float64  `json:"base_price"`
	BrandID     *string  `json:"brand_id"`
	CategoryID  *string  `json:"category_id"`
	WeightGram  int      `json:"weight_gram"`
	ImageURL    *string  `json:"image_url"`
	IsPublished bool     `json:"is_published"`
}

type UpdateProductRequest struct {
	Title       string   `json:"title"`
	Description *string  `json:"description"`
	BasePrice   float64  `json:"base_price"`
	BrandID     *string  `json:"brand_id"`
	CategoryID  *string  `json:"category_id"`
	WeightGram  int      `json:"weight_gram"`
	ImageURL    *string  `json:"image_url"`
	IsPublished bool     `json:"is_published"`
}

type CreateReviewRequest struct {
	Rating     int     `json:"rating"`
	ReviewText *string `json:"review_text"`
}

type CategoryRequest struct {
	CategoryName string  `json:"category_name"`
	Slug         string  `json:"slug"`
	IconURL      *string `json:"icon_url"`
}

type BrandRequest struct {
	BrandName  string  `json:"brand_name"`
	LogoURL    *string `json:"logo_url"`
	WebsiteURL *string `json:"website_url"`
}
