package db

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"shortener/internal/shared/config"
)

// Códigos de error de Postgres (SQLSTATE).
var UNIQUE_VIOLATION = "23505"
var FOREIGN_KEY_VIOLATION = "23503"

// IsPgCode indica si err es un error de Postgres con el código dado.
func IsPgCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}

// buildDSN arma la cadena de conexión leyendo <prefix>_USER, _PASSWORD, _HOST, _PORT, _NAME y _SSLMODE.
func buildDSN(prefix, defaultPort, defaultName string) string {
	u := url.URL{
		Scheme: "postgres",
		User: url.UserPassword(
			config.String(prefix+"_USER", "postgres"),
			config.String(prefix+"_PASSWORD", "postgres"),
		),
		Host: fmt.Sprintf("%s:%s", config.String(prefix+"_HOST", "localhost"), config.String(prefix+"_PORT", defaultPort)),
		Path: config.String(prefix+"_NAME", defaultName),
	}
	q := u.Query()
	q.Set("sslmode", config.String(prefix+"_SSLMODE", "disable"))
	u.RawQuery = q.Encode()
	return u.String()
}

// connect abre un pool de conexiones a Postgres y verifica que responda.
func connect(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
