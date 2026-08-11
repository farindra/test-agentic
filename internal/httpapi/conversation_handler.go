package httpapi

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"test-agentic/internal/store"
)

func (a *API) listConversations(c *fiber.Ctx) error {
	ctx, cancel := ctx15(c)
	defer cancel()
	rows, err := a.st.ListConversations(ctx)
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	return c.JSON(fiber.Map{"conversations": rows})
}

func (a *API) listConversationMessages(c *fiber.Ctx) error {
	ctx, cancel := ctx15(c)
	defer cancel()
	rows, err := a.st.ListMessages(ctx, c.Params("id"), 100)
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	return c.JSON(fiber.Map{"messages": rows})
}

type updateConversationReq struct {
	AutoReply *bool `json:"auto_reply"`
}

func (a *API) updateConversation(c *fiber.Ctx) error {
	var req updateConversationReq
	if err := c.BodyParser(&req); err != nil || req.AutoReply == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "auto_reply wajib"})
	}
	ctx, cancel := ctx15(c)
	defer cancel()
	if err := a.st.SetConversationAutoReply(ctx, c.Params("id"), *req.AutoReply); errors.Is(err, store.ErrNotFound) {
		return errJSON(c, fiber.StatusNotFound, err)
	} else if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	conv, err := a.st.GetConversation(ctx, c.Params("id"))
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	return c.JSON(conv)
}
