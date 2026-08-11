package aiprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// geminiClient nutup Google Gemini API, yang formatnya beda dari OpenAI:
// role "assistant" jadi "model", dan system prompt lewat field terpisah
// systemInstruction, bukan message berrole "system" di array yang sama.
type geminiClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiRequest struct {
	SystemInstruction *geminiContent  `json:"systemInstruction,omitempty"`
	Contents          []geminiContent `json:"contents"`
	GenerationConfig  struct {
		Temperature     float64 `json:"temperature,omitempty"`
		MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	} `json:"generationConfig"`
}

type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (g *geminiClient) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	body := geminiRequest{}
	body.GenerationConfig.Temperature = req.Temperature
	body.GenerationConfig.MaxOutputTokens = req.MaxTokens

	var systemParts []string
	for _, m := range req.Messages {
		switch m.Role {
		case "system":
			systemParts = append(systemParts, m.Content)
		case "assistant":
			body.Contents = append(body.Contents, geminiContent{Role: "model", Parts: []geminiPart{{Text: m.Content}}})
		default: // "user" dan role lain dianggap user
			body.Contents = append(body.Contents, geminiContent{Role: "user", Parts: []geminiPart{{Text: m.Content}}})
		}
	}
	if len(systemParts) > 0 {
		body.SystemInstruction = &geminiContent{Parts: []geminiPart{{Text: strings.Join(systemParts, "\n\n")}}}
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return ChatResponse{}, err
	}

	model := req.Model
	if model == "" {
		model = "gemini-1.5-flash"
	}
	endpoint := fmt.Sprintf("%s/models/%s:generateContent?key=%s",
		strings.TrimRight(g.baseURL, "/"), model, url.QueryEscape(g.apiKey))

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return ChatResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("aiprovider: request gagal: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return ChatResponse{}, err
	}

	var out geminiResponse
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
	if len(out.Candidates) == 0 || len(out.Candidates[0].Content.Parts) == 0 {
		return ChatResponse{}, fmt.Errorf("aiprovider: response tanpa candidates")
	}

	var text strings.Builder
	for _, p := range out.Candidates[0].Content.Parts {
		text.WriteString(p.Text)
	}

	return ChatResponse{
		Content:          text.String(),
		Model:            model,
		PromptTokens:     out.UsageMetadata.PromptTokenCount,
		CompletionTokens: out.UsageMetadata.CandidatesTokenCount,
	}, nil
}
