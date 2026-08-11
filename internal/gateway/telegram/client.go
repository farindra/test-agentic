// Package telegram membungkus Telegram Bot API secukupnya buat kebutuhan
// gateway ini: daftarin webhook, kirim pesan, dan proses update yang masuk.
// Sengaja gak pakai library pihak ketiga — permukaan API yang dipakai kecil,
// jadi HTTP client tipis lebih gampang dikontrol daripada nambah dependency.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// apiBaseURL bisa ditimpa di unit test biar apiClient nembak ke httptest
// server lokal, bukan API Telegram beneran.
var apiBaseURL = "https://api.telegram.org"

type apiClient struct {
	token  string
	client *http.Client
}

func newAPIClient(token string) *apiClient {
	return &apiClient{token: token, client: &http.Client{Timeout: 20 * time.Second}}
}

func (a *apiClient) endpoint(method string) string {
	return fmt.Sprintf("%s/bot%s/%s", apiBaseURL, a.token, method)
}

type apiResult struct {
	OK          bool            `json:"ok"`
	Description string          `json:"description"`
	Result      json.RawMessage `json:"result"`
}

func (a *apiClient) call(ctx context.Context, method string, payload any) (json.RawMessage, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint(method), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("telegram: request %s gagal: %w", method, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var out apiResult
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("telegram: response %s bukan JSON valid: %s", method, raw)
	}
	if !out.OK {
		return nil, fmt.Errorf("telegram: %s ditolak: %s", method, out.Description)
	}
	return out.Result, nil
}

type meResult struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

func (a *apiClient) getMe(ctx context.Context) (meResult, error) {
	raw, err := a.call(ctx, "getMe", struct{}{})
	if err != nil {
		return meResult{}, err
	}
	var me meResult
	if err := json.Unmarshal(raw, &me); err != nil {
		return meResult{}, err
	}
	return me, nil
}

func (a *apiClient) setWebhook(ctx context.Context, webhookURL, secretToken string) error {
	_, err := a.call(ctx, "setWebhook", map[string]string{
		"url":          webhookURL,
		"secret_token": secretToken,
	})
	return err
}

func (a *apiClient) deleteWebhook(ctx context.Context) error {
	_, err := a.call(ctx, "deleteWebhook", struct{}{})
	return err
}

func (a *apiClient) sendMessage(ctx context.Context, chatID, text string) error {
	_, err := a.call(ctx, "sendMessage", map[string]string{
		"chat_id": chatID,
		"text":    text,
	})
	return err
}

// webhookURLFor susun URL publik webhook buat satu sesi bot. sessionID
// dipakai apa adanya di path — dijamin aman karena selalu UUID hasil
// generate kita sendiri, bukan input user.
func webhookURLFor(publicBaseURL, sessionID string) string {
	return publicBaseURL + "/api/gateway/telegram/webhook/" + url.PathEscape(sessionID)
}
