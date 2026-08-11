# Test Agentic — Chatbot AI, WhatsApp & Telegram Gateway

Aplikasi admin untuk mengelola **chatbot AI** (ChatGPT, DeepSeek, Gemini, Ollama, atau endpoint custom OpenAI-compatible) dan menghubungkannya ke **WhatsApp** dan **Telegram** sebagai auto-reply. Dibangun dengan Go (Fiber), Vue 3, dan SQLite — satu binary, satu container.

**Demo:** [test-agentic.farindra.com](https://test-agentic.farindra.com) · login `admin` / `admin!23` (kredensialnya juga tertulis di halaman login)

---

## Daftar Isi

1. [Konsep Utama](#konsep-utama)
2. [Fitur](#fitur)
3. [Alur Kerja](#alur-kerja)
4. [Teknologi](#teknologi)
5. [Cara Install](#cara-install)
6. [Konfigurasi (.env)](#konfigurasi-env)
7. [Tutorial Penggunaan](#tutorial-penggunaan)
8. [Keamanan](#keamanan)
9. [Menjalankan Test](#menjalankan-test)
10. [Struktur Database](#struktur-database)
11. [Struktur Proyek](#struktur-proyek)
12. [Troubleshooting](#troubleshooting)

---

## Konsep Utama

Tiga aturan yang menjadi dasar seluruh aplikasi:

**1. Provider itu koneksi, Bot itu persona.**
Satu **AI Provider** (endpoint + API key) bisa dipakai banyak **Bot** sekaligus. Tiap Bot punya system prompt, model, dan parameter sendiri — jadi satu langganan OpenAI, misalnya, bisa melayani beberapa persona bot yang berbeda tanpa perlu daftar ulang.

**2. Semua pesan masuk dicatat dulu, baru (mungkin) dibalas.**
Setiap pesan dari WhatsApp/Telegram selalu masuk ke **Inbox** lebih dulu, apa pun kondisinya. Baru diteruskan ke bot untuk dibalas otomatis kalau auto-reply aktif di level sesi *dan* level percakapan. Mematikan auto-reply kapan saja untuk *handover* ke manusia tidak menghapus histori apa pun — percakapannya tetap utuh di Inbox.

**3. Data yang saling terikat tidak bisa dihapus sembarangan.**
Bot yang masih di-*binding* ke sesi WhatsApp/Telegram, atau Provider yang masih dipakai Bot, ditolak untuk dihapus (`409 Conflict`) — supaya tidak ada auto-reply yang diam-diam berhenti tanpa peringatan.

---

## Fitur

### AI Provider
- Koneksi ke **ChatGPT/OpenAI**, **DeepSeek**, **Gemini**, **Ollama** (self-hosted), atau **endpoint custom** OpenAI-compatible
- API key **dienkripsi AES-256-GCM** sebelum disimpan — tidak pernah dibalas utuh ke frontend, cuma penanda "sudah diisi" + 4 karakter terakhir
- Tombol **Test Connection** — kirim satu pesan ping buat validasi kredensial

### Chatbot
- System prompt, model, `temperature`, `max_tokens`, terikat ke satu AI Provider
- Tombol **"Ambil daftar model"** — ambil daftar model asli langsung dari API provider (`GET /models`), bukan ketik manual dan rawan salah ketik
- Tiap system prompt otomatis dilapisi *guard* anti prompt-injection (lihat [Keamanan](#keamanan))

### Variables
- Key-value custom (mis. `jam_buka`, `alamat`) yang bisa disisipkan ke system prompt chatbot mana pun lewat placeholder `{{nama_variable}}`
- Placeholder yang key-nya tidak ditemukan dibiarkan apa adanya di prompt — biar typo kelihatan, bukan diam-diam hilang

### Playground
- Coba chatbot langsung di browser, lengkap dengan histori per sesi percakapan, sebelum dihubungkan ke gateway mana pun

### Chat Gateway — WhatsApp
- Manajemen sesi multi-nomor via `whatsmeow` (protokol WhatsApp Web multi-device)
- Pairing lewat **QR code** yang otomatis diganti saat kedaluwarsa, dengan polling status tiap 3 detik supaya UI auto-refresh begitu HP berhasil connect
- Kirim pesan manual dan cek nomor terdaftar

### Chat Gateway — Telegram
- Manajemen bot Telegram (token dari **@BotFather**) lewat webhook, diverifikasi lewat secret token — bukan polling

### Inbox
- Semua percakapan dari WhatsApp & Telegram dalam satu tempat
- Toggle auto-reply **per percakapan** (di atas toggle auto-reply per sesi gateway) buat *handover* manual ke manusia

### Lain-lain
- Login admin dengan JWT dan password *ter-hash* (bcrypt)
- Tampilan **responsif**: sidebar jadi *drawer* di ponsel, tabel bisa di-*scroll* horizontal, panel chat jadi *single-pane* dengan tombol kembali
- Indikator *loading* di semua halaman yang mengambil data, plus banner otomatis saat browser terdeteksi offline

---

## Alur Kerja

Yang terjadi setiap ada pesan masuk dari WhatsApp/Telegram:

```
                Pesan masuk (WA/Telegram)
                          │
                          ▼
                 ┌────────────────┐
                 │  Dicatat ke    │   ← SELALU, apa pun kondisi di bawah ini
                 │  Inbox         │
                 └───────┬────────┘
                          │
              auto_reply sesi aktif? ──Tidak──► berhenti (nunggu dibalas manual)
                          │ Ya
              auto_reply percakapan aktif? ──Tidak──► berhenti (lagi di-handover ke manusia)
                          │ Ya
              ada bot ter-binding? ──Tidak──► berhenti (belum dipasang bot)
                          │ Ya
                 ┌────────────────┐
                 │ Panjang pesan  │──Kena limit──► berhenti (tercatat, TIDAK dibalas —
                 │ & rate limit   │                cegah biaya API membengkak dari spam)
                 │ per kontak OK? │
                 └───────┬────────┘
                          │ Ya
                 ┌─────────────────────────┐
                 │  bot.Orchestrator.Reply  │
                 │   promptGuard            │
                 │   + system prompt        │
                 │   + {{variable}}         │
                 │   + riwayat percakapan   │
                 │   + pesan baru           │
                 └────────────┬─────────────┘
                               ▼
                        Provider AI (Chat)
                               │
                     gagal? ──Ya──► dicatat di log server, TIDAK dibalas
                               │      (pesan tetap ada di Inbox buat dibalas manual)
                               │ Tidak
                               ▼
                Kirim balasan lewat gateway asal
                               │
                               ▼
                   Dicatat sebagai pesan keluar
```

---

## Teknologi

| Komponen | Pilihan | Alasan |
|---|---|---|
| Bahasa backend | Go 1.26 | Kompilasi jadi satu binary statis, gampang di-*deploy*, konkurensi murah buat ngurus banyak sesi gateway sekaligus |
| Web framework | Fiber v2 | Cepat, API-nya mirip Express, gampang dipasangin middleware |
| Database | SQLite (`modernc.org/sqlite`) | **Pure-Go** — tidak butuh CGO, jadi image Docker tetap ringan tanpa perlu toolchain compiler tambahan |
| Frontend | Vue 3 + Vite + Pinia | SPA ringan, di-*build* jadi static asset lalu di-*embed* langsung ke dalam binary Go lewat `//go:embed` |
| WhatsApp | `whatsmeow` | Library protokol WhatsApp Web multi-device |
| Auth | JWT (`golang-jwt/jwt`) + bcrypt | *Stateless*, tidak butuh session store terpisah |
| Enkripsi secret | AES-256-GCM | API key provider & token bot Telegram tidak pernah tersimpan plaintext |

**Kenapa satu binary?** Hasil `npm run build` Vue di-*embed* ke dalam binary Go lewat `//go:embed`, jadi backend sekaligus menyajikan frontend-nya sendiri — tidak butuh Nginx atau static hosting terpisah. Deploy-nya cukup satu image Docker, satu container, satu file SQLite.

---

## Cara Install

### Cara 1 — Docker Compose (disarankan)

Tidak perlu memasang Go atau Node.js di komputer. Cukup Docker.

```bash
# 1. Ambil kodenya
git clone git@github.com:farindra/test-agentic.git
cd test-agentic

# 2. Siapkan berkas konfigurasi
cp .env.example .env

# 3. Ganti secret-secret di .env dengan nilai acak
sed -i "s|^JWT_SECRET=.*|JWT_SECRET=$(openssl rand -base64 32)|" .env
sed -i "s|^ENCRYPTION_KEY=.*|ENCRYPTION_KEY=$(openssl rand -base64 32)|" .env
sed -i "s|^TELEGRAM_WEBHOOK_SECRET=.*|TELEGRAM_WEBHOOK_SECRET=$(openssl rand -hex 24)|" .env

# 4. Isi PUBLIC_BASE_URL dengan domain asli (wajib https — dipakai buat
#    webhook Telegram) dan ganti ADMIN_PASSWORD

# 5. Jalankan
docker compose up -d --build

# 6. Lihat log untuk memastikan admin pertama berhasil dibuat
docker compose logs -f app
```

`docker-compose.yml` cuma punya satu service (`app`) — Dockerfile *multi-stage* yang membangun frontend (Node) lalu membangun binary Go yang meng-*embed* hasilnya, jadi image akhir cuma satu binary + `ca-certificates`. Data (SQLite + sesi whatsmeow) disimpan di `./data` (*bind mount* host).

Perintah lain yang berguna:

```bash
docker compose logs -f app       # pantau log
docker compose restart           # restart aplikasi
docker compose down              # hentikan
```

### Cara 2 — Langsung dengan Go + Node.js

Butuh **Go 1.26+** dan **Node 20+**.

```bash
git clone git@github.com:farindra/test-agentic.git
cd test-agentic

cp .env.example .env
# isi minimal: JWT_SECRET, ENCRYPTION_KEY (openssl rand -base64 32),
# ADMIN_USERNAME, ADMIN_PASSWORD

# Backend
go run ./cmd/server
# server jalan di :8080, admin pertama otomatis dibuat dari .env

# Frontend (terminal terpisah) — dev server dgn hot reload + proxy /api ke :8080
cd web
npm install
npm run dev
# buka http://localhost:5173
```

Untuk mencoba *build* production secara lokal (satu binary, tanpa dev server terpisah):

```bash
cd web && npm install && npm run build && cd ..
go run ./cmd/server
# buka http://localhost:8080 — Vue yang sudah di-build ikut ke-serve dari sini
```

### Menjalankan di belakang domain (opsional)

Contoh untuk Traefik dengan *file provider*:

```yaml
http:
  routers:
    test-agentic:
      rule: "Host(`test-agentic.example.com`)"
      service: test-agentic
      entryPoints: [web, websecure]
  services:
    test-agentic:
      loadBalancer:
        servers:
          - url: "http://test-agentic-app-1:8080"
```

---

## Konfigurasi (.env)

| Variabel | Default | Keterangan |
|---|---|---|
| `PORT` | `8080` | Port HTTP server (di-*override* oleh `docker-compose.yml` saat deploy) |
| `DB_PATH` | `./data/test-agentic.db` | Lokasi berkas SQLite |
| `PUBLIC_BASE_URL` | `http://localhost:8080` | Dasar URL webhook Telegram — **wajib https** di produksi |
| `JWT_SECRET` | — | **Wajib diganti.** Tandatangan token sesi admin |
| `ENCRYPTION_KEY` | — | **Wajib diganti.** Base64, 32 byte — enkripsi API key provider & token bot Telegram di DB |
| `TELEGRAM_WEBHOOK_SECRET` | — | **Wajib diganti.** Verifikasi request webhook beneran dari Telegram |
| `ADMIN_USERNAME` / `ADMIN_PASSWORD` | `admin` / — | **Hanya dipakai sekali** saat tabel `users` masih kosong |
| `WA_DEVICE_NAME` | `Test Agentic Bot` | Nama yang muncul di WhatsApp → Perangkat Tertaut |
| `WA_DEFAULT_COUNTRY_CODE` | `62` | Kode negara buat normalisasi nomor lokal (`08xx` → `62xx`) |

> Mengubah `ADMIN_PASSWORD` setelah user admin pertama terlanjur dibuat **tidak** mengubah password yang tersimpan — env ini cuma dibaca sekali. Ganti lewat update langsung ke tabel `users` kalau perlu.
>
> Halaman login sengaja menampilkan kredensial admin karena ini aplikasi demo. Hapus blok itu di `web/src/views/Login.vue` bila dipakai untuk data sungguhan.

---

## Tutorial Penggunaan

### 1. Masuk

Buka aplikasi, login dengan username & password yang tertera di halaman login (default `admin` / `admin!23`).

### 2. Tambah AI Provider

Menu **AI Provider** → **+ Tambah Provider**. Pilih tipe (ChatGPT, DeepSeek, Gemini, Ollama, atau Custom), isi API key, lalu klik **Test** buat memastikan kredensialnya benar sebelum dipakai bot.

`base_url` wajib diisi untuk tipe `ollama` dan `custom` (tidak ada default yang masuk akal buat endpoint self-hosted); tipe lain boleh dikosongkan untuk pakai endpoint resmi masing-masing provider.

### 3. Bikin Chatbot

Menu **Chatbot** → **+ Tambah Chatbot**. Pilih provider, tulis system prompt, lalu klik **"Ambil daftar model"** buat milih model asli dari dropdown alih-alih ketik manual.

System prompt boleh berisi placeholder `{{nama_variable}}` — isi nilainya di menu **Variables** supaya tidak perlu hardcode info seperti jam buka atau alamat langsung di dalam prompt.

### 4. Coba di Playground

Menu **Playground** → pilih bot → **+ Chat Baru**. Ini tempat aman buat menguji persona bot sebelum dihubungkan ke nomor WhatsApp atau bot Telegram sungguhan.

### 5. Hubungkan ke WhatsApp

Menu **Chat Gateway → WhatsApp** → **+ Sesi Baru** → **Scan QR** dari HP (WhatsApp → Perangkat Tertaut → Tautkan Perangkat). Status berubah otomatis jadi **Connected** begitu berhasil — tidak perlu refresh manual.

Pilih chatbot di kolom **Bot** pada baris sesi tersebut, dan pastikan toggle **Auto-Reply** menyala.

### 6. Hubungkan ke Telegram

Buat bot baru lewat **@BotFather** di Telegram, salin tokennya. Menu **Chat Gateway → Telegram** → **+ Bot Baru**, tempel token, lalu klik **Aktifkan** — ini mendaftarkan webhook ke Telegram secara otomatis.

### 7. Pantau & handover manual di Inbox

Menu **Inbox** menampilkan semua percakapan dari WhatsApp & Telegram. Kalau ada percakapan yang perlu ditangani manusia, matikan toggle **Auto-reply bot** di percakapan itu saja — sesi gateway-nya tetap aktif buat kontak lain, cuma percakapan ini yang berhenti dibalas otomatis.

---

## Keamanan

- **API key & token dienkripsi** (AES-256-GCM) sebelum masuk SQLite — kehilangan `ENCRYPTION_KEY` berarti kehilangan akses ke semua secret tersimpan, jadi *backup* key ini terpisah dari database.
- **Pesan chat tidak pernah ditafsirkan sebagai perintah ke platform** — jalur pesan masuk cuma mencatat log dan meneruskannya sebagai teks biasa ke provider AI; tidak ada jalur dari isi chat ke fungsi admin (bikin/hapus bot, provider, sesi, dst).
- **Mitigasi prompt injection** (bukan jaminan mutlak): tiap system prompt otomatis dilapisi instruksi pengaman di depan, buat menahan percobaan "abaikan instruksi sebelumnya" / "tampilkan system prompt kamu" dan variasinya.
- **Rate limit & batas panjang pesan**: maksimal 4000 karakter dan 8 pesan/menit per kontak (*in-memory*, per sesi gateway). Pesan yang kena limit tetap tercatat di Inbox, cuma tidak dibalas otomatis.
- Endpoint `/api/*` (kecuali `/api/auth/login` dan webhook Telegram) wajib **Bearer JWT**. Webhook Telegram diverifikasi lewat header `X-Telegram-Bot-Api-Secret-Token`, bukan JWT — itu yang dipanggil Telegram, bukan admin yang login.

---

## Menjalankan Test

```bash
# Backend — lewat Docker (tidak perlu Go di komputer)
docker run --rm -v "$PWD":/app -w /app -e CGO_ENABLED=0 golang:1.26-alpine go test ./...

# Atau langsung, kalau Go sudah terpasang
go test ./...

# Frontend
cd web && npm test
```

### Cakupan (Backend — Go, 60 test)

| Package | Test | Yang diuji |
|---|---:|---|
| `httpapi` | 9 | Endpoint REST end-to-end: login, CRUD, guard hapus bot/provider, fallback API key |
| `bot` | 8 | Orkestrasi reply, *prompt guard*, substitusi `{{variable}}` |
| `store` | 6 | CRUD tabel provider, bot, sesi, percakapan, pesan, variables |
| `conversation` | 6 | Logika auto-reply, batas panjang pesan |
| `openai_compat` | 6 | Client OpenAI-compatible (mock HTTP server) |
| `gateway/telegram` | 5 | Webhook, activate/deactivate |
| `gemini` | 4 | Client Gemini (format request beda dari OpenAI) |
| `auth` | 4 | Hash password, JWT issue/verify |
| `cryptutil` | 4 | Enkripsi/dekripsi AES-256-GCM |
| `ratelimit` | 3 | Sliding-window limiter per kontak |
| `gateway/whatsapp` | 3 | Normalisasi nomor telepon |
| `db` | 2 | Migration jalan & idempotent |

### Cakupan (Frontend — Vitest, 13 test)

| Berkas | Test | Yang diuji |
|---|---:|---|
| `api/client.test.js` | 7 | Header auth, timeout *fetch*, pesan error jaringan |
| `stores/auth.test.js` | 3 | Login/logout |
| `composables/useOnline.test.js` | 3 | Deteksi online/offline browser |

---

## Struktur Database

```
  users                    ai_providers
  ─────                    ────────────
  id (PK)                  id (PK)
  username (UNIQUE)         name, type, base_url
  password_hash             api_key_enc
                             default_model, is_active
                                 │ 1:N
                                 ▼
                             bots
                             ────
                             id (PK)
                             provider_id (FK) ── ON DELETE CASCADE*
                             name, model, system_prompt
                             temperature, max_tokens, is_active
                                 │ 1:N
                    ┌────────────┴────────────┐
                    ▼                         ▼
           gateway_sessions             playground_sessions
           ────────────────             ───────────────────
           id (PK)                      bot_id (FK)
           kind: whatsapp | telegram    title
           bot_id (FK) ── ON DELETE SET NULL*     │ 1:N
           wa_jid / telegram_token_enc            ▼
           auto_reply, status              playground_messages
               │ 1:N                       role: user | assistant
               ▼                           content
           conversations
           ─────────────
           id (PK)
           session_id (FK), contact_id, contact_name
           auto_reply, last_message_at
               │ 1:N
               ▼
           messages
           ────────
           conversation_id (FK)
           direction: in | out
           sender: user | bot | admin
           content, provider_meta

  variables
  ─────────
  key (PK), value    ← dipakai lewat {{key}} di bots.system_prompt
```

`*` — secara *schema* `provider_id` di `bots` itu `ON DELETE CASCADE` dan `bot_id` di `gateway_sessions` itu `ON DELETE SET NULL`, tapi keduanya **tidak pernah kepakai** lewat API: `DELETE /providers/:id` dan `DELETE /bots/:id` mengecek dulu ada tidaknya baris yang bergantung, dan menolak dengan `409` kalau masih ada — lihat [Konsep Utama](#konsep-utama) #3.

---

## Struktur Proyek

```
.
├── docker-compose.yml       # definisi container (satu service: app)
├── Dockerfile                # multi-stage: build Vue → embed ke binary Go
├── go.mod
├── assets.go                 # //go:embed web/dist — WAJIB di root modul
├── .env.example
├── data/                      # berkas SQLite + sesi whatsmeow (tidak masuk git)
├── cmd/server/main.go         # entrypoint: wiring semua package + seed admin
├── internal/
│   ├── config/                # baca environment variable
│   ├── db/                    # buka SQLite + migration runner
│   ├── cryptutil/              # enkripsi AES-256-GCM buat secret di DB
│   ├── auth/                   # hashing password + JWT
│   ├── aiprovider/              # client OpenAI-compatible & Gemini
│   ├── store/                   # repository layer — semua query SQL
│   ├── bot/                     # orkestrasi: prompt guard + {{variable}} + panggil provider
│   ├── conversation/             # hub: log pesan masuk, rate limit, putuskan auto-reply
│   ├── gateway/whatsapp/          # integrasi whatsmeow (pairing, kirim, terima)
│   ├── gateway/telegram/          # integrasi Telegram Bot API (webhook)
│   └── httpapi/                   # route & handler REST
├── web/                        # Vue 3 + Vite + Pinia + Vue Router
│   └── src/
│       ├── views/                # satu file per halaman
│       ├── composables/           # mis. useOnline (deteksi offline)
│       ├── stores/                 # Pinia (auth)
│       └── api/client.js           # wrapper fetch: auth header, timeout, error handling
└── docs/API.md                 # referensi endpoint REST
```

Aturan bisnisnya sengaja dikumpulkan di `internal/bot` dan `internal/conversation`, bukan tersebar di `internal/httpapi`. Handler HTTP hanya membaca input, memanggil package domain, lalu menampilkan hasilnya — sehingga logika yang menentukan "kapan bot boleh membalas" hanya ada di satu tempat.

---

## Troubleshooting

**Bot tidak membalas padahal auto-reply sudah aktif**

Cek log container (`docker compose logs app`). Penyebab yang paling sering: provider AI menolak request (API key salah/kedaluwarsa), atau — khusus WhatsApp — JID pengirim berupa **LID** (identitas privasi WhatsApp) yang gagal dibalas. Versi saat ini sudah membalas ke JID persis yang mengirim pesan (bukan direkonstruksi dari nomor telepon), jadi kalau masih terjadi, kemungkinan besar penyebabnya di sisi provider AI, bukan WhatsApp.

**Respons error dari API cuma muncul "error code: 502" tanpa pesan yang jelas**

Ini gejala aplikasi berjalan di belakang Cloudflare (atau *reverse proxy* lain) yang meng-*intercept* status `5xx` dari origin dan menimpa isinya dengan halaman generik sendiri. Endpoint yang berpotensi gagal karena kondisi provider (test koneksi, ambil daftar model, chat playground) sudah sengaja membalas **400**, bukan 502, untuk menghindari ini — kalau menambah endpoint baru yang bisa gagal karena provider eksternal, pakai pola yang sama.

**QR WhatsApp tidak ke-*refresh*, status masih "Disconnected" padahal HP sudah connect**

Sebelum diperbaiki, polling QR yang gagal (karena sesi sudah tersambung) ditangani sebagai *error* tanpa me-*refresh* tabel. Versi saat ini aktif mengecek status tiap 3 detik selagi modal QR terbuka dan auto-*refresh* begitu status jadi `connected`. Kalau masih nyangkut, coba tutup & buka lagi modalnya, atau *refresh* halaman.

**Base URL / API key ke-isi otomatis dengan username & password login**

Itu perilaku *autofill* browser, bukan bug aplikasi — browser memasangkan input `type="password"` dengan input teks sebelumnya sebagai kredensial tersimpan. Field-field secret (API key, token bot) sudah ditandai `autocomplete="off"` / `autocomplete="new-password"`, tapi beberapa browser tetap keras kepala. Kosongkan manual kalau kejadian.

**Traefik tidak merutekan domain padahal config sudah benar (`404 page not found`)**

Kalau file provider Traefik tidak *auto-reload* meski `file.watch=true`, restart container Traefik-nya (`docker restart <nama-container-traefik>`) — biasanya file-watcher tidak menangkap perubahan yang dilakukan lewat penulisan langsung ke file.

**Gagal hapus Chatbot atau AI Provider (`409 Conflict`)**

Ini bukan bug — lihat [Konsep Utama](#konsep-utama) #3. Bot tidak bisa dihapus selama masih di-*binding* ke sesi WhatsApp/Telegram (lepas dulu lewat menu Chat Gateway), dan Provider tidak bisa dihapus selama masih dipakai Chatbot manapun.

---

## Lisensi

Proyek uji coba, bebas dipakai.
