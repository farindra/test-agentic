package store

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type PlaygroundSession struct {
	ID        string    `json:"id"`
	BotID     string    `json:"bot_id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

type PlaygroundMessage struct {
	ID                  string    `json:"id"`
	PlaygroundSessionID string    `json:"playground_session_id"`
	Role                string    `json:"role"` // user | assistant
	Content             string    `json:"content"`
	CreatedAt           time.Time `json:"created_at"`
}

func (s *Store) CreatePlaygroundSession(ctx context.Context, botID, title string) (*PlaygroundSession, error) {
	id := uuid.NewString()
	_, err := s.db.ExecContext(ctx, `INSERT INTO playground_sessions (id, bot_id, title) VALUES (?, ?, ?)`, id, botID, title)
	if err != nil {
		return nil, err
	}
	var ps PlaygroundSession
	err = s.db.QueryRowContext(ctx, `SELECT id, bot_id, title, created_at FROM playground_sessions WHERE id=?`, id).
		Scan(&ps.ID, &ps.BotID, &ps.Title, &ps.CreatedAt)
	return &ps, err
}

func (s *Store) ListPlaygroundSessions(ctx context.Context) ([]PlaygroundSession, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, bot_id, title, created_at FROM playground_sessions ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PlaygroundSession
	for rows.Next() {
		var ps PlaygroundSession
		if err := rows.Scan(&ps.ID, &ps.BotID, &ps.Title, &ps.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, ps)
	}
	return out, rows.Err()
}

func (s *Store) AddPlaygroundMessage(ctx context.Context, sessionID, role, content string) (*PlaygroundMessage, error) {
	id := uuid.NewString()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO playground_messages (id, playground_session_id, role, content) VALUES (?, ?, ?, ?)`,
		id, sessionID, role, content)
	if err != nil {
		return nil, err
	}
	var pm PlaygroundMessage
	err = s.db.QueryRowContext(ctx, `
		SELECT id, playground_session_id, role, content, created_at FROM playground_messages WHERE id=?`, id).
		Scan(&pm.ID, &pm.PlaygroundSessionID, &pm.Role, &pm.Content, &pm.CreatedAt)
	return &pm, err
}

func (s *Store) ListPlaygroundMessages(ctx context.Context, sessionID string) ([]PlaygroundMessage, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, playground_session_id, role, content, created_at
		FROM playground_messages WHERE playground_session_id = ? ORDER BY created_at ASC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PlaygroundMessage
	for rows.Next() {
		var pm PlaygroundMessage
		if err := rows.Scan(&pm.ID, &pm.PlaygroundSessionID, &pm.Role, &pm.Content, &pm.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, pm)
	}
	return out, rows.Err()
}
