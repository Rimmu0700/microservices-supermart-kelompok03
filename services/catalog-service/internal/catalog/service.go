package catalog

import (
	"errors"

	"github.com/google/uuid"
)

type Service interface {
	ListProducts(q ProductListQuery) ([]Product, int, error)
	GetProductByID(id string) (*Product, []ProductVariant, []ProductImage, error)
	CreateProduct(req CreateProductRequest) (*Product, error)
	UpdateProduct(id string, req UpdateProductRequest) error
	DeleteProduct(id string) error
	GetCategories() ([]Category, error)
	CreateCategory(req CategoryRequest) (*Category, error)
	GetBrands() ([]Brand, error)
	CreateBrand(req BrandRequest) (*Brand, error)
	GetProductReviews(productID string) ([]ProductReview, error)
	AddReview(userID, productID string, req CreateReviewRequest) error
	GetProductSummaries(ids []string) ([]ProductSummary, error)
}

type catalogService struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &catalogService{repo: repo}
}

func (s *catalogService) ListProducts(q ProductListQuery) ([]Product, int, error) {
	return s.repo.ListProducts(q)
}

func (s *catalogService) GetProductByID(id string) (*Product, []ProductVariant, []ProductImage, error) {
	product, err := s.repo.GetProductByID(id)
	if err != nil {
		return nil, nil, nil, err
	}
	variants, _ := s.repo.GetProductVariants(id)
	images, _ := s.repo.GetProductImages(id)
	return product, variants, images, nil
}

func (s *catalogService) CreateProduct(req CreateProductRequest) (*Product, error) {
	if req.SKU == "" || req.Title == "" || req.BasePrice <= 0 {
		return nil, errors.New("sku, title, and a valid base_price are required")
	}

	p := &Product{
		ID:          uuid.NewString(),
		SKU:         req.SKU,
		Title:       req.Title,
		Description: req.Description,
		BasePrice:   req.BasePrice,
		BrandID:     req.BrandID,
		WeightGram:  req.WeightGram,
		IsPublished: req.IsPublished,
	}

	err := s.repo.CreateProduct(p, req.CategoryID, req.ImageURL)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (s *catalogService) UpdateProduct(id string, req UpdateProductRequest) error {
	p := &Product{
		ID:          id,
		Title:       req.Title,
		Description: req.Description,
		BasePrice:   req.BasePrice,
		BrandID:     req.BrandID,
		WeightGram:  req.WeightGram,
		IsPublished: req.IsPublished,
	}
	return s.repo.UpdateProduct(p, req.CategoryID, req.ImageURL)
}

func (s *catalogService) DeleteProduct(id string) error {
	return s.repo.DeleteProduct(id)
}

func (s *catalogService) GetCategories() ([]Category, error) {
	return s.repo.GetCategories()
}

func (s *catalogService) CreateCategory(req CategoryRequest) (*Category, error) {
	c := &Category{
		ID:           uuid.NewString(),
		CategoryName: req.CategoryName,
		Slug:         req.Slug,
		IconURL:      req.IconURL,
	}
	if err := s.repo.CreateCategory(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *catalogService) GetBrands() ([]Brand, error) {
	return s.repo.GetBrands()
}

func (s *catalogService) CreateBrand(req BrandRequest) (*Brand, error) {
	b := &Brand{
		ID:         uuid.NewString(),
		BrandName:  req.BrandName,
		LogoURL:    req.LogoURL,
		WebsiteURL: req.WebsiteURL,
	}
	if err := s.repo.CreateBrand(b); err != nil {
		return nil, err
	}
	return b, nil
}

func (s *catalogService) GetProductReviews(productID string) ([]ProductReview, error) {
	return s.repo.GetProductReviews(productID)
}

func (s *catalogService) AddReview(userID, productID string, req CreateReviewRequest) error {
	if req.Rating < 1 || req.Rating > 5 {
		return errors.New("rating must be between 1 and 5")
	}
	rev := &ProductReview{
		ProductID:  productID,
		UserID:     userID,
		Rating:     req.Rating,
		ReviewText: req.ReviewText,
	}
	return s.repo.CreateProductReview(rev)
}
