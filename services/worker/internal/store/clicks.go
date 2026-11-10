package store

import (
	"context"
	"shortener/internal/shared/jobs"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Click struct {
	Code      string    `json:"code"`
	ClickedAt time.Time `json:"clicked_at"`
	IPHash    string    `json:"ip_hash"`
	UserAgent string    `json:"user_agent"`
	Referer   string    `json:"referer"`
	Browser   string    `json:"browser"`
}

// ClickStore accede a la tabla clicks de la base de datos de clics.
type ClickStore struct {
	pool *pgxpool.Pool
}

func NewClickStore(pool *pgxpool.Pool) *ClickStore {
	return &ClickStore{pool: pool}
}

func (c *ClickStore) SaveClick(ctx context.Context, item jobs.ClickJob) error {
	_, err := c.pool.Exec(
		ctx,
		"INSERT INTO clicks (code, clicked_at, ip_hash, user_agent, referer, browser) VALUES ($1, $2, $3, $4, $5, $6)",
		item.Code, item.ClickedAt, item.IPHash, item.UserAgent, item.Referer, item.Browser)

	return err
}

func (c *ClickStore) ListClicks(ctx context.Context) ([]Click, error) {
	rows, err := c.pool.Query(ctx, "SELECT code, clicked_at, ip_hash, user_agent, referer, browser FROM clicks")

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	clicks := []Click{}

	for rows.Next() {
		var c Click
		if err := rows.Scan(&c.Code, &c.ClickedAt, &c.IPHash, &c.UserAgent, &c.Referer, &c.Browser); err != nil {
			return nil, err
		}
		clicks = append(clicks, c)
	}

	return clicks, nil
}

func (s *ClickStore) Check(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	err := s.pool.Ping(ctx)
	return err
}
