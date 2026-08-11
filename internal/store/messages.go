package store

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	Direction      string    `json:"direction"` // in | out
	Sender         string    `json:"sender"`    // user | bot | admin
	Content        string    `json:"content"`
	ProviderMeta   string    `json:"provider_meta,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// AddMessage nyimpen pesan lalu nge-update last_message_at di percakapannya,
// dalam satu transaksi biar dua tabel itu konsisten.
func (s *Store) AddMessage(ctx context.Context, m Message) (*Message, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	id := uuid.NewString()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO messages (id, conversation_id, direction, sender, content, provider_meta)
		VALUES (?, ?, ?, ?, ?, ?)`,
		id, m.ConversationID, m.Direction, m.Sender, m.Content, m.ProviderMeta); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE conversations SET last_message_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		m.ConversationID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	var out Message
	err = s.db.QueryRowContext(ctx, `
		SELECT id, conversation_id, direction, sender, content, provider_meta, created_at FROM messages WHERE id=?`, id).
		Scan(&out.ID, &out.ConversationID, &out.Direction, &out.Sender, &out.Content, &out.ProviderMeta, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Store) ListMessages(ctx context.Context, conversationID string, limit int) ([]Message, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, conversation_id, direction, sender, content, provider_meta, created_at
		FROM messages WHERE conversation_id = ? ORDER BY created_at DESC LIMIT ?`, conversationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Direction, &m.Sender, &m.Content, &m.ProviderMeta, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	// dibalik urutannya jadi ASC (kronologis) biar gampang dipajang di UI —
	// query di atas DESC + LIMIT supaya yang diambil selalu N pesan TERBARU.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, rows.Err()
}
