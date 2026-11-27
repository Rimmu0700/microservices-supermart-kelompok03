package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/nusantara-supermart/backend/pkg/response"
)

func RolesRequired(allowedRoles ...string) fiber.Handler {
	return func(c fiber.Ctx) error {
		userRole, ok := c.Locals("userRole").(string)
		if !ok || userRole == "" {
			return response.Error(c, fiber.StatusForbidden, "Role access denied: unauthenticated", nil)
		}

		for _, role := range allowedRoles {
			if role == userRole || userRole == "SUPER_ADMIN" {
				return c.Next()
			}
		}

		return response.Error(c, fiber.StatusForbidden, "Forbidden: insufficient role privileges", nil)
	}
}

// StrictRolesRequired checks exact role match without bypassing for SUPER_ADMIN
func StrictRolesRequired(allowedRoles ...string) fiber.Handler {
	return func(c fiber.Ctx) error {
		userRole, ok := c.Locals("userRole").(string)
		if !ok || userRole == "" {
			return response.Error(c, fiber.StatusForbidden, "Role access denied: unauthenticated", nil)
		}

		for _, role := range allowedRoles {
			if role == userRole {
				return c.Next()
			}
		}

		return response.Error(c, fiber.StatusForbidden, "Forbidden: hanya pelanggan yang diizinkan untuk berbelanja", nil)
	}
}
