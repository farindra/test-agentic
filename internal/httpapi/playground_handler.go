package httpapi

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"test-agentic/internal/store"
)

func (a *API) listPlaygroundSessions(c *fiber.Ctx) error {
	ctx, cancel := ctx15(c)
	defer cancel()
	rows, err := a.st.ListPlaygroundSessions(ctx)
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	return c.JSON(fiber.Map{"sessions": rows})
}

type createPlaygroundReq struct {
	BotID string `json:"bot_id"`
	Title string `json:"title"`
}

func (a *API) createPlaygroundSession(c *fiber.Ctx) error {
	var req createPlaygroundReq
	if err := c.BodyParser(&req); err != nil || req.BotID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "bot_id wajib"})
	}
	ctx, cancel := ctx15(c)
	defer cancel()
	if req.Title == "" {
		req.Title = "Percakapan baru"
	}
	ps, err := a.st.CreatePlaygroundSession(ctx, req.BotID, req.Title)
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	return c.Status(fiber.StatusCreated).JSON(ps)
}

func (a *API) listPlaygroundMessages(c *fiber.Ctx) error {
	ctx, cancel := ctx15(c)
	defer cancel()
	rows, err := a.st.ListPlaygroundMessages(ctx, c.Params("id"))
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	return c.JSON(fiber.Map{"messages": rows})
}

type playgroundChatReq struct {
	Message string `json:"message"`
}

func (a *API) playgroundChat(c *fiber.Ctx) error {
	var req playgroundChatReq
	if err := c.BodyParser(&req); err != nil || strings.TrimSpace(req.Message) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "message wajib"})
	}
	sessionID := c.Params("id")

	ctx, cancel := ctx30(c)
	defer cancel()

	sessions, err := a.st.ListPlaygroundSessions(ctx)
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	var botID string
	found := false
	for _, s := range sessions {
		if s.ID == sessionID {
			botID, found = s.BotID, true
			break
		}
	}
	if !found {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "sesi playground tidak ditemukan"})
	}

	prevMsgs, err := a.st.ListPlaygroundMessages(ctx, sessionID)
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	history := make([]store.Message, 0, len(prevMsgs))
	for _, m := range prevMsgs {
		sender := "user"
		if m.Role == "assistant" {
			sender = "bot"
		}
		history = append(history, store.Message{Sender: sender, Content: m.Content})
	}

	if _, err := a.st.AddPlaygroundMessage(ctx, sessionID, "user", req.Message); err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}

	reply, err := a.orch.Reply(ctx, botID, history, req.Message)
	if err != nil {
		return errJSON(c, fiber.StatusBadGateway, err)
	}

	out, err := a.st.AddPlaygroundMessage(ctx, sessionID, "assistant", reply)
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	return c.JSON(out)
}
