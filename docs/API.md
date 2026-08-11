# API Reference

Base URL: `https://test-agentic.farindra.com/api` (atau `http://localhost:8080/api` lokal).

Semua endpoint di bawah ini butuh header:

```
Authorization: Bearer <token>
```

kecuali `POST /auth/login` dan webhook Telegram (`POST /gateway/telegram/webhook/:id`,
diverifikasi lewat header `X-Telegram-Bot-Api-Secret-Token`, bukan JWT).

Semua request/response berupa JSON. Error selalu berbentuk `{"error": "pesan"}`.

---

## Auth

### `POST /auth/login`
```json
// request
{ "username": "admin", "password": "secret" }
// response 200
{ "token": "eyJ...", "username": "admin" }
// response 401
{ "error": "username atau password salah" }
```

### `GET /auth/me`
```json
// response 200
{ "id": "uuid", "username": "admin" }
```

---

## AI Provider

`Provider.type`: `openai` | `deepseek` | `gemini` | `ollama` | `custom`

### `GET /providers`
```json
{ "providers": [
  { "id": "uuid", "name": "OpenAI Prod", "type": "openai", "base_url": "",
    "default_model": "gpt-4o-mini", "is_active": true,
    "api_key_set": true, "api_key_preview": "••••ab12" }
]}
```
API key **tidak pernah** dibalikin utuh — cuma penanda `api_key_set` + 4
karakter terakhir.

### `POST /providers`
```json
// request
{ "name": "OpenAI Prod", "type": "openai", "base_url": "", "api_key": "sk-...",
  "default_model": "gpt-4o-mini", "is_active": true }
```
`base_url` wajib diisi buat `type: ollama` dan `type: custom` (gak ada
default resmi). Kosong buat `openai`/`deepseek`/`gemini` = pakai default
resmi provider itu.

### `PUT /providers/:id`
Sama seperti create. `api_key` boleh dikosongkan — kalau kosong, key lama
dipertahankan (gak perlu re-submit secret tiap edit).

### `DELETE /providers/:id`
### `POST /providers/:id/test`
Ngirim satu pesan ping ke provider, buat validasi kredensial dari UI.
```json
// response 200
{ "ok": true, "reply": "pong" }
// response 502
{ "error": "provider error: invalid api key" }
```

---

## Chatbot

### `GET /bots` / `POST /bots` / `PUT /bots/:id` / `DELETE /bots/:id`
```json
// POST/PUT request
{ "name": "CS Bot", "provider_id": "uuid", "model": "gpt-4o-mini",
  "system_prompt": "Kamu asisten CS yang ramah...",
  "temperature": 0.7, "max_tokens": 1024, "is_active": true }
```
`model` kosong = pakai `default_model` dari provider-nya.

---

## Playground

### `GET /playground/sessions` / `POST /playground/sessions`
```json
// POST request
{ "bot_id": "uuid", "title": "Test 1" }
```

### `GET /playground/sessions/:id/messages`
### `POST /playground/sessions/:id/chat`
```json
// request
{ "message": "Halo, jam buka kapan?" }
// response 200 — pesan assistant yang baru tersimpan
{ "id": "uuid", "playground_session_id": "uuid", "role": "assistant",
  "content": "Halo! Kami buka Senin-Sabtu jam 9-17.", "created_at": "..." }
```

---

## Chat Gateway — WhatsApp

### `GET /gateway/whatsapp/sessions` / `POST /gateway/whatsapp/sessions`
```json
// POST request
{ "label": "CS Toko A" }
```

### `GET /gateway/whatsapp/sessions/:id/qr`
```json
// response 200
{ "qr": "data:image/png;base64,...", "ttl_sec": 60 }
```
Refetch endpoint ini tiap `ttl_sec` detik sampai status sesi jadi
`connected` — QR WhatsApp berumur pendek dan diganti otomatis.

### `GET /gateway/whatsapp/sessions/:id/status`
```json
{ "connected": true, "state": "connected", "jid": "6281234567890" }
```

### `POST /gateway/whatsapp/sessions/:id/send`
```json
{ "phone": "081234567890", "message": "Halo dari test" }
```

### `PATCH /gateway/whatsapp/sessions/:id`
Binding ke chatbot + toggle auto-reply.
```json
{ "bot_id": "uuid", "auto_reply": true }
```
`bot_id: ""` melepas binding (sesi jadi gak auto-reply walau `auto_reply: true`).

### `DELETE /gateway/whatsapp/sessions/:id`
Logout dari WhatsApp + hapus sesi.

---

## Chat Gateway — Telegram

### `GET /gateway/telegram/sessions` / `POST /gateway/telegram/sessions`
```json
// POST request
{ "label": "CS Bot Telegram", "token": "123456:ABC-DEF..." }
```

### `POST /gateway/telegram/sessions/:id/activate`
Daftarin webhook Telegram (`PUBLIC_BASE_URL` + `/api/gateway/telegram/webhook/:id`)
dan ambil username bot lewat `getMe`.

### `POST /gateway/telegram/sessions/:id/deactivate`
Hapus webhook.

### `POST /gateway/telegram/sessions/:id/send`
```json
{ "chat_id": "123456789", "message": "Halo dari test" }
```

### `PATCH /gateway/telegram/sessions/:id`
Sama seperti WhatsApp: `{ "bot_id": "uuid", "auto_reply": true }`.

### `DELETE /gateway/telegram/sessions/:id`

---

## Inbox

### `GET /conversations`
```json
{ "conversations": [
  { "id": "uuid", "session_id": "uuid", "contact_id": "628111234567",
    "contact_name": "Budi", "auto_reply": true, "last_message_at": "..." }
]}
```

### `GET /conversations/:id/messages`
```json
{ "messages": [
  { "id": "uuid", "direction": "in", "sender": "user", "content": "Halo", "created_at": "..." },
  { "id": "uuid", "direction": "out", "sender": "bot", "content": "Halo juga!", "created_at": "..." }
]}
```

### `PATCH /conversations/:id`
Toggle auto-reply per percakapan (buat handover manual ke manusia tanpa
mematikan auto-reply di seluruh sesi gateway-nya).
```json
{ "auto_reply": false }
```

---

## Dashboard

### `GET /dashboard/summary`
```json
{ "providers_total": 2, "bots_total": 3,
  "whatsapp_sessions": 1, "whatsapp_connected": 1,
  "telegram_sessions": 1, "telegram_connected": 0,
  "conversations_total": 42 }
```

---

## Settings

### `GET /settings`
```json
{ "default_bot_id": "uuid" }
```

### `PUT /settings`
Body = object key-value, di-upsert semua.
```json
{ "default_bot_id": "uuid" }
```
