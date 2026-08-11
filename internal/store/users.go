package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("store: data tidak ditemukan")

type User struct {
	ID           string
	Username     string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (s *Store) CreateUser(ctx context.Context, username, passwordHash string) (*User, error) {
	u := &User{ID: uuid.NewString(), Username: username, PasswordHash: passwordHash}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (id, username, password_hash) VALUES (?, ?, ?)`,
		u.ID, u.Username, u.PasswordHash)
	if err != nil {
		return nil, err
	}
	return s.GetUserByID(ctx, u.ID)
}

func (s *Store) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	return s.scanUser(s.db.QueryRowContext(ctx, `
		SELECT id, username, password_hash, created_at, updated_at FROM users WHERE username = ?`, username))
}

func (s *Store) GetUserByID(ctx context.Context, id string) (*User, error) {
	return s.scanUser(s.db.QueryRowContext(ctx, `
		SELECT id, username, password_hash, created_at, updated_at FROM users WHERE id = ?`, id))
}

func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM users`).Scan(&n)
	return n, err
}

func (s *Store) scanUser(row *sql.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
