package store

import (
	"context"
	"path/filepath"
	"testing"

	"test-agentic/internal/cryptutil"
	"test-agentic/internal/db"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	sqlDB, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	box, err := cryptutil.New("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
	if err != nil {
		t.Fatalf("cryptutil.New: %v", err)
	}
	return New(sqlDB, box)
}

func TestUserCreateAndGet(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	u, err := s.CreateUser(ctx, "admin", "hashed-password")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	got, err := s.GetUserByUsername(ctx, "admin")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	if got.ID != u.ID || got.PasswordHash != "hashed-password" {
		t.Fatalf("unexpected user: %+v", got)
	}

	if _, err := s.GetUserByUsername(ctx, "nobody"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestProviderCRUDEncryptsAPIKey(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	p, err := s.CreateProvider(ctx, Provider{
		Name: "OpenAI Prod", Type: ProviderOpenAI, BaseURL: "https://api.openai.com/v1",
		APIKey: "sk-secret-123", DefaultModel: "gpt-4o-mini", IsActive: true,
	})
	if err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}
	if p.APIKey != "sk-secret-123" {
		t.Fatalf("expected decrypted key on create response, got %q", p.APIKey)
	}

	// api_key_enc di kolom mentah harus BEDA dari plaintext.
	var raw string
	if err := s.db.QueryRow(`SELECT api_key_enc FROM ai_providers WHERE id=?`, p.ID).Scan(&raw); err != nil {
		t.Fatalf("query raw: %v", err)
	}
	if raw == "sk-secret-123" {
		t.Fatalf("api_key_enc kesimpen plaintext, harusnya terenkripsi")
	}

	got, err := s.GetProvider(ctx, p.ID)
	if err != nil {
		t.Fatalf("GetProvider: %v", err)
	}
	if got.APIKey != "sk-secret-123" {
		t.Fatalf("roundtrip APIKey mismatch: %q", got.APIKey)
	}

	got.Name = "OpenAI Prod Updated"
	updated, err := s.UpdateProvider(ctx, *got)
	if err != nil {
		t.Fatalf("UpdateProvider: %v", err)
	}
	if updated.Name != "OpenAI Prod Updated" {
		t.Fatalf("update tidak nempel: %+v", updated)
	}

	list, err := s.ListProviders(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("ListProviders: %v %v", list, err)
	}

	if err := s.DeleteProvider(ctx, p.ID); err != nil {
		t.Fatalf("DeleteProvider: %v", err)
	}
	if _, err := s.GetProvider(ctx, p.ID); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestBotCRUD(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	p, err := s.CreateProvider(ctx, Provider{Name: "P", Type: ProviderOllama, DefaultModel: "llama3", IsActive: true})
	if err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}

	b, err := s.CreateBot(ctx, Bot{Name: "CS Bot", ProviderID: p.ID, Model: "llama3", SystemPrompt: "Kamu ramah.", Temperature: 0.5, MaxTokens: 512, IsActive: true})
	if err != nil {
		t.Fatalf("CreateBot: %v", err)
	}

	got, err := s.GetBot(ctx, b.ID)
	if err != nil || got.Name != "CS Bot" {
		t.Fatalf("GetBot: %+v %v", got, err)
	}

	if err := s.DeleteBot(ctx, b.ID); err != nil {
		t.Fatalf("DeleteBot: %v", err)
	}
	if _, err := s.GetBot(ctx, b.ID); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete bot")
	}
}

func TestSessionEncryptsTelegramToken(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	sess, err := s.CreateSession(ctx, GatewaySession{Kind: KindTelegram, Label: "Bot Support", TelegramToken: "123:ABC-token", AutoReply: true})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	var raw string
	if err := s.db.QueryRow(`SELECT telegram_token_enc FROM gateway_sessions WHERE id=?`, sess.ID).Scan(&raw); err != nil {
		t.Fatalf("query raw: %v", err)
	}
	if raw == "123:ABC-token" {
		t.Fatalf("telegram token kesimpen plaintext")
	}

	got, err := s.GetSession(ctx, sess.ID)
	if err != nil || got.TelegramToken != "123:ABC-token" {
		t.Fatalf("roundtrip token mismatch: %+v %v", got, err)
	}
}

func TestConversationAndMessageFlow(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	sess, err := s.CreateSession(ctx, GatewaySession{Kind: KindWhatsApp, Label: "WA 1", AutoReply: true})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	conv, err := s.GetOrCreateConversation(ctx, sess.ID, "628111234567", "Budi")
	if err != nil {
		t.Fatalf("GetOrCreateConversation: %v", err)
	}
	// dipanggil lagi dengan contact yang sama harus balikin baris yang SAMA, bukan bikin baru.
	conv2, err := s.GetOrCreateConversation(ctx, sess.ID, "628111234567", "Budi")
	if err != nil {
		t.Fatalf("GetOrCreateConversation (idempotent): %v", err)
	}
	if conv.ID != conv2.ID {
		t.Fatalf("GetOrCreateConversation bikin baris baru, harusnya idempotent: %s vs %s", conv.ID, conv2.ID)
	}

	if _, err := s.AddMessage(ctx, Message{ConversationID: conv.ID, Direction: "in", Sender: "user", Content: "Halo"}); err != nil {
		t.Fatalf("AddMessage in: %v", err)
	}
	if _, err := s.AddMessage(ctx, Message{ConversationID: conv.ID, Direction: "out", Sender: "bot", Content: "Halo juga!"}); err != nil {
		t.Fatalf("AddMessage out: %v", err)
	}

	msgs, err := s.ListMessages(ctx, conv.ID, 10)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Content != "Halo" || msgs[1].Content != "Halo juga!" {
		t.Fatalf("urutan pesan salah (harus kronologis): %+v", msgs)
	}

	refreshed, err := s.GetConversation(ctx, conv.ID)
	if err != nil {
		t.Fatalf("GetConversation: %v", err)
	}
	if refreshed.LastMessageAt == nil {
		t.Fatalf("last_message_at harusnya keisi setelah AddMessage")
	}

	if err := s.SetConversationAutoReply(ctx, conv.ID, false); err != nil {
		t.Fatalf("SetConversationAutoReply: %v", err)
	}
	refreshed2, _ := s.GetConversation(ctx, conv.ID)
	if refreshed2.AutoReply {
		t.Fatalf("auto_reply harusnya false setelah di-toggle")
	}
}

func TestVariables(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	if err := s.SetVariable(ctx, "jam_buka", "09.00"); err != nil {
		t.Fatalf("SetVariable: %v", err)
	}
	v, err := s.GetVariable(ctx, "jam_buka")
	if err != nil || v != "09.00" {
		t.Fatalf("GetVariable: %q %v", v, err)
	}

	if err := s.SetVariable(ctx, "jam_buka", "10.00"); err != nil {
		t.Fatalf("SetVariable update: %v", err)
	}
	v, _ = s.GetVariable(ctx, "jam_buka")
	if v != "10.00" {
		t.Fatalf("expected updated value, got %q", v)
	}

	missing, err := s.GetVariable(ctx, "not-set")
	if err != nil || missing != "" {
		t.Fatalf("expected empty string for missing key, got %q %v", missing, err)
	}

	all, err := s.ListVariables(ctx)
	if err != nil || all["jam_buka"] != "10.00" {
		t.Fatalf("ListVariables: %+v %v", all, err)
	}

	if err := s.DeleteVariable(ctx, "jam_buka"); err != nil {
		t.Fatalf("DeleteVariable: %v", err)
	}
	all, _ = s.ListVariables(ctx)
	if _, ok := all["jam_buka"]; ok {
		t.Fatalf("jam_buka harusnya udah kehapus, masih ada: %+v", all)
	}
}
