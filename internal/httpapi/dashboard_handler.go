package httpapi

import (
	"github.com/gofiber/fiber/v2"

	"test-agentic/internal/store"
)

func (a *API) dashboardSummary(c *fiber.Ctx) error {
	ctx, cancel := ctx15(c)
	defer cancel()

	providers, err := a.st.ListProviders(ctx)
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	bots, err := a.st.ListBots(ctx)
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	waSessions, err := a.st.ListSessions(ctx, store.KindWhatsApp)
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	tgSessions, err := a.st.ListSessions(ctx, store.KindTelegram)
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	conversations, err := a.st.ListConversations(ctx)
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}

	waConnected := 0
	for _, s := range waSessions {
		if a.waMgr != nil && a.waMgr.Status(s.ID).Connected {
			waConnected++
		}
	}
	tgConnected := 0
	for _, s := range tgSessions {
		if s.Status == "connected" {
			tgConnected++
		}
	}

	return c.JSON(fiber.Map{
		"providers_total":     len(providers),
		"bots_total":          len(bots),
		"whatsapp_sessions":   len(waSessions),
		"whatsapp_connected":  waConnected,
		"telegram_sessions":   len(tgSessions),
		"telegram_connected":  tgConnected,
		"conversations_total": len(conversations),
	})
}
