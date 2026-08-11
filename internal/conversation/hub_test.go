package conversation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"test-agentic/internal/bot"
	"test-agentic/internal/cryptutil"
	"test-agentic/internal/db"
	"test-agentic/internal/store"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	box, err := cryptutil.New("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
	if err != nil {
		t.Fatalf("cryptutil.New: %v", err)
	}
	return store.New(sqlDB, box)
}

func newBotWithReply(t *testing.T, st *store.Store, replyText string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"role": "assistant", "content": replyText}}},
		})
	}))
	t.Cleanup(srv.Close)

	ctx := context.Background()
	p, err := st.CreateProvider(ctx, store.Provider{Name: "P", Type: store.ProviderCustom, BaseURL: srv.URL, IsActive: true})
	if err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}
	b, err := st.CreateBot(ctx, store.Bot{Name: "Bot", ProviderID: p.ID, IsActive: true})
	if err != nil {
		t.Fatalf("CreateBot: %v", err)
	}
	return b.ID
}

func TestHandleIncomingAutoRepliesWhenBound(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	botID := newBotWithReply(t, st, "Balasan otomatis")

	sess, err := st.CreateSession(ctx, store.GatewaySession{Kind: store.KindWhatsApp, Label: "WA", AutoReply: true, BotID: &botID})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	var sentText string
	sendCalled := false
	send := func(ctx context.Context, text string) error {
		sendCalled = true
		sentText = text
		return nil
	}

	hub := New(st, bot.New(st))
	if err := hub.HandleIncoming(ctx, sess.ID, "628111", "Budi", "Halo", send); err != nil {
		t.Fatalf("HandleIncoming: %v", err)
	}

	if !sendCalled {
		t.Fatalf("send seharusnya dipanggil karena auto_reply aktif dan bot ter-binding")
	}
	if sentText != "Balasan otomatis" {
		t.Fatalf("unexpected sent text: %q", sentText)
	}

	conv, err := st.GetOrCreateConversation(ctx, sess.ID, "628111", "Budi")
	if err != nil {
		t.Fatalf("GetOrCreateConversation: %v", err)
	}
	msgs, err := st.ListMessages(ctx, conv.ID, 10)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 pesan (in+out) tercatat, got %d", len(msgs))
	}
	if msgs[0].Direction != "in" || msgs[1].Direction != "out" {
		t.Fatalf("urutan direction salah: %+v", msgs)
	}
}

func TestHandleIncomingSkipsReplyWhenSessionAutoReplyOff(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	botID := newBotWithReply(t, st, "harusnya tidak terpanggil")

	sess, err := st.CreateSession(ctx, store.GatewaySession{Kind: store.KindWhatsApp, Label: "WA", AutoReply: false, BotID: &botID})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	sendCalled := false
	send := func(ctx context.Context, text string) error { sendCalled = true; return nil }

	hub := New(st, bot.New(st))
	if err := hub.HandleIncoming(ctx, sess.ID, "628111", "Budi", "Halo", send); err != nil {
		t.Fatalf("HandleIncoming: %v", err)
	}
	if sendCalled {
		t.Fatalf("send seharusnya TIDAK dipanggil karena session auto_reply mati")
	}

	conv, _ := st.GetOrCreateConversation(ctx, sess.ID, "628111", "Budi")
	msgs, _ := st.ListMessages(ctx, conv.ID, 10)
	if len(msgs) != 1 || msgs[0].Direction != "in" {
		t.Fatalf("pesan masuk tetap harus tercatat meski auto_reply mati: %+v", msgs)
	}
}

func TestHandleIncomingSkipsReplyWhenNoBotBound(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	sess, err := st.CreateSession(ctx, store.GatewaySession{Kind: store.KindWhatsApp, Label: "WA", AutoReply: true, BotID: nil})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	sendCalled := false
	send := func(ctx context.Context, text string) error { sendCalled = true; return nil }

	hub := New(st, bot.New(st))
	if err := hub.HandleIncoming(ctx, sess.ID, "628111", "Budi", "Halo", send); err != nil {
		t.Fatalf("HandleIncoming: %v", err)
	}
	if sendCalled {
		t.Fatalf("send seharusnya TIDAK dipanggil karena belum ada bot yang di-binding")
	}
}

func TestHandleIncomingProviderErrorDoesNotFailButLogsMessage(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	// provider custom tanpa base_url valid -> Chat bakal gagal
	p, err := st.CreateProvider(ctx, store.Provider{Name: "Broken", Type: store.ProviderCustom, BaseURL: "http://127.0.0.1:1", IsActive: true})
	if err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}
	b, err := st.CreateBot(ctx, store.Bot{Name: "Bot", ProviderID: p.ID, IsActive: true})
	if err != nil {
		t.Fatalf("CreateBot: %v", err)
	}
	sess, err := st.CreateSession(ctx, store.GatewaySession{Kind: store.KindWhatsApp, Label: "WA", AutoReply: true, BotID: &b.ID})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	sendCalled := false
	send := func(ctx context.Context, text string) error { sendCalled = true; return nil }

	hub := New(st, bot.New(st))
	if err := hub.HandleIncoming(ctx, sess.ID, "628111", "Budi", "Halo", send); err != nil {
		t.Fatalf("HandleIncoming seharusnya tidak mengembalikan error walau provider gagal: %v", err)
	}
	if sendCalled {
		t.Fatalf("send seharusnya tidak dipanggil kalau provider gagal")
	}

	conv, _ := st.GetOrCreateConversation(ctx, sess.ID, "628111", "Budi")
	msgs, _ := st.ListMessages(ctx, conv.ID, 10)
	if len(msgs) != 1 || msgs[0].Direction != "in" {
		t.Fatalf("pesan masuk tetap harus tercatat walau bot gagal balas: %+v", msgs)
	}
}

func TestHandleIncomingSkipsReplyWhenMessageTooLong(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	botID := newBotWithReply(t, st, "harusnya tidak terpanggil")

	sess, err := st.CreateSession(ctx, store.GatewaySession{Kind: store.KindWhatsApp, Label: "WA", AutoReply: true, BotID: &botID})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	sendCalled := false
	send := func(ctx context.Context, text string) error { sendCalled = true; return nil }

	longText := strings.Repeat("a", maxMessageLength+1)
	hub := New(st, bot.New(st))
	if err := hub.HandleIncoming(ctx, sess.ID, "628111", "Budi", longText, send); err != nil {
		t.Fatalf("HandleIncoming: %v", err)
	}
	if sendCalled {
		t.Fatalf("send seharusnya TIDAK dipanggil karena pesan melebihi batas panjang")
	}

	conv, _ := st.GetOrCreateConversation(ctx, sess.ID, "628111", "Budi")
	msgs, _ := st.ListMessages(ctx, conv.ID, 10)
	if len(msgs) != 1 || msgs[0].Direction != "in" {
		t.Fatalf("pesan yang kepanjangan tetap harus tercatat: %+v", msgs)
	}
}

func TestHandleIncomingSkipsReplyWhenRateLimited(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	botID := newBotWithReply(t, st, "balasan")

	sess, err := st.CreateSession(ctx, store.GatewaySession{Kind: store.KindWhatsApp, Label: "WA", AutoReply: true, BotID: &botID})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	sendCount := 0
	send := func(ctx context.Context, text string) error { sendCount++; return nil }

	hub := New(st, bot.New(st))
	for i := 0; i < rateLimitMax; i++ {
		if err := hub.HandleIncoming(ctx, sess.ID, "628111", "Budi", "Halo", send); err != nil {
			t.Fatalf("HandleIncoming ke-%d: %v", i+1, err)
		}
	}
	if sendCount != rateLimitMax {
		t.Fatalf("expected %d balasan sebelum kena limit, got %d", rateLimitMax, sendCount)
	}

	// pesan berikutnya, masih dalam window yang sama, harusnya kena limit
	if err := hub.HandleIncoming(ctx, sess.ID, "628111", "Budi", "Halo lagi", send); err != nil {
		t.Fatalf("HandleIncoming: %v", err)
	}
	if sendCount != rateLimitMax {
		t.Fatalf("send seharusnya TIDAK bertambah setelah kena rate limit, got count=%d", sendCount)
	}

	// kontak lain gak boleh kepengaruh limit punya "628111"
	if err := hub.HandleIncoming(ctx, sess.ID, "628999", "Ani", "Halo juga", send); err != nil {
		t.Fatalf("HandleIncoming kontak lain: %v", err)
	}
	if sendCount != rateLimitMax+1 {
		t.Fatalf("kontak lain seharusnya tetap dibales, got count=%d", sendCount)
	}
}
