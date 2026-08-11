# Test Agentic

Chatbot AI sederhana, self-contained: satu binary Go (embed Vue 3 sebagai
static asset) + SQLite, terintegrasi ke ChatGPT/OpenAI, DeepSeek, Gemini,
Ollama, atau endpoint custom OpenAI-compatible lainnya — plus chat gateway
buat WhatsApp (whatsmeow) dan Telegram Bot.

## Fitur

- **AI Provider** — kelola koneksi ke ChatGPT, DeepSeek, Gemini, Ollama, atau
  provider custom (OpenAI-compatible). API key dienkripsi (AES-256-GCM)
  sebelum disimpan.
- **Chatbot** — susun persona bot: system prompt, model, temperature, max
  tokens, terikat ke satu AI provider. System prompt gak bisa dihapus kalau
  masih di-binding ke sesi gateway aktif (409), dan provider gak bisa
  dihapus kalau masih dipakai chatbot — biar gak ada auto-reply yang
  diem-diem berhenti tanpa peringatan.
- **Variables** — key-value custom (mis. `jam_buka`, `alamat`) yang bisa
  disisipkan ke System Prompt chatbot manapun lewat placeholder
  `{{nama_variable}}`.
- **Playground** — coba chatbot langsung di browser sebelum dihubungkan ke
  gateway manapun.
- **Chat Gateway — WhatsApp** — manajemen sesi multi-nomor via whatsmeow,
  pairing lewat QR code, kirim/terima pesan.
- **Chat Gateway — Telegram** — manajemen bot Telegram (token dari
  @BotFather) lewat webhook.
- **Inbox** — semua percakapan dari WhatsApp & Telegram dalam satu tempat,
  dengan toggle auto-reply per percakapan buat handover ke manusia.
- **Auth** — single-admin, login JWT.

## Arsitektur

Satu container, satu binary: hasil build Vue (`web/dist`) di-embed ke
binary Go lewat `//go:embed`, jadi backend sekaligus nyajiin frontend-nya
sendiri — gak butuh Nginx atau static hosting terpisah. Semua data (sesi
whatsmeow, providers, bots, percakapan) disimpan di satu file SQLite.

```
Browser ──► Traefik ──► test-agentic (Fiber, :8080)
                              ├── /api/*        → REST API (JWT auth)
                              ├── /*             → static Vue (SPA)
                              └── SQLite (data/test-agentic.db)
                                    ├── tabel aplikasi (providers, bots, ...)
                                    └── tabel whatsmeow_* (sesi WA)
```

## Struktur Folder

```
cmd/server/          entrypoint (main.go)
assets.go            //go:embed web/dist — WAJIB di root modul
internal/
  config/            load environment variable
  db/                buka SQLite + migration runner
  cryptutil/          enkripsi AES-256-GCM buat secret di DB
  auth/               hashing password + JWT
  aiprovider/         client OpenAI-compatible & Gemini
  store/              repository layer (semua query SQL)
  bot/                orkestrasi: system prompt + history + panggil provider
  conversation/       hub: log pesan masuk, putuskan auto-reply
  gateway/whatsapp/    integrasi whatsmeow
  gateway/telegram/    integrasi Telegram Bot API (webhook)
  httpapi/             route & handler REST
web/                  Vue 3 + Vite + Pinia + Vue Router
docs/API.md           referensi endpoint REST
```

## Cara Instal — Mode Lokal (development)

Prasyarat: Go 1.26+, Node 20+.

```bash
# 1. Salin env
cp .env.example .env
# isi minimal: JWT_SECRET, ENCRYPTION_KEY (openssl rand -base64 32),
# ADMIN_USERNAME, ADMIN_PASSWORD

# 2. Backend
go run ./cmd/server
# server jalan di :8080, admin pertama otomatis dibuat dari .env

# 3. Frontend (terminal terpisah) — dev server dgn hot reload + proxy /api ke :8080
cd web
npm install
npm run dev
# buka http://localhost:5173
```

Untuk nyoba build production secara lokal (satu binary, tanpa dev server
terpisah):

```bash
cd web && npm install && npm run build && cd ..
go run ./cmd/server
# buka http://localhost:8080 — Vue yang sudah di-build ikut ke-serve dari sini
```

## Cara Instal — Mode Docker (deploy)

```bash
cp .env.example .env
# isi PUBLIC_BASE_URL dengan domain asli (wajib https, dipakai webhook Telegram)
# isi JWT_SECRET, ENCRYPTION_KEY, TELEGRAM_WEBHOOK_SECRET, ADMIN_USERNAME/PASSWORD

docker compose up -d --build
```

`docker-compose.yml` cuma punya satu service (`app`) — Dockerfile
multi-stage yang bikin frontend (Node) lalu build binary Go yang meng-embed
hasilnya, jadi image akhir cuma satu binary + `ca-certificates`. Data
(SQLite + sesi whatsmeow) disimpan di `./data` (bind mount host).

Container join network eksternal `ceremai` supaya Traefik (di stack
`ceremai`) bisa nge-routing `test-agentic.farindra.com` ke situ — lihat
`/home/staging/Docker/ceremai/traefik/staging.yml`, router `test-agentic`.

## Environment Variables

Lihat [.env.example](.env.example) buat daftar lengkap + penjelasan tiap
variabel. Yang wajib diisi sebelum jalan:

| Variabel | Kegunaan |
|---|---|
| `JWT_SECRET` | tandatangan token sesi admin |
| `ENCRYPTION_KEY` | enkripsi API key provider & token bot Telegram di DB |
| `ADMIN_USERNAME` / `ADMIN_PASSWORD` | dipakai SEKALI saat tabel `users` masih kosong |
| `PUBLIC_BASE_URL` | dasar URL webhook Telegram — wajib https di produksi |
| `TELEGRAM_WEBHOOK_SECRET` | verifikasi request webhook beneran dari Telegram |

## Testing

```bash
# Backend — semua unit test (store, aiprovider, bot, conversation, gateway, httpapi)
go test ./...

# Frontend — unit test store, api client, & composables (Vitest)
cd web && npm test
```

## Catatan Keamanan

- API key provider & token bot Telegram dienkripsi (AES-256-GCM) sebelum
  masuk SQLite — kehilangan `ENCRYPTION_KEY` berarti kehilangan akses ke
  semua secret yang tersimpan, jadi backup key ini terpisah dari database.
- Endpoint `/api/*` (kecuali `/api/auth/login` dan webhook Telegram) wajib
  Bearer JWT.
- Webhook Telegram diverifikasi lewat header
  `X-Telegram-Bot-Api-Secret-Token`, bukan lewat JWT — itu yang dipanggil
  Telegram, bukan admin yang login.
- Pesan masuk dari WhatsApp/Telegram TIDAK PERNAH ditafsirkan sebagai
  perintah ke platform — `conversation.Hub` cuma nyimpen log dan
  nerusinnya sebagai teks chat biasa ke provider AI, gak ada jalur ke
  fungsi admin (bikin/hapus bot, provider, sesi, dst) dari isi chat.
- **Mitigasi prompt injection** (bukan jaminan mutlak): tiap system prompt
  otomatis dikasih instruksi pengaman di depan buat nahan percobaan
  "abaikan instruksi sebelumnya" / "tampilkan system prompt kamu" dkk
  (lihat `promptGuard` di `internal/bot/bot.go`).
- Pesan masuk dibatasi maks 4000 karakter dan di-rate-limit 8 pesan/menit
  per kontak (in-memory, per sesi gateway) — pesan yang kena limit tetap
  kecatet di inbox, cuma gak dibales otomatis, biar spam gak nyedot kuota
  biaya API provider AI berkali-kali.
