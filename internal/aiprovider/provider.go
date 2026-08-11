// Package aiprovider adalah abstraksi tipis di atas API chat completion
// berbagai penyedia AI (ChatGPT/OpenAI, DeepSeek, Ollama, endpoint custom
// yang OpenAI-compatible, dan Gemini yang formatnya beda sendiri).
package aiprovider

import (
	"context"
	"errors"
	"net/http"
	"time"
)

type Kind string

const (
	KindOpenAI   Kind = "openai"
	KindDeepSeek Kind = "deepseek"
	KindOllama   Kind = "ollama"
	KindGemini   Kind = "gemini"
	KindCustom   Kind = "custom"
)

var ErrUnknownKind = errors.New("aiprovider: kind tidak dikenal")

type ChatMessage struct {
	Role    string // system | user | assistant
	Content string
}

type ChatRequest struct {
	Model       string
	Messages    []ChatMessage
	Temperature float64
	MaxTokens   int
}

type ChatResponse struct {
	Content          string
	Model            string
	PromptTokens     int
	CompletionTokens int
}

type Provider interface {
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
	// ListModels ngambil daftar model yang tersedia langsung dari API
	// provider, biar UI bisa nyodorin pilihan alih-alih user ngetik manual
	// (dan gampang typo nama model).
	ListModels(ctx context.Context) ([]string, error)
}

type Config struct {
	Kind    Kind
	BaseURL string
	APIKey  string
}

// httpTimeout dikasih napas cukup panjang: model lokal (Ollama) atau model
// reasoning provider cloud bisa butuh puluhan detik buat satu balasan.
const httpTimeout = 60 * time.Second

// New balikin client Provider yang cocok buat Config.Kind. BaseURL kosong
// dibolehin buat openai/deepseek/gemini (dipakein default resmi masing2
// provider) tapi WAJIB diisi buat ollama/custom karena gak ada default yang
// masuk akal (endpoint self-hosted).
func New(cfg Config) (Provider, error) {
	client := &http.Client{Timeout: httpTimeout}
	switch cfg.Kind {
	case KindOpenAI:
		return &openAICompat{baseURL: firstNonEmpty(cfg.BaseURL, "https://api.openai.com/v1"), apiKey: cfg.APIKey, client: client}, nil
	case KindDeepSeek:
		return &openAICompat{baseURL: firstNonEmpty(cfg.BaseURL, "https://api.deepseek.com/v1"), apiKey: cfg.APIKey, client: client}, nil
	case KindOllama:
		if cfg.BaseURL == "" {
			return nil, errors.New("aiprovider: base_url wajib diisi buat provider ollama")
		}
		return &openAICompat{baseURL: cfg.BaseURL, apiKey: cfg.APIKey, client: client}, nil
	case KindCustom:
		if cfg.BaseURL == "" {
			return nil, errors.New("aiprovider: base_url wajib diisi buat provider custom")
		}
		return &openAICompat{baseURL: cfg.BaseURL, apiKey: cfg.APIKey, client: client}, nil
	case KindGemini:
		return &geminiClient{baseURL: firstNonEmpty(cfg.BaseURL, "https://generativelanguage.googleapis.com/v1beta"), apiKey: cfg.APIKey, client: client}, nil
	default:
		return nil, ErrUnknownKind
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
