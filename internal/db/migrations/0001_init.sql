CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS ai_providers (
    id             TEXT PRIMARY KEY,
    name           TEXT NOT NULL,
    type           TEXT NOT NULL, -- openai | deepseek | ollama | custom | gemini
    base_url       TEXT NOT NULL DEFAULT '',
    api_key_enc    TEXT NOT NULL DEFAULT '',
    default_model  TEXT NOT NULL DEFAULT '',
    is_active      INTEGER NOT NULL DEFAULT 1,
    created_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS bots (
    id             TEXT PRIMARY KEY,
    name           TEXT NOT NULL,
    provider_id    TEXT NOT NULL REFERENCES ai_providers(id) ON DELETE CASCADE,
    model          TEXT NOT NULL DEFAULT '',
    system_prompt  TEXT NOT NULL DEFAULT '',
    temperature    REAL NOT NULL DEFAULT 0.7,
    max_tokens     INTEGER NOT NULL DEFAULT 1024,
    is_active      INTEGER NOT NULL DEFAULT 1,
    created_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS gateway_sessions (
    id                  TEXT PRIMARY KEY,
    kind                TEXT NOT NULL, -- whatsapp | telegram
    label               TEXT NOT NULL DEFAULT '',
    status              TEXT NOT NULL DEFAULT 'disconnected',
    wa_jid              TEXT,
    device_jid          TEXT,
    telegram_token_enc  TEXT,
    telegram_username   TEXT,
    bot_id              TEXT REFERENCES bots(id) ON DELETE SET NULL,
    auto_reply          INTEGER NOT NULL DEFAULT 1,
    created_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS conversations (
    id               TEXT PRIMARY KEY,
    session_id       TEXT NOT NULL REFERENCES gateway_sessions(id) ON DELETE CASCADE,
    contact_id       TEXT NOT NULL,
    contact_name     TEXT NOT NULL DEFAULT '',
    auto_reply       INTEGER NOT NULL DEFAULT 1,
    last_message_at  TIMESTAMP,
    created_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(session_id, contact_id)
);

CREATE TABLE IF NOT EXISTS messages (
    id               TEXT PRIMARY KEY,
    conversation_id  TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    direction        TEXT NOT NULL, -- in | out
    sender           TEXT NOT NULL, -- user | bot | admin
    content          TEXT NOT NULL,
    provider_meta    TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id, created_at);

CREATE TABLE IF NOT EXISTS playground_sessions (
    id          TEXT PRIMARY KEY,
    bot_id      TEXT NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    title       TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS playground_messages (
    id                      TEXT PRIMARY KEY,
    playground_session_id  TEXT NOT NULL REFERENCES playground_sessions(id) ON DELETE CASCADE,
    role                    TEXT NOT NULL, -- user | assistant
    content                 TEXT NOT NULL,
    created_at              TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_playground_messages_session ON playground_messages(playground_session_id, created_at);

CREATE TABLE IF NOT EXISTS settings (
    key    TEXT PRIMARY KEY,
    value  TEXT NOT NULL DEFAULT ''
);
