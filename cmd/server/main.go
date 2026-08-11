package main

import (
	"context"
	"io/fs"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	waLog "go.mau.fi/whatsmeow/util/log"

	"test-agentic"
	"test-agentic/internal/auth"
	"test-agentic/internal/bot"
	"test-agentic/internal/config"
	"test-agentic/internal/conversation"
	"test-agentic/internal/cryptutil"
	"test-agentic/internal/db"
	"test-agentic/internal/gateway/telegram"
	"test-agentic/internal/gateway/whatsapp"
	"test-agentic/internal/httpapi"
	"test-agentic/internal/store"
)

func main() {
	cfg := config.Load()

	box, err := cryptutil.New(cfg.EncryptionKey)
	if err != nil {
		log.Fatalf("encryption key: %v (set ENCRYPTION_KEY — generate dengan `openssl rand -base64 32`)", err)
	}

	sqlDB, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer sqlDB.Close()
	log.Println("✅ SQLite siap:", cfg.DBPath)

	st := store.New(sqlDB, box)
	if err := seedAdmin(st, cfg); err != nil {
		log.Fatalf("seed admin: %v", err)
	}

	authSvc := auth.New(cfg.JWTSecret)
	orch := bot.New(st)
	hub := conversation.New(st, orch)

	waLogger := waLog.Stdout("wa", "WARN", true)
	waMgr, err := whatsapp.New(sqlDB, st, hub, waLogger, cfg.WADeviceName)
	if err != nil {
		log.Fatalf("whatsapp manager: %v", err)
	}
	if err := waMgr.LoadAndConnect(context.Background()); err != nil {
		log.Printf("warn: reconnect sesi whatsapp: %v", err)
	}
	log.Println("✅ WhatsApp gateway siap")

	tgMgr := telegram.New(st, hub, cfg.PublicBaseURL, cfg.TelegramWebhookSecret)

	api := httpapi.New(st, authSvc, orch, waMgr, tgMgr)

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	api.Register(app)
	registerStaticFrontend(app)

	log.Printf("🚀 test-agentic listening on :%s", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

// seedAdmin bikin user admin pertama kali server nyala di DB kosong, dari
// ADMIN_USERNAME/ADMIN_PASSWORD. Sesudah user pertama ada, env ini diabaikan
// selamanya — ganti password lewat re-seed manual di DB kalau perlu.
func seedAdmin(st *store.Store, cfg config.Config) error {
	ctx := context.Background()
	n, err := st.CountUsers(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	if cfg.AdminPassword == "" {
		log.Println("⚠️  belum ada user & ADMIN_PASSWORD kosong — set ADMIN_USERNAME/ADMIN_PASSWORD lalu restart buat bikin akun admin pertama")
		return nil
	}
	hash, err := auth.HashPassword(cfg.AdminPassword)
	if err != nil {
		return err
	}
	if _, err := st.CreateUser(ctx, cfg.AdminUsername, hash); err != nil {
		return err
	}
	log.Printf("✅ admin %q dibuat dari ADMIN_USERNAME/ADMIN_PASSWORD", cfg.AdminUsername)
	return nil
}

// registerStaticFrontend nyajiin hasil build Vue (web/dist) yang di-embed di
// binary. NotFoundFile: "index.html" bikin semua route non-API (mis. /providers,
// /bots) tetap dapet index.html — routing halaman-nya sendiri ditangani
// vue-router di sisi klien.
func registerStaticFrontend(app *fiber.App) {
	sub, err := fs.Sub(assets.DistFS, "web/dist")
	if err != nil {
		log.Fatalf("static assets: %v", err)
	}
	app.Use(filesystem.New(filesystem.Config{
		Root:         http.FS(sub),
		Index:        "index.html",
		NotFoundFile: "index.html",
	}))
}
