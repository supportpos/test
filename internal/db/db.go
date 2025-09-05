package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"notificationservice/internal/model"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS notifications (
        id SERIAL PRIMARY KEY,
        sender TEXT NOT NULL,
        recipient TEXT NOT NULL,
        message TEXT NOT NULL,
        hash TEXT UNIQUE NOT NULL,
        created_at TIMESTAMPTZ NOT NULL DEFAULT now()
    )`)
	return err
}

func hashNotification(n model.NotificationInput) string {
	h := sha256.New()
	h.Write([]byte(n.Sender))
	h.Write([]byte{0})
	h.Write([]byte(n.Recipient))
	h.Write([]byte{0})
	h.Write([]byte(n.Message))
	return hex.EncodeToString(h.Sum(nil))
}

func (s *Store) CreateNotification(ctx context.Context, in model.NotificationInput) error {
	hash := hashNotification(in)
	_, err := s.pool.Exec(ctx, `INSERT INTO notifications (sender, recipient, message, hash) VALUES ($1,$2,$3,$4) ON CONFLICT (hash) DO NOTHING`,
		in.Sender, in.Recipient, in.Message, hash)
	return err
}

type ListFilter struct {
	Sender    *string
	Recipient *string
	Limit     int
	Offset    int
}

func (s *Store) ListNotifications(ctx context.Context, f ListFilter) ([]model.Notification, error) {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 50
	}
	query := `SELECT id, sender, recipient, message, created_at FROM notifications`
	var args []interface{}
	var cond []string
	if f.Sender != nil {
		cond = append(cond, fmt.Sprintf("sender=$%d", len(args)+1))
		args = append(args, *f.Sender)
	}
	if f.Recipient != nil {
		cond = append(cond, fmt.Sprintf("recipient=$%d", len(args)+1))
		args = append(args, *f.Recipient)
	}
	if len(cond) > 0 {
		query += " WHERE " + strings.Join(cond, " AND ")
	}
	query += fmt.Sprintf(" ORDER BY id DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, f.Limit, f.Offset)
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Notification
	for rows.Next() {
		var n model.Notification
		if err := rows.Scan(&n.ID, &n.Sender, &n.Recipient, &n.Message, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}
