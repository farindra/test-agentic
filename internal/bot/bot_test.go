package bot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

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

func TestOrchestratorReplyIncludesSystemPromptAndHistory(t *testing.T) {
	var capturedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedBody)
		json.NewEncoder(w).Encode(map[string]any{
			"model":   "test-model",
			"choices": []map[string]any{{"message": map[string]string{"role": "assistant", "content": "Jawaban bot"}}},
		})
	}))
	defer srv.Close()

	st := newTestStore(t)
	ctx := context.Background()

	p, err := st.CreateProvider(ctx, store.Provider{Name: "Custom", Type: store.ProviderCustom, BaseURL: srv.URL, APIKey: "k", DefaultModel: "test-model", IsActive: true})
	if err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}
	b, err := st.CreateBot(ctx, store.Bot{Name: "CS", ProviderID: p.ID, SystemPrompt: "Kamu ramah.", Temperature: 0.3, MaxTokens: 100, IsActive: true})
	if err != nil {
		t.Fatalf("CreateBot: %v", err)
	}

	orch := New(st)
	history := []store.Message{
		{Sender: "user", Content: "Halo"},
		{Sender: "bot", Content: "Hai, ada yang bisa dibantu?"},
	}
	reply, err := orch.Reply(ctx, b.ID, history, "Jam buka kapan?")
	if err != nil {
		t.Fatalf("Reply: %v", err)
	}
	if reply != "Jawaban bot" {
		t.Fatalf("unexpected reply: %q", reply)
	}

	msgs, _ := capturedBody["messages"].([]any)
	if len(msgs) != 4 { // system + 2 history + pesan baru
		t.Fatalf("expected 4 messages dikirim ke provider, got %d: %+v", len(msgs), msgs)
	}
	first := msgs[0].(map[string]any)
	firstContent, _ := first["content"].(string)
	if first["role"] != "system" || !strings.Contains(firstContent, promptGuard) || !strings.Contains(firstContent, "Kamu ramah.") {
		t.Fatalf("system message harus gabungan promptGuard + system prompt bot: %+v", first)
	}
	last := msgs[3].(map[string]any)
	if last["role"] != "user" || last["content"] != "Jam buka kapan?" {
		t.Fatalf("pesan baru tidak ditaruh di akhir: %+v", last)
	}
}

func TestOrchestratorReplyAlwaysIncludesPromptGuardEvenWithoutCustomPrompt(t *testing.T) {
	var capturedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedBody)
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"role": "assistant", "content": "ok"}}},
		})
	}))
	defer srv.Close()

	st := newTestStore(t)
	ctx := context.Background()
	p, _ := st.CreateProvider(ctx, store.Provider{Name: "P", Type: store.ProviderCustom, BaseURL: srv.URL, IsActive: true})
	b, _ := st.CreateBot(ctx, store.Bot{Name: "Bot", ProviderID: p.ID, SystemPrompt: "", IsActive: true})

	orch := New(st)
	if _, err := orch.Reply(ctx, b.ID, nil, "Halo"); err != nil {
		t.Fatalf("Reply: %v", err)
	}

	msgs, _ := capturedBody["messages"].([]any)
	first := msgs[0].(map[string]any)
	content, _ := first["content"].(string)
	if content != promptGuard {
		t.Fatalf("bot tanpa system_prompt custom tetap harus dapet promptGuard doang: %q", content)
	}
}

func TestOrchestratorReplySubstitutesVariablesInSystemPrompt(t *testing.T) {
	var capturedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedBody)
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"role": "assistant", "content": "ok"}}},
		})
	}))
	defer srv.Close()

	st := newTestStore(t)
	ctx := context.Background()
	if err := st.SetVariable(ctx, "jam_buka", "09.00 - 17.00"); err != nil {
		t.Fatalf("SetVariable: %v", err)
	}
	p, _ := st.CreateProvider(ctx, store.Provider{Name: "P", Type: store.ProviderCustom, BaseURL: srv.URL, IsActive: true})
	b, _ := st.CreateBot(ctx, store.Bot{
		Name: "Bot", ProviderID: p.ID, IsActive: true,
		SystemPrompt: "Toko kami buka jam {{jam_buka}}. Placeholder gak dikenal: {{tidak_ada}}.",
	})

	orch := New(st)
	if _, err := orch.Reply(ctx, b.ID, nil, "Halo"); err != nil {
		t.Fatalf("Reply: %v", err)
	}

	msgs, _ := capturedBody["messages"].([]any)
	content, _ := msgs[0].(map[string]any)["content"].(string)
	if !strings.Contains(content, "Toko kami buka jam 09.00 - 17.00.") {
		t.Fatalf("variable {{jam_buka}} harusnya tersubstitusi: %q", content)
	}
	if !strings.Contains(content, "{{tidak_ada}}") {
		t.Fatalf("placeholder yang gak dikenal harusnya dibiarin apa adanya: %q", content)
	}
}

func TestOrchestratorReplyRejectsInactiveBot(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	p, _ := st.CreateProvider(ctx, store.Provider{Name: "P", Type: store.ProviderCustom, BaseURL: "http://example.invalid", IsActive: true})
	b, _ := st.CreateBot(ctx, store.Bot{Name: "Nonaktif", ProviderID: p.ID, IsActive: false})

	orch := New(st)
	_, err := orch.Reply(ctx, b.ID, nil, "halo")
	if err == nil || !strings.Contains(err.Error(), "nonaktif") {
		t.Fatalf("expected error nonaktif, got %v", err)
	}
}

func TestOrchestratorReplyRejectsInactiveProvider(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	p, _ := st.CreateProvider(ctx, store.Provider{Name: "P", Type: store.ProviderCustom, BaseURL: "http://example.invalid", IsActive: false})
	b, _ := st.CreateBot(ctx, store.Bot{Name: "Bot", ProviderID: p.ID, IsActive: true})

	orch := New(st)
	_, err := orch.Reply(ctx, b.ID, nil, "halo")
	if err == nil || !strings.Contains(err.Error(), "nonaktif") {
		t.Fatalf("expected error provider nonaktif, got %v", err)
	}
}

func TestOrchestratorTestProvider(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"role": "assistant", "content": "pong"}}},
		})
	}))
	defer srv.Close()

	st := newTestStore(t)
	ctx := context.Background()
	p, _ := st.CreateProvider(ctx, store.Provider{Name: "P", Type: store.ProviderCustom, BaseURL: srv.URL, DefaultModel: "m", IsActive: true})

	orch := New(st)
	reply, err := orch.TestProvider(ctx, p.ID)
	if err != nil {
		t.Fatalf("TestProvider: %v", err)
	}
	if reply != "pong" {
		t.Fatalf("unexpected reply: %q", reply)
	}
}

func TestSubstituteVariables(t *testing.T) {
	vars := map[string]string{"nama_toko": "Toko Maju", "jam_buka": "09.00"}

	cases := []struct {
		name   string
		prompt string
		want   string
	}{
		{"basic", "Selamat datang di {{nama_toko}}.", "Selamat datang di Toko Maju."},
		{"dengan spasi di kurung", "Buka jam {{ jam_buka }}.", "Buka jam 09.00."},
		{"variable ganda", "{{nama_toko}} buka jam {{jam_buka}}.", "Toko Maju buka jam 09.00."},
		{"key gak dikenal dibiarin", "{{tidak_ada}}", "{{tidak_ada}}"},
		{"tanpa placeholder", "teks biasa", "teks biasa"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := substituteVariables(c.prompt, vars)
			if got != c.want {
				t.Fatalf("substituteVariables(%q) = %q, want %q", c.prompt, got, c.want)
			}
		})
	}
}

func TestSubstituteVariablesEmptyMapReturnsPromptUnchanged(t *testing.T) {
	got := substituteVariables("halo {{apa}}", nil)
	if got != "halo {{apa}}" {
		t.Fatalf("expected prompt gak berubah kalau vars kosong, got %q", got)
	}
}
