package httpapi

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"test-agentic/internal/store"
)

// sessionView: telegram_token TIDAK PERNAH dibalikin ke frontend — cuma
// penanda "sudah diisi" (token_set), sama seperti API key provider.
type sessionView struct {
	ID               string  `json:"id"`
	Kind             string  `json:"kind"`
	Label            string  `json:"label"`
	Status           string  `json:"status"`
	WaJID            *string `json:"wa_jid,omitempty"`
	TelegramUsername *string `json:"telegram_username,omitempty"`
	TokenSet         bool    `json:"token_set,omitempty"`
	BotID            *string `json:"bot_id,omitempty"`
	AutoReply        bool    `json:"auto_reply"`
}

func toSessionView(s store.GatewaySession) sessionView {
	return sessionView{
		ID: s.ID, Kind: string(s.Kind), Label: s.Label, Status: s.Status,
		WaJID: s.WaJID, TelegramUsername: s.TelegramUsername, TokenSet: s.TelegramToken != "",
		BotID: s.BotID, AutoReply: s.AutoReply,
	}
}

func (a *API) listSessions(kind store.SessionKind) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := ctx15(c)
		defer cancel()
		rows, err := a.st.ListSessions(ctx, kind)
		if err != nil {
			return errJSON(c, fiber.StatusInternalServerError, err)
		}
		out := make([]sessionView, 0, len(rows))
		for _, s := range rows {
			v := toSessionView(s)
			if kind == store.KindWhatsApp && a.waMgr != nil {
				v.Status = a.waMgr.Status(s.ID).State
			}
			out = append(out, v)
		}
		return c.JSON(fiber.Map{"sessions": out})
	}
}

type updateBindingReq struct {
	BotID     *string `json:"bot_id"`
	AutoReply *bool   `json:"auto_reply"`
}

func (a *API) updateSessionBinding(c *fiber.Ctx) error {
	var req updateBindingReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "body tidak valid"})
	}
	ctx, cancel := ctx15(c)
	defer cancel()

	existing, err := a.st.GetSession(ctx, c.Params("id"))
	if errors.Is(err, store.ErrNotFound) {
		return errJSON(c, fiber.StatusNotFound, err)
	}
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}

	botID := existing.BotID
	if req.BotID != nil {
		if *req.BotID == "" {
			botID = nil
		} else {
			botID = req.BotID
		}
	}
	autoReply := boolOr(req.AutoReply, existing.AutoReply)

	updated, err := a.st.UpdateSessionBinding(ctx, existing.ID, botID, autoReply)
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	return c.JSON(toSessionView(*updated))
}
