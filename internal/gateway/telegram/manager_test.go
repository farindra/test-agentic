package telegram

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"test-agentic/internal/bot"
	"test-agentic/internal/conversation"
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

// fakeTelegramAPI ngerekam method apa aja yang dipanggil dan balikin
// response Telegram yang wajar buat masing-masing.
func fakeTelegramAPI(t *testing.T, onSendMessage func(chatID, text string)) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/getMe"):
			json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": map[string]any{"id": 1, "username": "test_bot"}})
		case strings.HasSuffix(r.URL.Path, "/setWebhook"):
			json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": true})
		case strings.HasSuffix(r.URL.Path, "/deleteWebhook"):
			json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": true})
		case strings.HasSuffix(r.URL.Path, "/sendMessage"):
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			if onSendMessage != nil {
				onSendMessage(body["chat_id"], body["text"])
			}
			json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": map[string]any{}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	return srv
}

func TestActivateSetsWebhookAndUsername(t *testing.T) {
	srv := fakeTelegramAPI(t, nil)
	defer srv.Close()
	orig := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = orig }()

	st := newTestStore(t)
	ctx := context.Background()
	hub := conversation.New(st, bot.New(st))
	mgr := New(st, hub, "https://test-agentic.farindra.com", "webhook-secret")

	sess, err := st.CreateSession(ctx, store.GatewaySession{Kind: store.KindTelegram, Label: "Support", TelegramToken: "123:ABC", AutoReply: true})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if err := mgr.Activate(ctx, sess.ID); err != nil {
		t.Fatalf("Activate: %v", err)
	}

	got, err := st.GetSession(ctx, sess.ID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.Status != "connected" {
		t.Fatalf("expected status connected, got %q", got.Status)
	}
	if got.TelegramUsername == nil || *got.TelegramUsername != "test_bot" {
		t.Fatalf("expected username test_bot, got %+v", got.TelegramUsername)
	}
}

func TestDeactivateSetsDisconnected(t *testing.T) {
	srv := fakeTelegramAPI(t, nil)
	defer srv.Close()
	orig := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = orig }()

	st := newTestStore(t)
	ctx := context.Background()
	hub := conversation.New(st, bot.New(st))
	mgr := New(st, hub, "https://test-agentic.farindra.com", "webhook-secret")

	sess, _ := st.CreateSession(ctx, store.GatewaySession{Kind: store.KindTelegram, Label: "S", TelegramToken: "123:ABC", AutoReply: true})
	if err := mgr.Deactivate(ctx, sess.ID); err != nil {
		t.Fatalf("Deactivate: %v", err)
	}
	got, _ := st.GetSession(ctx, sess.ID)
	if got.Status != "disconnected" {
		t.Fatalf("expected disconnected, got %q", got.Status)
	}
}

func TestVerifyWebhookSecret(t *testing.T) {
	mgr := New(nil, nil, "", "the-secret")
	if !mgr.VerifyWebhookSecret("the-secret") {
		t.Fatalf("expected true buat secret yang cocok")
	}
	if mgr.VerifyWebhookSecret("salah") {
		t.Fatalf("expected false buat secret yang salah")
	}
	if mgr.VerifyWebhookSecret("") {
		t.Fatalf("expected false buat secret kosong")
	}
}

func TestHandleWebhookRoutesTextMessageAndAutoReplies(t *testing.T) {
	var sentChatID, sentText string
	srv := fakeTelegramAPI(t, func(chatID, text string) { sentChatID, sentText = chatID, text })
	defer srv.Close()
	orig := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = orig }()

	// Provider AI-nya sendiri juga di-mock lewat httptest server terpisah.
	aiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"role": "assistant", "content": "Halo dari bot"}}},
		})
	}))
	defer aiSrv.Close()

	st := newTestStore(t)
	ctx := context.Background()
	provider, err := st.CreateProvider(ctx, store.Provider{Name: "P", Type: store.ProviderCustom, BaseURL: aiSrv.URL, IsActive: true})
	if err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}
	b, err := st.CreateBot(ctx, store.Bot{Name: "Bot", ProviderID: provider.ID, IsActive: true})
	if err != nil {
		t.Fatalf("CreateBot: %v", err)
	}
	sess, err := st.CreateSession(ctx, store.GatewaySession{Kind: store.KindTelegram, Label: "S", TelegramToken: "123:ABC", AutoReply: true, BotID: &b.ID})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	hub := conversation.New(st, bot.New(st))
	mgr := New(st, hub, "https://test-agentic.farindra.com", "secret")

	update := map[string]any{
		"update_id": 1,
		"message": map[string]any{
			"chat": map[string]any{"id": 999},
			"from": map[string]any{"first_name": "Budi", "username": "budi123"},
			"text": "Halo bot",
		},
	}
	body, _ := json.Marshal(update)

	if err := mgr.HandleWebhook(ctx, sess.ID, body); err != nil {
		t.Fatalf("HandleWebhook: %v", err)
	}

	if sentChatID != "999" {
		t.Fatalf("expected reply dikirim ke chat 999, got %q", sentChatID)
	}
	if sentText != "Halo dari bot" {
		t.Fatalf("unexpected reply text: %q", sentText)
	}
}

func TestHandleWebhookIgnoresNonMessageUpdates(t *testing.T) {
	st := newTestStore(t)
	hub := conversation.New(st, bot.New(st))
	mgr := New(st, hub, "https://x", "secret")

	body := []byte(`{"update_id": 1}`)
	if err := mgr.HandleWebhook(context.Background(), "any-session", body); err != nil {
		t.Fatalf("HandleWebhook harus diam2 skip update tanpa message: %v", err)
	}
}
