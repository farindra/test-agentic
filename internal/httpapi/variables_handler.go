package httpapi

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// listVariables balikin semua variable custom admin (key-value) yang bisa
// dipakai di system prompt chatbot lewat placeholder {{nama_variable}}.
func (a *API) listVariables(c *fiber.Ctx) error {
	ctx, cancel := ctx15(c)
	defer cancel()
	rows, err := a.st.ListVariables(ctx)
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	return c.JSON(rows)
}

type setVariableReq struct {
	Value string `json:"value"`
}

// setVariable: satu key satu request — sengaja gak dibikin "PUT bulk map"
// kayak sebelumnya, karena itu gak punya cara buat NGHAPUS key (bulk-upsert
// doang), jadi tombol hapus di UI keliatan jalan tapi key-nya nyangkut lagi
// pas di-reload.
func (a *API) setVariable(c *fiber.Ctx) error {
	key := strings.TrimSpace(c.Params("key"))
	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "key wajib"})
	}
	var req setVariableReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "body tidak valid"})
	}
	ctx, cancel := ctx15(c)
	defer cancel()
	if err := a.st.SetVariable(ctx, key, req.Value); err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	return c.JSON(fiber.Map{"key": key, "value": req.Value})
}

func (a *API) deleteVariable(c *fiber.Ctx) error {
	key := c.Params("key")
	ctx, cancel := ctx15(c)
	defer cancel()
	if err := a.st.DeleteVariable(ctx, key); err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}
