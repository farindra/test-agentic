// Package store adalah repository layer: satu tempat semua query SQL ke
// SQLite. Secret (API key provider, token bot Telegram) dienkripsi/dekripsi
// transparan di sini lewat cryptutil.Box, jadi package pemanggil (httpapi,
// bot, gateway) selalu kerja dengan plaintext.
package store

import (
	"database/sql"

	"test-agentic/internal/cryptutil"
)

type Store struct {
	db  *sql.DB
	box *cryptutil.Box
}

func New(db *sql.DB, box *cryptutil.Box) *Store {
	return &Store{db: db, box: box}
}
