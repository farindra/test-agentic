// Package conversation adalah titik temu semua pesan masuk dari gateway mana
// pun (WhatsApp, Telegram, dst nanti): log ke inbox, putuskan apakah harus
// dibales otomatis oleh bot yang di-binding, lalu serahkan pengiriman balasan
// ke fungsi transport spesifik gateway lewat callback Sender.
package conversation

import (
	"context"
	"fmt"
	"log"
	"time"
	"unicode/utf8"

	"test-agentic/internal/bot"
	"test-agentic/internal/store"
)

const historyWindow = 20

// maxMessageLength ngebatesin jumlah karakter pesan masuk yang diproses ke
// AI provider. Bukan cuma soal biaya (tiap karakter = token = biaya) — pesan
// raksasa juga cara gampang buat nge-flood dengan satu kali kirim.
const maxMessageLength = 4000

// Rate limit spam per kontak: maksimal rateLimitMax pesan dalam
// rateLimitWindow. Bot tetap CATET pesan yang kena limit (buat audit di
// inbox), cuma gak dibales otomatis — biar spam gak nyedot kuota biaya
// provider AI berkali-kali dalam waktu singkat.
const (
	rateLimitWindow = time.Minute
	rateLimitMax    = 8
)

type Hub struct {
	st      *store.Store
	orch    *bot.Orchestrator
	limiter *rateLimiter
}

func New(st *store.Store, orch *bot.Orchestrator) *Hub {
	return &Hub{st: st, orch: orch, limiter: newRateLimiter(rateLimitWindow, rateLimitMax)}
}

// Sender ngirim teks balasan lewat transport gateway asal pesan (whatsmeow
// SendMessage, Telegram sendMessage API, dst).
type Sender func(ctx context.Context, text string) error

// HandleIncoming: dipanggil event handler gateway tiap ada pesan masuk.
// Pesan selalu dicatat ke inbox terlebih dulu (buat human-review) sebelum
// dipertimbangkan buat auto-reply — jadi log tetep lengkap walau bot lagi
// off atau providernya lagi error.
func (h *Hub) HandleIncoming(ctx context.Context, sessionID, contactID, contactName, text string, send Sender) error {
	sess, err := h.st.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("conversation: load session: %w", err)
	}

	conv, err := h.st.GetOrCreateConversation(ctx, sessionID, contactID, contactName)
	if err != nil {
		return fmt.Errorf("conversation: get/create conversation: %w", err)
	}

	history, err := h.st.ListMessages(ctx, conv.ID, historyWindow)
	if err != nil {
		return fmt.Errorf("conversation: load history: %w", err)
	}

	if _, err := h.st.AddMessage(ctx, store.Message{ConversationID: conv.ID, Direction: "in", Sender: "user", Content: text}); err != nil {
		return fmt.Errorf("conversation: log pesan masuk: %w", err)
	}

	if !sess.AutoReply || !conv.AutoReply || sess.BotID == nil {
		return nil // sengaja diem: sesi/percakapan lagi handover ke manusia, atau belum ada bot di-binding
	}

	if utf8.RuneCountInString(text) > maxMessageLength {
		log.Printf("conversation: pesan dari %s (sesi %s) %d karakter, lewat batas %d — auto-reply dilewati", contactID, sessionID, utf8.RuneCountInString(text), maxMessageLength)
		return nil
	}

	if !h.limiter.allow(sessionID + ":" + contactID) {
		log.Printf("conversation: rate limit kena buat %s (sesi %s) — auto-reply dilewati", contactID, sessionID)
		return nil
	}

	replyText, err := h.orch.Reply(ctx, *sess.BotID, history, text)
	if err != nil {
		// Kegagalan panggil AI provider TIDAK menghentikan alur — pesan user
		// tetap sudah tercatat di inbox dan bisa dibales manual dari sana.
		log.Printf("conversation: bot reply gagal buat sesi %s: %v", sessionID, err)
		return nil
	}

	if err := send(ctx, replyText); err != nil {
		return fmt.Errorf("conversation: kirim balasan gagal: %w", err)
	}

	if _, err := h.st.AddMessage(ctx, store.Message{ConversationID: conv.ID, Direction: "out", Sender: "bot", Content: replyText}); err != nil {
		return fmt.Errorf("conversation: log pesan keluar: %w", err)
	}
	return nil
}
