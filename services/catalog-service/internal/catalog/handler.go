package catalog

import (
	"math"

	"github.com/gofiber/fiber/v3"
	"github.com/nusantara-supermart/backend/pkg/response"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListProducts(c fiber.Ctx) error {
	var q ProductListQuery
	if err := c.Bind().Query(&q); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid query parameters", err.Error())
	}

	if q.Page <= 0 {
		q.Page = 1
	}
	if q.Limit <= 0 {
		q.Limit = 12
	}

	products, total, err := h.service.ListProducts(q)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch products", err.Error())
	}

	totalPages := int(math.Ceil(float64(total) / float64(q.Limit)))

	return response.SuccessPaginated(c, fiber.StatusOK, "Products retrieved", products, response.Pagination{
		Page:       q.Page,
		Limit:      q.Limit,
		TotalItems: total,
		TotalPages: totalPages,
	})
}

func (h *Handler) GetProduct(c fiber.Ctx) error {
	id := c.Params("id")
	product, variants, images, err := h.service.GetProductByID(id)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "Product not found", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Product details retrieved", fiber.Map{
		"product":  product,
		"variants": variants,
		"images":   images,
	})
}

func (h *Handler) CreateProduct(c fiber.Ctx) error {
	var req CreateProductRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	product, err := h.service.CreateProduct(req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to create product", err.Error())
	}

	return response.Success(c, fiber.StatusCreated, "Product created successfully", product)
}

func (h *Handler) UpdateProduct(c fiber.Ctx) error {
	id := c.Params("id")
	var req UpdateProductRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	if err := h.service.UpdateProduct(id, req); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to update product", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Product updated successfully", nil)
}

func (h *Handler) DeleteProduct(c fiber.Ctx) error {
	id := c.Params("id")
	if err := h.service.DeleteProduct(id); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to delete product", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Product deleted successfully", nil)
}

func (h *Handler) GetCategories(c fiber.Ctx) error {
	categories, err := h.service.GetCategories()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch categories", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Categories retrieved", categories)
}

func (h *Handler) CreateCategory(c fiber.Ctx) error {
	var req CategoryRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	cat, err := h.service.CreateCategory(req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to create category", err.Error())
	}

	return response.Success(c, fiber.StatusCreated, "Category created successfully", cat)
}

func (h *Handler) GetBrands(c fiber.Ctx) error {
	brands, err := h.service.GetBrands()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch brands", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Brands retrieved", brands)
}

func (h *Handler) CreateBrand(c fiber.Ctx) error {
	var req BrandRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	b, err := h.service.CreateBrand(req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to create brand", err.Error())
	}

	return response.Success(c, fiber.StatusCreated, "Brand created successfully", b)
}

func (h *Handler) GetReviews(c fiber.Ctx) error {
	productID := c.Params("id")
	reviews, err := h.service.GetProductReviews(productID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch reviews", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Reviews retrieved", reviews)
}

func (h *Handler) AddReview(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized", nil)
	}
	productID := c.Params("id")

	var req CreateReviewRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	if err := h.service.AddReview(userID, productID, req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Failed to submit review", err.Error())
	}

	return response.Success(c, fiber.StatusCreated, "Review submitted successfully", nil)
}
