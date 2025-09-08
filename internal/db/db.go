package db

import (
        "context"
        "encoding/base64"
        "errors"
        "fmt"
        "os"
        "strings"
        "time"

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
        sql, err := os.ReadFile("migrations/001_create_notifications.sql")
        if err != nil {
                return err
        }
        _, err = s.pool.Exec(ctx, string(sql))
        return err
}

var ErrDuplicate = errors.New("duplicate notification")

func (s *Store) CreateNotification(ctx context.Context, key string, in model.NotificationInput) error {
        ct, err := s.pool.Exec(ctx, `INSERT INTO notifications (sender, recipient, message, idempotency_key) VALUES ($1,$2,$3,$4) ON CONFLICT (idempotency_key) DO NOTHING`,
                in.Sender, in.Recipient, in.Message, key)
        if err != nil {
                return err
        }
        if ct.RowsAffected() == 0 {
                return ErrDuplicate
        }
        return nil
}

type Cursor struct {
        CreatedAt time.Time
        ID        int64
}

func encodeCursor(c Cursor) string {
        b := []byte(c.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + fmt.Sprintf("%d", c.ID))
        return base64.StdEncoding.EncodeToString(b)
}

func parseCursor(s string) (Cursor, error) {
        data, err := base64.StdEncoding.DecodeString(s)
        if err != nil {
                return Cursor{}, err
        }
        parts := strings.SplitN(string(data), "|", 2)
        if len(parts) != 2 {
                return Cursor{}, errors.New("invalid cursor")
        }
        t, err := time.Parse(time.RFC3339Nano, parts[0])
        if err != nil {
                return Cursor{}, err
        }
        var id int64
        if _, err := fmt.Sscanf(parts[1], "%d", &id); err != nil {
                return Cursor{}, err
        }
        return Cursor{CreatedAt: t, ID: id}, nil
}

type ListFilter struct {
        Sender    *string
        Recipient *string
        Limit     int
        After     *Cursor
}

func (s *Store) ListNotifications(ctx context.Context, f ListFilter) ([]model.Notification, *string, error) {
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
        if f.After != nil {
                cond = append(cond, fmt.Sprintf("(created_at,id) < ($%d,$%d)", len(args)+1, len(args)+2))
                args = append(args, f.After.CreatedAt, f.After.ID)
        }
        if len(cond) > 0 {
                query += " WHERE " + strings.Join(cond, " AND ")
        }
        query += fmt.Sprintf(" ORDER BY created_at DESC, id DESC LIMIT $%d", len(args)+1)
        args = append(args, f.Limit)
        rows, err := s.pool.Query(ctx, query, args...)
        if err != nil {
                return nil, nil, err
        }
        defer rows.Close()
        var out []model.Notification
        for rows.Next() {
                var n model.Notification
                if err := rows.Scan(&n.ID, &n.Sender, &n.Recipient, &n.Message, &n.CreatedAt); err != nil {
                        return nil, nil, err
                }
                out = append(out, n)
        }
        if err := rows.Err(); err != nil {
                return nil, nil, err
        }
        var next *string
        if len(out) == f.Limit {
                c := encodeCursor(Cursor{CreatedAt: out[len(out)-1].CreatedAt, ID: out[len(out)-1].ID})
                next = &c
        }
        return out, next, nil
}

func (s *Store) ParseCursor(str string) (*Cursor, error) {
        if str == "" {
                return nil, nil
        }
        c, err := parseCursor(str)
        if err != nil {
                return nil, err
        }
        return &c, nil
}

func (s *Store) Ping(ctx context.Context) error {
        return s.pool.Ping(ctx)
}
