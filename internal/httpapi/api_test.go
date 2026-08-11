package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v2"

	"test-agentic/internal/auth"
	"test-agentic/internal/bot"
	"test-agentic/internal/cryptutil"
	"test-agentic/internal/db"
	"test-agentic/internal/store"
)

type testEnv struct {
	app   *fiber.App
	st    *store.Store
	token string
}

func newTestEnv(t *testing.T) *testEnv {
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
	st := store.New(sqlDB, box)
	authSvc := auth.New("test-secret")
	orch := bot.New(st)

	hash, _ := auth.HashPassword("admin123")
	if _, err := st.CreateUser(context.Background(), "admin", hash); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	api := New(st, authSvc, orch, nil, nil)
	app := fiber.New()
	api.Register(app)

	token, err := authSvc.IssueToken("does-not-matter", "admin")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}

	return &testEnv{app: app, st: st, token: token}
}

func (e *testEnv) do(t *testing.T, method, path string, body any, authed bool) *http.Response {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if authed {
		req.Header.Set("Authorization", "Bearer "+e.token)
	}
	resp, err := e.app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	return resp
}

func TestLoginSuccessAndFailure(t *testing.T) {
	env := newTestEnv(t)

	resp := env.do(t, http.MethodPost, "/api/auth/login", map[string]string{"username": "admin", "password": "admin123"}, false)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var out map[string]string
	json.NewDecoder(resp.Body).Decode(&out)
	if out["token"] == "" {
		t.Fatalf("expected token di response, got %+v", out)
	}

	resp2 := env.do(t, http.MethodPost, "/api/auth/login", map[string]string{"username": "admin", "password": "salah"}, false)
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 buat password salah, got %d", resp2.StatusCode)
	}
}

func TestProtectedRouteRejectsWithoutToken(t *testing.T) {
	env := newTestEnv(t)
	resp := env.do(t, http.MethodGet, "/api/bots", nil, false)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 tanpa token, got %d", resp.StatusCode)
	}
}

func TestProviderCRUDViaHTTP(t *testing.T) {
	env := newTestEnv(t)

	createResp := env.do(t, http.MethodPost, "/api/providers", providerReq{
		Name: "OpenAI", Type: "openai", APIKey: "sk-abcd1234", DefaultModel: "gpt-4o-mini",
	}, true)
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", createResp.StatusCode)
	}
	var created providerView
	json.NewDecoder(createResp.Body).Decode(&created)
	if created.APIKeyPreview == "" || created.APIKeyPreview == "sk-abcd1234" {
		t.Fatalf("api key harusnya di-mask di response, got %+v", created)
	}
	if !created.APIKeySet {
		t.Fatalf("expected api_key_set true")
	}

	listResp := env.do(t, http.MethodGet, "/api/providers", nil, true)
	var listOut struct {
		Providers []providerView `json:"providers"`
	}
	json.NewDecoder(listResp.Body).Decode(&listOut)
	if len(listOut.Providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(listOut.Providers))
	}

	delResp := env.do(t, http.MethodDelete, "/api/providers/"+created.ID, nil, true)
	if delResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on delete, got %d", delResp.StatusCode)
	}
}

func TestBotAndPlaygroundFlowViaHTTP(t *testing.T) {
	env := newTestEnv(t)

	aiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"role": "assistant", "content": "Halo, ada yang bisa dibantu?"}}},
		})
	}))
	defer aiSrv.Close()

	provResp := env.do(t, http.MethodPost, "/api/providers", providerReq{Name: "Custom", Type: "custom", BaseURL: aiSrv.URL, IsActive: boolPtr(true)}, true)
	var prov providerView
	json.NewDecoder(provResp.Body).Decode(&prov)

	botResp := env.do(t, http.MethodPost, "/api/bots", botReq{Name: "CS", ProviderID: prov.ID}, true)
	if botResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 create bot, got %d", botResp.StatusCode)
	}
	var createdBot store.Bot
	json.NewDecoder(botResp.Body).Decode(&createdBot)

	pgResp := env.do(t, http.MethodPost, "/api/playground/sessions", createPlaygroundReq{BotID: createdBot.ID}, true)
	if pgResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 create playground session, got %d", pgResp.StatusCode)
	}
	var pgSess store.PlaygroundSession
	json.NewDecoder(pgResp.Body).Decode(&pgSess)

	chatResp := env.do(t, http.MethodPost, "/api/playground/sessions/"+pgSess.ID+"/chat", playgroundChatReq{Message: "Halo"}, true)
	if chatResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 chat, got %d", chatResp.StatusCode)
	}
	var reply store.PlaygroundMessage
	json.NewDecoder(chatResp.Body).Decode(&reply)
	if reply.Content != "Halo, ada yang bisa dibantu?" {
		t.Fatalf("unexpected reply: %q", reply.Content)
	}
}

func TestListProviderModelsViaHTTP(t *testing.T) {
	env := newTestEnv(t)

	aiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]string{{"id": "gpt-4o-mini"}, {"id": "gpt-4o"}},
		})
	}))
	defer aiSrv.Close()

	resp := env.do(t, http.MethodPost, "/api/providers/models", listModelsReq{Type: "custom", BaseURL: aiSrv.URL}, true)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var out struct {
		Models []string `json:"models"`
	}
	json.NewDecoder(resp.Body).Decode(&out)
	if len(out.Models) != 2 {
		t.Fatalf("expected 2 models, got %+v", out.Models)
	}
}

func TestListProviderModelsFallsBackToStoredKeyOnEdit(t *testing.T) {
	env := newTestEnv(t)

	var gotAuthHeader string
	aiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthHeader = r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"id": "m1"}}})
	}))
	defer aiSrv.Close()

	createResp := env.do(t, http.MethodPost, "/api/providers", providerReq{
		Name: "P", Type: "custom", BaseURL: aiSrv.URL, APIKey: "stored-secret-key",
	}, true)
	var created providerView
	json.NewDecoder(createResp.Body).Decode(&created)

	// api_key sengaja dikosongin di request, kayak yang dilakuin form edit di FE.
	resp := env.do(t, http.MethodPost, "/api/providers/models", listModelsReq{
		Type: "custom", BaseURL: aiSrv.URL, ProviderID: &created.ID,
	}, true)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if gotAuthHeader != "Bearer stored-secret-key" {
		t.Fatalf("expected fallback ke key tersimpan, got header %q", gotAuthHeader)
	}
}

func boolPtr(b bool) *bool { return &b }
