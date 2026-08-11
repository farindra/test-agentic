package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

type SessionKind string

const (
	KindWhatsApp SessionKind = "whatsapp"
	KindTelegram SessionKind = "telegram"
)

type GatewaySession struct {
	ID               string
	Kind             SessionKind
	Label            string
	Status           string
	WaJID            *string
	DeviceJID        *string
	TelegramToken    string // plaintext, sudah didekripsi (kosong kalau kind != telegram)
	TelegramUsername *string
	BotID            *string
	AutoReply        bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (s *Store) CreateSession(ctx context.Context, sess GatewaySession) (*GatewaySession, error) {
	encToken, err := s.box.Encrypt(sess.TelegramToken)
	if err != nil {
		return nil, err
	}
	id := uuid.NewString()
	if sess.Status == "" {
		sess.Status = "disconnected"
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO gateway_sessions (id, kind, label, status, telegram_token_enc, bot_id, auto_reply)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, string(sess.Kind), sess.Label, sess.Status, encToken, sess.BotID, boolToInt(sess.AutoReply))
	if err != nil {
		return nil, err
	}
	return s.GetSession(ctx, id)
}

func (s *Store) UpdateSessionBinding(ctx context.Context, id string, botID *string, autoReply bool) (*GatewaySession, error) {
	res, err := s.db.ExecContext(ctx, `
		UPDATE gateway_sessions SET bot_id=?, auto_reply=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		botID, boolToInt(autoReply), id)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrNotFound
	}
	return s.GetSession(ctx, id)
}

func (s *Store) SetSessionStatus(ctx context.Context, id, status string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE gateway_sessions SET status=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, status, id)
	return err
}

// SetWhatsAppLinked simpan JID device setelah pairing WhatsApp sukses.
func (s *Store) SetWhatsAppLinked(ctx context.Context, id, deviceJID, waJID, status string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE gateway_sessions SET device_jid=?, wa_jid=?, status=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		deviceJID, waJID, status, id)
	return err
}

func (s *Store) SetWhatsAppUnlinked(ctx context.Context, id, status string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE gateway_sessions SET device_jid=NULL, wa_jid=NULL, status=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		status, id)
	return err
}

func (s *Store) SetTelegramUsername(ctx context.Context, id, username string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE gateway_sessions SET telegram_username=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, username, id)
	return err
}

func (s *Store) DeleteSession(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM gateway_sessions WHERE id=?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

const sessionSelect = `SELECT id, kind, label, status, wa_jid, device_jid, telegram_token_enc, telegram_username, bot_id, auto_reply, created_at, updated_at FROM gateway_sessions`

func (s *Store) GetSession(ctx context.Context, id string) (*GatewaySession, error) {
	return s.scanSession(s.db.QueryRowContext(ctx, sessionSelect+` WHERE id = ?`, id))
}

func (s *Store) ListSessions(ctx context.Context, kind SessionKind) ([]GatewaySession, error) {
	q := sessionSelect
	args := []any{}
	if kind != "" {
		q += ` WHERE kind = ?`
		args = append(args, string(kind))
	}
	q += ` ORDER BY created_at`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GatewaySession
	for rows.Next() {
		sess, err := s.scanSessionRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *sess)
	}
	return out, rows.Err()
}

func (s *Store) scanSession(row *sql.Row) (*GatewaySession, error) {
	var g GatewaySession
	var kind, encToken string
	var autoReply int
	err := row.Scan(&g.ID, &kind, &g.Label, &g.Status, &g.WaJID, &g.DeviceJID, &encToken, &g.TelegramUsername, &g.BotID, &autoReply, &g.CreatedAt, &g.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	g.Kind = SessionKind(kind)
	g.AutoReply = autoReply != 0
	g.TelegramToken, err = s.box.Decrypt(encToken)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (s *Store) scanSessionRows(rows *sql.Rows) (*GatewaySession, error) {
	var g GatewaySession
	var kind, encToken string
	var autoReply int
	if err := rows.Scan(&g.ID, &kind, &g.Label, &g.Status, &g.WaJID, &g.DeviceJID, &encToken, &g.TelegramUsername, &g.BotID, &autoReply, &g.CreatedAt, &g.UpdatedAt); err != nil {
		return nil, err
	}
	g.Kind = SessionKind(kind)
	g.AutoReply = autoReply != 0
	tok, err := s.box.Decrypt(encToken)
	if err != nil {
		return nil, err
	}
	g.TelegramToken = tok
	return &g, nil
}
