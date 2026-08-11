package aiprovider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAICompatChatSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected Authorization header: %q", got)
		}
		var body oaRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Model != "gpt-4o-mini" || len(body.Messages) != 2 {
			t.Fatalf("unexpected body: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(oaResponse{
			Model: "gpt-4o-mini",
			Choices: []struct {
				Message oaMessage `json:"message"`
			}{{Message: oaMessage{Role: "assistant", Content: "Halo juga!"}}},
		})
	}))
	defer srv.Close()

	p, err := New(Config{Kind: KindCustom, BaseURL: srv.URL, APIKey: "test-key"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	resp, err := p.Chat(context.Background(), ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []ChatMessage{
			{Role: "system", Content: "Kamu ramah."},
			{Role: "user", Content: "Halo"},
		},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Content != "Halo juga!" {
		t.Fatalf("unexpected content: %q", resp.Content)
	}
}

func TestOpenAICompatChatProviderError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "invalid api key"}})
	}))
	defer srv.Close()

	p, err := New(Config{Kind: KindCustom, BaseURL: srv.URL, APIKey: "bad-key"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = p.Chat(context.Background(), ChatRequest{Model: "m", Messages: []ChatMessage{{Role: "user", Content: "hi"}}})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "invalid api key") {
		t.Fatalf("expected error message to surface provider message, got %v", err)
	}
}

func TestOpenAICompatRequiresBaseURLForCustomAndOllama(t *testing.T) {
	if _, err := New(Config{Kind: KindCustom}); err == nil {
		t.Fatalf("expected error when base_url kosong buat custom")
	}
	if _, err := New(Config{Kind: KindOllama}); err == nil {
		t.Fatalf("expected error when base_url kosong buat ollama")
	}
}

func TestOpenAICompatDefaultsForOpenAIAndDeepSeek(t *testing.T) {
	p1, err := New(Config{Kind: KindOpenAI, APIKey: "k"})
	if err != nil {
		t.Fatalf("New openai: %v", err)
	}
	oa1 := p1.(*openAICompat)
	if oa1.baseURL != "https://api.openai.com/v1" {
		t.Fatalf("unexpected default base url: %s", oa1.baseURL)
	}

	p2, err := New(Config{Kind: KindDeepSeek, APIKey: "k"})
	if err != nil {
		t.Fatalf("New deepseek: %v", err)
	}
	oa2 := p2.(*openAICompat)
	if oa2.baseURL != "https://api.deepseek.com/v1" {
		t.Fatalf("unexpected default base url: %s", oa2.baseURL)
	}
}

func TestOpenAICompatListModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" || r.Method != http.MethodGet {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected Authorization header: %q", got)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]string{{"id": "gpt-4o-mini"}, {"id": "gpt-4o"}},
		})
	}))
	defer srv.Close()

	p, err := New(Config{Kind: KindCustom, BaseURL: srv.URL, APIKey: "test-key"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	models, err := p.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(models) != 2 || models[0] != "gpt-4o" || models[1] != "gpt-4o-mini" {
		t.Fatalf("unexpected models (harus urut alfabet): %+v", models)
	}
}

func TestOpenAICompatListModelsProviderError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "unknown endpoint"}})
	}))
	defer srv.Close()

	p, _ := New(Config{Kind: KindCustom, BaseURL: srv.URL})
	_, err := p.ListModels(context.Background())
	if err == nil || !strings.Contains(err.Error(), "unknown endpoint") {
		t.Fatalf("expected provider error to surface, got %v", err)
	}
}
