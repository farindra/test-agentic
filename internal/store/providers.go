package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ProviderType = jenis AI provider yang didukung. "openai", "deepseek", dan
// "ollama" semuanya dilayani lewat client OpenAI-compatible yang sama
// (base_url beda), sementara "gemini" pakai format request sendiri.
type ProviderType string

const (
	ProviderOpenAI   ProviderType = "openai"
	ProviderDeepSeek ProviderType = "deepseek"
	ProviderOllama   ProviderType = "ollama"
	ProviderGemini   ProviderType = "gemini"
	ProviderCustom   ProviderType = "custom"
)

type Provider struct {
	ID           string
	Name         string
	Type         ProviderType
	BaseURL      string
	APIKey       string // plaintext, sudah didekripsi
	DefaultModel string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (s *Store) CreateProvider(ctx context.Context, p Provider) (*Provider, error) {
	encKey, err := s.box.Encrypt(p.APIKey)
	if err != nil {
		return nil, err
	}
	id := uuid.NewString()
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO ai_providers (id, name, type, base_url, api_key_enc, default_model, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, p.Name, string(p.Type), p.BaseURL, encKey, p.DefaultModel, boolToInt(p.IsActive))
	if err != nil {
		return nil, err
	}
	return s.GetProvider(ctx, id)
}

func (s *Store) UpdateProvider(ctx context.Context, p Provider) (*Provider, error) {
	encKey, err := s.box.Encrypt(p.APIKey)
	if err != nil {
		return nil, err
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE ai_providers SET name=?, type=?, base_url=?, api_key_enc=?, default_model=?, is_active=?, updated_at=CURRENT_TIMESTAMP
		WHERE id=?`,
		p.Name, string(p.Type), p.BaseURL, encKey, p.DefaultModel, boolToInt(p.IsActive), p.ID)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrNotFound
	}
	return s.GetProvider(ctx, p.ID)
}

func (s *Store) DeleteProvider(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM ai_providers WHERE id=?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) GetProvider(ctx context.Context, id string) (*Provider, error) {
	return s.scanProvider(s.db.QueryRowContext(ctx, providerSelect+` WHERE id = ?`, id))
}

func (s *Store) ListProviders(ctx context.Context) ([]Provider, error) {
	rows, err := s.db.QueryContext(ctx, providerSelect+` ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Provider
	for rows.Next() {
		p, err := s.scanProviderRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

const providerSelect = `SELECT id, name, type, base_url, api_key_enc, default_model, is_active, created_at, updated_at FROM ai_providers`

func (s *Store) scanProvider(row *sql.Row) (*Provider, error) {
	var p Provider
	var typ string
	var isActive int
	var encKey string
	err := row.Scan(&p.ID, &p.Name, &typ, &p.BaseURL, &encKey, &p.DefaultModel, &isActive, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	p.Type = ProviderType(typ)
	p.IsActive = isActive != 0
	p.APIKey, err = s.box.Decrypt(encKey)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) scanProviderRows(rows *sql.Rows) (*Provider, error) {
	var p Provider
	var typ string
	var isActive int
	var encKey string
	if err := rows.Scan(&p.ID, &p.Name, &typ, &p.BaseURL, &encKey, &p.DefaultModel, &isActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	p.Type = ProviderType(typ)
	p.IsActive = isActive != 0
	apiKey, err := s.box.Decrypt(encKey)
	if err != nil {
		return nil, err
	}
	p.APIKey = apiKey
	return &p, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
