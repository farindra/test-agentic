package telegram

import (
	"context"
	"encoding/json"
	"fmt"

	"test-agentic/internal/conversation"
	"test-agentic/internal/store"
)

type Manager struct {
	st            *store.Store
	hub           *conversation.Hub
	publicBaseURL string
	webhookSecret string
}

func New(st *store.Store, hub *conversation.Hub, publicBaseURL, webhookSecret string) *Manager {
	return &Manager{st: st, hub: hub, publicBaseURL: publicBaseURL, webhookSecret: webhookSecret}
}

// Activate daftarin webhook Telegram buat sesi ini, terus simpan username
// bot (buat ditampilin di UI) dan tandain sesi connected.
func (m *Manager) Activate(ctx context.Context, sessionID string) error {
	sess, err := m.st.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("telegram: load sesi: %w", err)
	}
	if sess.Kind != store.KindTelegram {
		return fmt.Errorf("telegram: sesi %q bukan tipe telegram", sessionID)
	}
	if sess.TelegramToken == "" {
		return fmt.Errorf("telegram: token bot belum diisi")
	}

	api := newAPIClient(sess.TelegramToken)
	me, err := api.getMe(ctx)
	if err != nil {
		return fmt.Errorf("telegram: token tidak valid: %w", err)
	}
	if err := api.setWebhook(ctx, webhookURLFor(m.publicBaseURL, sessionID), m.webhookSecret); err != nil {
		return fmt.Errorf("telegram: daftar webhook gagal: %w", err)
	}
	if err := m.st.SetTelegramUsername(ctx, sessionID, me.Username); err != nil {
		return err
	}
	return m.st.SetSessionStatus(ctx, sessionID, "connected")
}

func (m *Manager) Deactivate(ctx context.Context, sessionID string) error {
	sess, err := m.st.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("telegram: load sesi: %w", err)
	}
	if sess.TelegramToken != "" {
		api := newAPIClient(sess.TelegramToken)
		if err := api.deleteWebhook(ctx); err != nil {
			return fmt.Errorf("telegram: hapus webhook gagal: %w", err)
		}
	}
	return m.st.SetSessionStatus(ctx, sessionID, "disconnected")
}

// Send ngirim pesan keluar — dipakai auto-reply DAN endpoint "kirim pesan
// test" manual dari UI, sama seperti gateway WhatsApp.
func (m *Manager) Send(ctx context.Context, sessionID, chatID, text string) error {
	sess, err := m.st.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("telegram: load sesi: %w", err)
	}
	if sess.TelegramToken == "" {
		return fmt.Errorf("telegram: sesi %q belum punya token", sessionID)
	}
	api := newAPIClient(sess.TelegramToken)
	return api.sendMessage(ctx, chatID, text)
}

type telegramUpdate struct {
	Message *struct {
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		From struct {
			FirstName string `json:"first_name"`
			Username  string `json:"username"`
		} `json:"from"`
		Text string `json:"text"`
	} `json:"message"`
}

// VerifyWebhookSecret dicek DI DEPAN, sebelum body update di-parse — Telegram
// ngirim header X-Telegram-Bot-Api-Secret-Token persis nilai yang kita kasih
// pas setWebhook, jadi ini satu-satunya cara mastiin request beneran dari
// Telegram dan bukan pihak lain yang nebak URL webhook-nya.
func (m *Manager) VerifyWebhookSecret(headerValue string) bool {
	return headerValue != "" && headerValue == m.webhookSecret
}

// HandleWebhook parse update dari Telegram lalu serahkan ke conversation.Hub
// kalau isinya pesan teks. Update lain (edited_message, callback_query, dst)
// sengaja diabaikan buat scope chatbot sederhana ini.
func (m *Manager) HandleWebhook(ctx context.Context, sessionID string, body []byte) error {
	var upd telegramUpdate
	if err := json.Unmarshal(body, &upd); err != nil {
		return fmt.Errorf("telegram: update bukan JSON valid: %w", err)
	}
	if upd.Message == nil || upd.Message.Text == "" {
		return nil
	}

	chatID := fmt.Sprintf("%d", upd.Message.Chat.ID)
	name := upd.Message.From.FirstName
	if name == "" {
		name = upd.Message.From.Username
	}

	send := func(ctx context.Context, replyText string) error {
		return m.Send(ctx, sessionID, chatID, replyText)
	}
	return m.hub.HandleIncoming(ctx, sessionID, chatID, name, upd.Message.Text, send)
}
