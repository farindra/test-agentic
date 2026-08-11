package httpapi

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"

	"test-agentic/internal/store"
)

func (a *API) listBots(c *fiber.Ctx) error {
	ctx, cancel := ctx15(c)
	defer cancel()
	rows, err := a.st.ListBots(ctx)
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	return c.JSON(fiber.Map{"bots": rows})
}

func (a *API) getBot(c *fiber.Ctx) error {
	ctx, cancel := ctx15(c)
	defer cancel()
	b, err := a.st.GetBot(ctx, c.Params("id"))
	if errors.Is(err, store.ErrNotFound) {
		return errJSON(c, fiber.StatusNotFound, err)
	}
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	return c.JSON(b)
}

type botReq struct {
	Name         string   `json:"name"`
	ProviderID   string   `json:"provider_id"`
	Model        string   `json:"model"`
	SystemPrompt string   `json:"system_prompt"`
	Temperature  *float64 `json:"temperature"`
	MaxTokens    *int     `json:"max_tokens"`
	IsActive     *bool    `json:"is_active"`
}

func (a *API) createBot(c *fiber.Ctx) error {
	var req botReq
	if err := c.BodyParser(&req); err != nil || strings.TrimSpace(req.Name) == "" || req.ProviderID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name & provider_id wajib"})
	}
	ctx, cancel := ctx15(c)
	defer cancel()

	b, err := a.st.CreateBot(ctx, store.Bot{
		Name: req.Name, ProviderID: req.ProviderID, Model: req.Model, SystemPrompt: req.SystemPrompt,
		Temperature: floatOr(req.Temperature, 0.7), MaxTokens: intOr(req.MaxTokens, 1024), IsActive: boolOr(req.IsActive, true),
	})
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	return c.Status(fiber.StatusCreated).JSON(b)
}

func (a *API) updateBot(c *fiber.Ctx) error {
	var req botReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "body tidak valid"})
	}
	ctx, cancel := ctx15(c)
	defer cancel()

	existing, err := a.st.GetBot(ctx, c.Params("id"))
	if errors.Is(err, store.ErrNotFound) {
		return errJSON(c, fiber.StatusNotFound, err)
	}
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}

	updated, err := a.st.UpdateBot(ctx, store.Bot{
		ID: existing.ID, Name: req.Name, ProviderID: req.ProviderID, Model: req.Model, SystemPrompt: req.SystemPrompt,
		Temperature: floatOr(req.Temperature, existing.Temperature), MaxTokens: intOr(req.MaxTokens, existing.MaxTokens),
		IsActive: boolOr(req.IsActive, existing.IsActive),
	})
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	return c.JSON(updated)
}

func (a *API) deleteBot(c *fiber.Ctx) error {
	ctx, cancel := ctx15(c)
	defer cancel()
	id := c.Params("id")

	// gateway_sessions.bot_id di DB itu ON DELETE SET NULL — tanpa pengecekan
	// ini, hapus bot bakal diem-diem ngelepas binding di sesi WA/Telegram yang
	// masih make dia, auto-reply berhenti tanpa peringatan apa pun.
	n, err := a.st.CountSessionsByBot(ctx, id)
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	if n > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": fmt.Sprintf("bot masih dipakai %d sesi WhatsApp/Telegram — lepas binding-nya dulu di menu Chat Gateway", n),
		})
	}

	if err := a.st.DeleteBot(ctx, id); errors.Is(err, store.ErrNotFound) {
		return errJSON(c, fiber.StatusNotFound, err)
	} else if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

func floatOr(v *float64, def float64) float64 {
	if v != nil {
		return *v
	}
	return def
}

func intOr(v *int, def int) int {
	if v != nil {
		return *v
	}
	return def
}

func boolOr(v *bool, def bool) bool {
	if v != nil {
		return *v
	}
	return def
}
