// Package httpapi mendaftarkan seluruh route REST (/api/...) dan webhook
// publik, lalu nyambungin ke package domain (store, bot, gateway) di
// belakangnya. Semua handler HTTP hidup di sini; logic bisnis-nya sendiri
// TIDAK — biar httpapi gampang dites/diganti tanpa nyentuh logic.
package httpapi

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"

	"test-agentic/internal/auth"
	"test-agentic/internal/bot"
	"test-agentic/internal/gateway/telegram"
	"test-agentic/internal/gateway/whatsapp"
	"test-agentic/internal/store"
)

type API struct {
	st      *store.Store
	authSvc *auth.Auth
	orch    *bot.Orchestrator
	waMgr   *whatsapp.Manager
	tgMgr   *telegram.Manager
}

func New(st *store.Store, authSvc *auth.Auth, orch *bot.Orchestrator, waMgr *whatsapp.Manager, tgMgr *telegram.Manager) *API {
	return &API{st: st, authSvc: authSvc, orch: orch, waMgr: waMgr, tgMgr: tgMgr}
}

func (a *API) Register(app *fiber.App) {
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Webhook Telegram: publik (Telegram yang manggil), diverifikasi lewat
	// secret token di header, BUKAN lewat middleware JWT.
	app.Post("/api/gateway/telegram/webhook/:id", a.telegramWebhook)
	app.Post("/api/auth/login", a.login)

	api := app.Group("/api", auth.RequireAuth(a.authSvc))

	api.Get("/auth/me", a.me)

	providers := api.Group("/providers")
	providers.Get("", a.listProviders)
	providers.Post("", a.createProvider)
	providers.Get("/:id", a.getProvider)
	providers.Put("/:id", a.updateProvider)
	providers.Delete("/:id", a.deleteProvider)
	providers.Post("/:id/test", a.testProvider)

	bots := api.Group("/bots")
	bots.Get("", a.listBots)
	bots.Post("", a.createBot)
	bots.Get("/:id", a.getBot)
	bots.Put("/:id", a.updateBot)
	bots.Delete("/:id", a.deleteBot)

	pg := api.Group("/playground")
	pg.Get("/sessions", a.listPlaygroundSessions)
	pg.Post("/sessions", a.createPlaygroundSession)
	pg.Get("/sessions/:id/messages", a.listPlaygroundMessages)
	pg.Post("/sessions/:id/chat", a.playgroundChat)

	wa := api.Group("/gateway/whatsapp/sessions")
	wa.Get("", a.listSessions(store.KindWhatsApp))
	wa.Post("", a.createWhatsAppSession)
	wa.Get("/:id/qr", a.waQR)
	wa.Get("/:id/status", a.waStatus)
	wa.Post("/:id/send", a.waSend)
	wa.Patch("/:id", a.updateSessionBinding)
	wa.Delete("/:id", a.deleteWhatsAppSession)

	tg := api.Group("/gateway/telegram/sessions")
	tg.Get("", a.listSessions(store.KindTelegram))
	tg.Post("", a.createTelegramSession)
	tg.Post("/:id/activate", a.telegramActivate)
	tg.Post("/:id/deactivate", a.telegramDeactivate)
	tg.Post("/:id/send", a.telegramSend)
	tg.Patch("/:id", a.updateSessionBinding)
	tg.Delete("/:id", a.deleteTelegramSession)

	conv := api.Group("/conversations")
	conv.Get("", a.listConversations)
	conv.Get("/:id/messages", a.listConversationMessages)
	conv.Patch("/:id", a.updateConversation)

	api.Get("/dashboard/summary", a.dashboardSummary)

	settings := api.Group("/settings")
	settings.Get("", a.listSettings)
	settings.Put("", a.updateSettings)
}

func ctx15(c *fiber.Ctx) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.UserContext(), 15*time.Second)
}

func ctx30(c *fiber.Ctx) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.UserContext(), 30*time.Second)
}

func errJSON(c *fiber.Ctx, status int, err error) error {
	return c.Status(status).JSON(fiber.Map{"error": err.Error()})
}

func (a *API) me(c *fiber.Ctx) error {
	userID, _ := c.Locals(auth.LocalsUserIDKey).(string)
	ctx, cancel := ctx15(c)
	defer cancel()
	u, err := a.st.GetUserByID(ctx, userID)
	if err != nil {
		return errJSON(c, fiber.StatusNotFound, err)
	}
	return c.JSON(fiber.Map{"id": u.ID, "username": u.Username})
}
