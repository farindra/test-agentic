package store

import (
	"context"
	"database/sql"
	"errors"
)

// Variables: key-value custom milik admin (mis. jam_buka, alamat_toko) yang
// bisa dipakai di system prompt chatbot lewat placeholder {{nama_variable}}
// — lihat bot.Orchestrator.Reply.

func (s *Store) GetVariable(ctx context.Context, key string) (string, error) {
	var v string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM variables WHERE key=?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return v, err
}

func (s *Store) SetVariable(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO variables (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
	return err
}

func (s *Store) DeleteVariable(ctx context.Context, key string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM variables WHERE key=?`, key)
	return err
}

func (s *Store) ListVariables(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM variables`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}
