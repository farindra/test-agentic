package aiprovider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGeminiChatSuccessAndSystemInstruction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "gemini-1.5-flash:generateContent") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("key") != "gem-key" {
			t.Fatalf("api key tidak dikirim di query: %s", r.URL.RawQuery)
		}
		var body geminiRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body.SystemInstruction == nil || body.SystemInstruction.Parts[0].Text != "Kamu ramah." {
			t.Fatalf("system instruction tidak sesuai: %+v", body.SystemInstruction)
		}
		if len(body.Contents) != 1 || body.Contents[0].Role != "user" {
			t.Fatalf("contents tidak sesuai: %+v", body.Contents)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{
				{"content": map[string]any{"parts": []map[string]string{{"text": "Halo dari Gemini"}}}},
			},
		})
	}))
	defer srv.Close()

	p, err := New(Config{Kind: KindGemini, BaseURL: srv.URL, APIKey: "gem-key"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	resp, err := p.Chat(context.Background(), ChatRequest{
		Messages: []ChatMessage{
			{Role: "system", Content: "Kamu ramah."},
			{Role: "user", Content: "Halo"},
		},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Content != "Halo dari Gemini" {
		t.Fatalf("unexpected content: %q", resp.Content)
	}
}

func TestGeminiChatAssistantRoleMappedToModel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body geminiRequest
		json.NewDecoder(r.Body).Decode(&body)
		if len(body.Contents) != 2 || body.Contents[1].Role != "model" {
			t.Fatalf("role assistant harusnya di-map ke 'model': %+v", body.Contents)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{{"content": map[string]any{"parts": []map[string]string{{"text": "ok"}}}}},
		})
	}))
	defer srv.Close()

	p, _ := New(Config{Kind: KindGemini, BaseURL: srv.URL, APIKey: "k"})
	_, err := p.Chat(context.Background(), ChatRequest{
		Messages: []ChatMessage{
			{Role: "user", Content: "Halo"},
			{Role: "assistant", Content: "Hai juga"},
		},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
}

func TestGeminiChatProviderError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"message": "API key invalid"}})
	}))
	defer srv.Close()

	p, _ := New(Config{Kind: KindGemini, BaseURL: srv.URL, APIKey: "bad"})
	_, err := p.Chat(context.Background(), ChatRequest{Messages: []ChatMessage{{Role: "user", Content: "hi"}}})
	if err == nil || !strings.Contains(err.Error(), "API key invalid") {
		t.Fatalf("expected provider error to surface, got %v", err)
	}
}

func TestGeminiListModelsFiltersToGenerateContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("key") != "gem-key" {
			t.Fatalf("api key tidak dikirim di query: %s", r.URL.RawQuery)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"models": []map[string]any{
				{"name": "models/gemini-1.5-flash", "supportedGenerationMethods": []string{"generateContent"}},
				{"name": "models/embedding-001", "supportedGenerationMethods": []string{"embedContent"}},
				{"name": "models/gemini-1.5-pro", "supportedGenerationMethods": []string{"generateContent", "countTokens"}},
			},
		})
	}))
	defer srv.Close()

	p, err := New(Config{Kind: KindGemini, BaseURL: srv.URL, APIKey: "gem-key"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	models, err := p.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(models) != 2 || models[0] != "gemini-1.5-flash" || models[1] != "gemini-1.5-pro" {
		t.Fatalf("expected cuma model generateContent (prefix models/ dibuang), got %+v", models)
	}
}
