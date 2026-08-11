package aiprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// openAICompat nutup semua provider yang ngikutin bentuk
// POST /chat/completions ala OpenAI: ChatGPT, DeepSeek, Ollama (mode
// OpenAI-compat-nya), dan endpoint custom mana pun yang ngikutin format ini.
type openAICompat struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

type oaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type oaRequest struct {
	Model       string      `json:"model"`
	Messages    []oaMessage `json:"messages"`
	Temperature float64     `json:"temperature,omitempty"`
	MaxTokens   int         `json:"max_tokens,omitempty"`
}

type oaResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message oaMessage `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (o *openAICompat) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	body := oaRequest{
		Model:       req.Model,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}
	for _, m := range req.Messages {
		body.Messages = append(body.Messages, oaMessage{Role: m.Role, Content: m.Content})
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return ChatResponse{}, err
	}

	url := strings.TrimRight(o.baseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return ChatResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if o.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)
	}

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("aiprovider: request gagal: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return ChatResponse{}, err
	}

	var out oaResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return ChatResponse{}, fmt.Errorf("aiprovider: response bukan JSON valid (status %d): %s", resp.StatusCode, truncate(raw, 300))
	}
	if resp.StatusCode >= 300 {
		msg := fmt.Sprintf("status %d", resp.StatusCode)
		if out.Error != nil && out.Error.Message != "" {
			msg = out.Error.Message
		}
		return ChatResponse{}, fmt.Errorf("aiprovider: provider error: %s", msg)
	}
	if len(out.Choices) == 0 {
		return ChatResponse{}, fmt.Errorf("aiprovider: response tanpa choices")
	}

	return ChatResponse{
		Content:          out.Choices[0].Message.Content,
		Model:            out.Model,
		PromptTokens:     out.Usage.PromptTokens,
		CompletionTokens: out.Usage.CompletionTokens,
	}, nil
}

func truncate(b []byte, n int) string {
	s := string(b)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
