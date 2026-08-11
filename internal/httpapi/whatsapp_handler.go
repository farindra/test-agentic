package httpapi

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"test-agentic/internal/store"
)

type createSessionReq struct {
	Label string `json:"label"`
}

func (a *API) createWhatsAppSession(c *fiber.Ctx) error {
	var req createSessionReq
	_ = c.BodyParser(&req)
	ctx, cancel := ctx15(c)
	defer cancel()
	sess, err := a.st.CreateSession(ctx, store.GatewaySession{Kind: store.KindWhatsApp, Label: req.Label, AutoReply: true})
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	return c.Status(fiber.StatusCreated).JSON(toSessionView(*sess))
}

func (a *API) waQR(c *fiber.Ctx) error {
	id := c.Params("id")
	ctx15Ctx, cancel15 := ctx15(c)
	if _, err := a.st.GetSession(ctx15Ctx, id); errors.Is(err, store.ErrNotFound) {
		cancel15()
		return errJSON(c, fiber.StatusNotFound, err)
	} else if err != nil {
		cancel15()
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	cancel15()

	ctx, cancel := context.WithTimeout(c.UserContext(), 20*time.Second)
	defer cancel()
	qr, err := a.waMgr.StartPairing(ctx, id)
	if err != nil {
		return errJSON(c, fiber.StatusServiceUnavailable, err)
	}
	return c.JSON(fiber.Map{"qr": qr.Image, "ttl_sec": int(qr.TTL.Seconds())})
}

func (a *API) waStatus(c *fiber.Ctx) error {
	return c.JSON(a.waMgr.Status(c.Params("id")))
}

type waSendReq struct {
	Phone   string `json:"phone"`
	Message string `json:"message"`
}

func (a *API) waSend(c *fiber.Ctx) error {
	var req waSendReq
	if err := c.BodyParser(&req); err != nil || strings.TrimSpace(req.Phone) == "" || strings.TrimSpace(req.Message) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "phone & message wajib"})
	}
	ctx, cancel := ctx30(c)
	defer cancel()
	if err := a.waMgr.Send(ctx, c.Params("id"), req.Phone, req.Message); err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	return c.JSON(fiber.Map{"success": true})
}

func (a *API) deleteWhatsAppSession(c *fiber.Ctx) error {
	ctx, cancel := ctx15(c)
	defer cancel()
	id := c.Params("id")
	_ = a.waMgr.Disconnect(ctx, id)
	if err := a.st.DeleteSession(ctx, id); errors.Is(err, store.ErrNotFound) {
		return errJSON(c, fiber.StatusNotFound, err)
	} else if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}
