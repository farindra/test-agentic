package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

type Bot struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	ProviderID   string    `json:"provider_id"`
	Model        string    `json:"model"`
	SystemPrompt string    `json:"system_prompt"`
	Temperature  float64   `json:"temperature"`
	MaxTokens    int       `json:"max_tokens"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (s *Store) CreateBot(ctx context.Context, b Bot) (*Bot, error) {
	id := uuid.NewString()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO bots (id, name, provider_id, model, system_prompt, temperature, max_tokens, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, b.Name, b.ProviderID, b.Model, b.SystemPrompt, b.Temperature, b.MaxTokens, boolToInt(b.IsActive))
	if err != nil {
		return nil, err
	}
	return s.GetBot(ctx, id)
}

func (s *Store) UpdateBot(ctx context.Context, b Bot) (*Bot, error) {
	res, err := s.db.ExecContext(ctx, `
		UPDATE bots SET name=?, provider_id=?, model=?, system_prompt=?, temperature=?, max_tokens=?, is_active=?, updated_at=CURRENT_TIMESTAMP
		WHERE id=?`,
		b.Name, b.ProviderID, b.Model, b.SystemPrompt, b.Temperature, b.MaxTokens, boolToInt(b.IsActive), b.ID)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrNotFound
	}
	return s.GetBot(ctx, b.ID)
}

// CountBotsByProvider: dipakai buat nolak hapus provider yang masih dipakai
// chatbot (bots.provider_id ON DELETE CASCADE — tanpa pengecekan ini, hapus
// provider diem-diem ngehapus semua chatbot yang pakai dia juga).
func (s *Store) CountBotsByProvider(ctx context.Context, providerID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM bots WHERE provider_id = ?`, providerID).Scan(&n)
	return n, err
}

func (s *Store) DeleteBot(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM bots WHERE id=?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

const botSelect = `SELECT id, name, provider_id, model, system_prompt, temperature, max_tokens, is_active, created_at, updated_at FROM bots`

func (s *Store) GetBot(ctx context.Context, id string) (*Bot, error) {
	return scanBot(s.db.QueryRowContext(ctx, botSelect+` WHERE id = ?`, id))
}

func (s *Store) ListBots(ctx context.Context) ([]Bot, error) {
	rows, err := s.db.QueryContext(ctx, botSelect+` ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Bot
	for rows.Next() {
		var b Bot
		var isActive int
		if err := rows.Scan(&b.ID, &b.Name, &b.ProviderID, &b.Model, &b.SystemPrompt, &b.Temperature, &b.MaxTokens, &isActive, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		b.IsActive = isActive != 0
		out = append(out, b)
	}
	return out, rows.Err()
}

func scanBot(row *sql.Row) (*Bot, error) {
	var b Bot
	var isActive int
	err := row.Scan(&b.ID, &b.Name, &b.ProviderID, &b.Model, &b.SystemPrompt, &b.Temperature, &b.MaxTokens, &isActive, &b.CreatedAt, &b.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	b.IsActive = isActive != 0
	return &b, nil
}
