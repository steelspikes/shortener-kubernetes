package store

import (
	"context"
	"errors"
	"shortener/internal/shared/db"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrCodeTaken = errors.New("Code already taken")
var ErrNotFound = errors.New("Element not found")

type Link struct {
	ID        int64      `json:"id"`
	Code      string     `json:"code"`
	URL       string     `json:"url"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at"` // nil = no expira
}

type NewLink struct {
	URL       string     `json:"url"`
	Code      string     `json:"code"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type LinkStore struct {
	pool *pgxpool.Pool
}

func NewLinkStore(pool *pgxpool.Pool) *LinkStore {
	return &LinkStore{pool: pool}
}

// List devuelve todos los links, del más reciente al más antiguo.
func (s *LinkStore) ListAll(ctx context.Context) ([]Link, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, code, url, created_at, expires_at
		   FROM links
		  ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	links := []Link{}
	for rows.Next() {
		var l Link
		if err := rows.Scan(&l.ID, &l.Code, &l.URL, &l.CreatedAt, &l.ExpiresAt); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, rows.Err()
}

func (s *LinkStore) Create(ctx context.Context, link NewLink) (Link, error) {
	var l Link
	err := s.pool.QueryRow(ctx,
		`INSERT INTO links (code, url, expires_at)
		  VALUES ($1, $2, $3)
		  RETURNING id, code, url, created_at, expires_at`, link.Code, link.URL, link.ExpiresAt).Scan(&l.ID, &l.Code, &l.URL, &l.CreatedAt, &l.ExpiresAt)

	if err != nil {
		if db.IsPgCode(err, db.UNIQUE_VIOLATION) {
			return Link{}, ErrCodeTaken
		}
		return Link{}, err
	}

	return l, nil
}

func (s *LinkStore) List(ctx context.Context, code string) (Link, error) {
	var l Link
	err := s.pool.QueryRow(ctx,
		`SELECT id, code, url, created_at, expires_at
		   FROM links WHERE code = $1`, code).Scan(&l.ID, &l.Code, &l.URL, &l.CreatedAt, &l.ExpiresAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Link{}, ErrNotFound
		}
		return Link{}, err
	}
	return l, nil
}
