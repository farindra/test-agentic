package whatsapp

import (
	"context"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
)

// handleIncomingMessage: dipanggil di goroutine sendiri dari handler().
// Grup dan pesan dari diri sendiri sengaja dilewati — scope chatbot ini buat
// percakapan personal 1:1 sama kontak, bukan moderasi grup.
func (m *Manager) handleIncomingMessage(sessionID string, evt *events.Message) {
	if evt.Info.IsFromMe || evt.Info.IsGroup {
		return
	}
	text := extractText(evt.Message)
	if text == "" {
		return
	}

	contact := evt.Info.Sender.User
	name := evt.Info.PushName

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	send := func(ctx context.Context, replyText string) error {
		return m.Send(ctx, sessionID, contact, replyText)
	}

	if err := m.hub.HandleIncoming(ctx, sessionID, contact, name, text, send); err != nil {
		m.log.Errorf("whatsapp: proses pesan masuk sesi %s: %v", sessionID, err)
	}
}

func extractText(msg *waE2E.Message) string {
	if msg == nil {
		return ""
	}
	if c := msg.GetConversation(); c != "" {
		return c
	}
	if ext := msg.GetExtendedTextMessage(); ext != nil {
		return ext.GetText()
	}
	return ""
}
