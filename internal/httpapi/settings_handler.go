package httpapi

import "github.com/gofiber/fiber/v2"

func (a *API) listSettings(c *fiber.Ctx) error {
	ctx, cancel := ctx15(c)
	defer cancel()
	rows, err := a.st.ListSettings(ctx)
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	return c.JSON(rows)
}

func (a *API) updateSettings(c *fiber.Ctx) error {
	var req map[string]string
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "body tidak valid"})
	}
	ctx, cancel := ctx15(c)
	defer cancel()
	for k, v := range req {
		if err := a.st.SetSetting(ctx, k, v); err != nil {
			return errJSON(c, fiber.StatusInternalServerError, err)
		}
	}
	return c.JSON(fiber.Map{"success": true})
}
