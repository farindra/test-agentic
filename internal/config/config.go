// Package config membaca konfigurasi service dari environment variable.
package config

import (
	"os"
	"strings"
)

type Config struct {
	Port                  string
	DBPath                string
	PublicBaseURL         string // dipakai buat susun URL webhook Telegram, mis. https://test-agentic.farindra.com
	JWTSecret             string
	EncryptionKey         string // base64, 32 byte setelah decode — buat enkripsi API key/token di DB
	AdminUsername         string
	AdminPassword         string // dipakai sekali buat seed admin pertama kali; sesudahnya diabaikan
	WADeviceName          string
	TelegramWebhookSecret string
}

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func Load() Config {
	return Config{
		Port:                  env("PORT", "8080"),
		DBPath:                env("DB_PATH", "./data/test-agentic.db"),
		PublicBaseURL:         strings.TrimRight(env("PUBLIC_BASE_URL", "http://localhost:8080"), "/"),
		JWTSecret:             env("JWT_SECRET", "change-me-jwt-secret"),
		EncryptionKey:         env("ENCRYPTION_KEY", ""),
		AdminUsername:         env("ADMIN_USERNAME", "admin"),
		AdminPassword:         env("ADMIN_PASSWORD", ""),
		WADeviceName:          env("WA_DEVICE_NAME", "Test Agentic Bot"),
		TelegramWebhookSecret: env("TELEGRAM_WEBHOOK_SECRET", "change-me-tg-secret"),
	}
}
