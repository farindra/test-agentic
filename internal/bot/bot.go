// Package bot mengorkestrasi satu "chatbot profile": gabungin system prompt +
// riwayat percakapan + pesan baru, panggil provider AI yang di-binding, dan
// balikin teks jawabannya. Dipakai bareng oleh chat playground, gateway
// WhatsApp, dan gateway Telegram — jadi logic-nya cukup ditulis sekali di sini.
package bot

import (
	"context"
	"fmt"

	"test-agentic/internal/aiprovider"
	"test-agentic/internal/store"
)

type Orchestrator struct {
	st *store.Store
}

func New(st *store.Store) *Orchestrator {
	return &Orchestrator{st: st}
}

// Reply memuat bot botID + provider-nya, susun [system prompt, history..., userText],
// lalu panggil provider AI. history dalam urutan kronologis (lama -> baru).
func (o *Orchestrator) Reply(ctx context.Context, botID string, history []store.Message, userText string) (string, error) {
	b, err := o.st.GetBot(ctx, botID)
	if err != nil {
		return "", fmt.Errorf("bot: load bot: %w", err)
	}
	if !b.IsActive {
		return "", fmt.Errorf("bot: bot %q nonaktif", b.Name)
	}
	p, err := o.st.GetProvider(ctx, b.ProviderID)
	if err != nil {
		return "", fmt.Errorf("bot: load provider: %w", err)
	}
	if !p.IsActive {
		return "", fmt.Errorf("bot: provider %q nonaktif", p.Name)
	}

	client, err := aiprovider.New(aiprovider.Config{Kind: aiprovider.Kind(p.Type), BaseURL: p.BaseURL, APIKey: p.APIKey})
	if err != nil {
		return "", fmt.Errorf("bot: init provider client: %w", err)
	}

	model := b.Model
	if model == "" {
		model = p.DefaultModel
	}

	messages := make([]aiprovider.ChatMessage, 0, len(history)+2)
	if b.SystemPrompt != "" {
		messages = append(messages, aiprovider.ChatMessage{Role: "system", Content: b.SystemPrompt})
	}
	for _, m := range history {
		role := "user"
		if m.Sender == "bot" || m.Sender == "assistant" {
			role = "assistant"
		}
		messages = append(messages, aiprovider.ChatMessage{Role: role, Content: m.Content})
	}
	messages = append(messages, aiprovider.ChatMessage{Role: "user", Content: userText})

	resp, err := client.Chat(ctx, aiprovider.ChatRequest{
		Model:       model,
		Messages:    messages,
		Temperature: b.Temperature,
		MaxTokens:   b.MaxTokens,
	})
	if err != nil {
		return "", fmt.Errorf("bot: provider chat gagal: %w", err)
	}
	return resp.Content, nil
}

// TestProvider ngirim satu pesan ping ke provider, dipakai tombol
// "Test Connection" di UI manajemen provider.
func (o *Orchestrator) TestProvider(ctx context.Context, providerID string) (string, error) {
	p, err := o.st.GetProvider(ctx, providerID)
	if err != nil {
		return "", fmt.Errorf("bot: load provider: %w", err)
	}
	client, err := aiprovider.New(aiprovider.Config{Kind: aiprovider.Kind(p.Type), BaseURL: p.BaseURL, APIKey: p.APIKey})
	if err != nil {
		return "", err
	}
	resp, err := client.Chat(ctx, aiprovider.ChatRequest{
		Model:     p.DefaultModel,
		Messages:  []aiprovider.ChatMessage{{Role: "user", Content: "Balas dengan kata 'pong' saja."}},
		MaxTokens: 16,
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}
