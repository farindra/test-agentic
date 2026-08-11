package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

type Conversation struct {
	ID            string     `json:"id"`
	SessionID     string     `json:"session_id"`
	ContactID     string     `json:"contact_id"`
	ContactName   string     `json:"contact_name"`
	AutoReply     bool       `json:"auto_reply"`
	LastMessageAt *time.Time `json:"last_message_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// GetOrCreateConversation dipanggil setiap ada pesan masuk dari gateway.
// contact_id + session_id itu unik (lihat migration), jadi INSERT OR IGNORE
// aman dipanggil berkali-kali dari goroutine event handler yang berbeda.
func (s *Store) GetOrCreateConversation(ctx context.Context, sessionID, contactID, contactName string) (*Conversation, error) {
	_, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO conversations (id, session_id, contact_id, contact_name)
		VALUES (?, ?, ?, ?)`,
		uuid.NewString(), sessionID, contactID, contactName)
	if err != nil {
		return nil, err
	}
	return s.scanConversation(s.db.QueryRowContext(ctx, conversationSelect+` WHERE session_id = ? AND contact_id = ?`, sessionID, contactID))
}

func (s *Store) GetConversation(ctx context.Context, id string) (*Conversation, error) {
	return s.scanConversation(s.db.QueryRowContext(ctx, conversationSelect+` WHERE id = ?`, id))
}

func (s *Store) ListConversations(ctx context.Context) ([]Conversation, error) {
	rows, err := s.db.QueryContext(ctx, conversationSelect+` ORDER BY COALESCE(last_message_at, created_at) DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Conversation
	for rows.Next() {
		c, err := scanConversationRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (s *Store) SetConversationAutoReply(ctx context.Context, id string, autoReply bool) error {
	res, err := s.db.ExecContext(ctx, `UPDATE conversations SET auto_reply=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, boolToInt(autoReply), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) touchConversation(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE conversations SET last_message_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, id)
	return err
}

const conversationSelect = `SELECT id, session_id, contact_id, contact_name, auto_reply, last_message_at, created_at, updated_at FROM conversations`

func (s *Store) scanConversation(row *sql.Row) (*Conversation, error) {
	var c Conversation
	var autoReply int
	err := row.Scan(&c.ID, &c.SessionID, &c.ContactID, &c.ContactName, &autoReply, &c.LastMessageAt, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	c.AutoReply = autoReply != 0
	return &c, nil
}

func scanConversationRows(rows *sql.Rows) (*Conversation, error) {
	var c Conversation
	var autoReply int
	if err := rows.Scan(&c.ID, &c.SessionID, &c.ContactID, &c.ContactName, &autoReply, &c.LastMessageAt, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, err
	}
	c.AutoReply = autoReply != 0
	return &c, nil
}
