package auth

import (
	"github.com/gofiber/fiber/v3"
	"github.com/nusantara-supermart/backend/pkg/response"
	"github.com/nusantara-supermart/backend/pkg/validator"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Login(c fiber.Ctx) error {
	var req LoginRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	errs := make(validator.ValidationErrors)
	validator.ValidateRequired("email", req.Email, errs)
	validator.ValidateRequired("password", req.Password, errs)
	if err := validator.Check(errs); err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "Validation failed", err.Error())
	}

	res, err := h.service.Login(req)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "Login successful", res)
}

func (h *Handler) Register(c fiber.Ctx) error {
	var req RegisterRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	errs := make(validator.ValidationErrors)
	validator.ValidateRequired("email", req.Email, errs)
	validator.ValidateRequired("password", req.Password, errs)
	validator.ValidateRequired("full_name", req.FullName, errs)
	if err := validator.Check(errs); err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "Validation failed", err.Error())
	}

	res, err := h.service.Register(req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusCreated, "Registration successful", res)
}

func (h *Handler) GetMe(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized", nil)
	}

	user, profile, err := h.service.GetMe(userID)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "User not found", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "User profile retrieved", fiber.Map{
		"user":    user,
		"profile": profile,
	})
}

func (h *Handler) UpdateProfile(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized", nil)
	}

	var req UpdateProfileRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	if err := h.service.UpdateProfile(userID, req); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to update profile", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Profile updated successfully", nil)
}

func (h *Handler) GetAddresses(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized", nil)
	}

	addresses, err := h.service.GetAddresses(userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch addresses", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Addresses retrieved", addresses)
}

func (h *Handler) AddAddress(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized", nil)
	}

	var req AddressRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	errs := make(validator.ValidationErrors)
	validator.ValidateRequired("address_label", req.AddressLabel, errs)
	validator.ValidateRequired("recipient_name", req.RecipientName, errs)
	validator.ValidateRequired("street_address", req.StreetAddress, errs)
	validator.ValidateRequired("city", req.City, errs)
	validator.ValidateRequired("province", req.Province, errs)
	validator.ValidateRequired("postal_code", req.PostalCode, errs)
	if err := validator.Check(errs); err != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "Validation failed", err.Error())
	}

	if err := h.service.AddAddress(userID, req); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to add address", err.Error())
	}

	return response.Success(c, fiber.StatusCreated, "Address added successfully", nil)
}

func (h *Handler) DeleteAddress(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized", nil)
	}
	id := c.Params("id")

	if err := h.service.DeleteAddress(id, userID); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to delete address", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Address deleted successfully", nil)
}

func (h *Handler) SubmitKYC(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized", nil)
	}

	var req KYCRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	if err := h.service.SubmitKYC(userID, req); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to submit KYC", err.Error())
	}

	return response.Success(c, fiber.StatusCreated, "KYC submitted successfully", nil)
}

func (h *Handler) GetKYC(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok || userID == "" {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized", nil)
	}

	kyc, err := h.service.GetKYC(userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch KYC document", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "KYC retrieved", kyc)
}

func (h *Handler) VerifyKYC(c fiber.Ctx) error {
	id := c.Params("id")
	var req VerifyKYCRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	if err := h.service.VerifyKYC(id, req.Status); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to verify KYC", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "KYC status updated", nil)
}

func (h *Handler) GetAllUsers(c fiber.Ctx) error {
	users, err := h.service.GetAllUsers()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch users", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Users retrieved", users)
}

func (h *Handler) GetRoles(c fiber.Ctx) error {
	roles, err := h.service.GetRoles()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch roles", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Roles retrieved", roles)
}

func (h *Handler) GetPermissions(c fiber.Ctx) error {
	permissions, err := h.service.GetPermissions()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch permissions", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Permissions retrieved", permissions)
}

func (h *Handler) CreateUser(c fiber.Ctx) error {
	var req AdminCreateUserRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Payload request tidak valid", err.Error())
	}

	user, err := h.service.CreateUserByAdmin(req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Gagal membuat pengguna baru", err.Error())
	}

	return response.Success(c, fiber.StatusCreated, "Pengguna berhasil dibuat", user)
}

func (h *Handler) UpdateUser(c fiber.Ctx) error {
	id := c.Params("id")
	var req AdminUpdateUserRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Payload request tidak valid", err.Error())
	}

	if err := h.service.UpdateUserByAdmin(id, req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Gagal memperbarui pengguna", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Data pengguna berhasil diperbarui", nil)
}

func (h *Handler) ToggleUserStatus(c fiber.Ctx) error {
	id := c.Params("id")
	var req ToggleStatusRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Payload request tidak valid", err.Error())
	}

	if err := h.service.ToggleUserStatus(id, req.IsActive); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Gagal mengubah status pengguna", err.Error())
	}

	msg := "Akun pengguna berhasil diaktifkan"
	if !req.IsActive {
		msg = "Akun pengguna berhasil dinonaktifkan"
	}
	return response.Success(c, fiber.StatusOK, msg, nil)
}

func (h *Handler) DeleteUser(c fiber.Ctx) error {
	id := c.Params("id")
	if err := h.service.DeleteUser(id); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Gagal menghapus pengguna", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Pengguna berhasil dihapus", nil)
}

