package httpapi

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"test-agentic/internal/auth"
	"test-agentic/internal/store"
)

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (a *API) login(c *fiber.Ctx) error {
	var req loginReq
	if err := c.BodyParser(&req); err != nil || req.Username == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "username & password wajib"})
	}
	ctx, cancel := ctx15(c)
	defer cancel()

	u, err := a.st.GetUserByUsername(ctx, req.Username)
	if errors.Is(err, store.ErrNotFound) || (err == nil && !auth.CheckPassword(u.PasswordHash, req.Password)) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "username atau password salah"})
	}
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}

	token, err := a.authSvc.IssueToken(u.ID, u.Username)
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	return c.JSON(fiber.Map{"token": token, "username": u.Username})
}
