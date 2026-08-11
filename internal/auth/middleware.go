package auth

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

const LocalsUserIDKey = "auth_user_id"

// RequireAuth cek Bearer token di header Authorization. Frontend nyimpen
// token di localStorage (bukan cookie) supaya gak ribet CSRF buat SPA
// single-admin ini.
func RequireAuth(a *Auth) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get("Authorization")
		tokenStr, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || tokenStr == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		claims, err := a.ParseToken(tokenStr)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		c.Locals(LocalsUserIDKey, claims.UserID)
		return c.Next()
	}
}
