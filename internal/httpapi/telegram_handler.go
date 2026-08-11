package httpapi

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"

	"test-agentic/internal/store"
)

type createTelegramReq struct {
	Label string `json:"label"`
	Token string `json:"token"`
}

func (a *API) createTelegramSession(c *fiber.Ctx) error {
	var req createTelegramReq
	if err := c.BodyParser(&req); err != nil || strings.TrimSpace(req.Token) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "token wajib"})
	}
	ctx, cancel := ctx15(c)
	defer cancel()
	sess, err := a.st.CreateSession(ctx, store.GatewaySession{
		Kind: store.KindTelegram, Label: req.Label, TelegramToken: req.Token, AutoReply: true,
	})
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	return c.Status(fiber.StatusCreated).JSON(toSessionView(*sess))
}

func (a *API) telegramActivate(c *fiber.Ctx) error {
	ctx, cancel := ctx30(c)
	defer cancel()
	if err := a.tgMgr.Activate(ctx, c.Params("id")); err != nil {
		// 400, bukan 502: Cloudflare di depan origin nge-intercept status 5xx
		// dan nimpa body-nya sama halaman error generik sendiri — pesan error
		// aslinya (mis. "token tidak valid") jadi nggak pernah sampai ke user.
		return errJSON(c, fiber.StatusBadRequest, err)
	}
	return c.JSON(fiber.Map{"success": true})
}

func (a *API) telegramDeactivate(c *fiber.Ctx) error {
	ctx, cancel := ctx30(c)
	defer cancel()
	if err := a.tgMgr.Deactivate(ctx, c.Params("id")); err != nil {
		return errJSON(c, fiber.StatusBadRequest, err)
	}
	return c.JSON(fiber.Map{"success": true})
}

type telegramSendReq struct {
	ChatID  string `json:"chat_id"`
	Message string `json:"message"`
}

func (a *API) telegramSend(c *fiber.Ctx) error {
	var req telegramSendReq
	if err := c.BodyParser(&req); err != nil || req.ChatID == "" || strings.TrimSpace(req.Message) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "chat_id & message wajib"})
	}
	ctx, cancel := ctx30(c)
	defer cancel()
	if err := a.tgMgr.Send(ctx, c.Params("id"), req.ChatID, req.Message); err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	return c.JSON(fiber.Map{"success": true})
}

func (a *API) deleteTelegramSession(c *fiber.Ctx) error {
	ctx, cancel := ctx30(c)
	defer cancel()
	id := c.Params("id")
	_ = a.tgMgr.Deactivate(ctx, id)
	if err := a.st.DeleteSession(ctx, id); errors.Is(err, store.ErrNotFound) {
		return errJSON(c, fiber.StatusNotFound, err)
	} else if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

// telegramWebhook: endpoint publik yang dipanggil Telegram, DIVERIFIKASI
// lewat header secret token — bukan lewat JWT auth middleware kayak endpoint
// lain, karena yang manggil ini Telegram, bukan admin yang login.
func (a *API) telegramWebhook(c *fiber.Ctx) error {
	secret := c.Get("X-Telegram-Bot-Api-Secret-Token")
	if !a.tgMgr.VerifyWebhookSecret(secret) {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	ctx, cancel := ctx30(c)
	defer cancel()
	if err := a.tgMgr.HandleWebhook(ctx, c.Params("id"), c.Body()); err != nil {
		// Tetep 200 ke Telegram walau proses internal gagal — response
		// non-2xx bikin Telegram nge-retry update yang sama berkali-kali.
		return c.JSON(fiber.Map{"ok": false})
	}
	return c.JSON(fiber.Map{"ok": true})
}
