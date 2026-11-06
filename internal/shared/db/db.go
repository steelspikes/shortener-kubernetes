package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DSN arma la cadena de conexión de la base de datos de links (DB_*).
func DSN() string {
	return buildDSN("DB", "5432", "shortener")
}

// Connect abre el pool a la base de datos de links (solo tabla links).
// Los clics viven en otra base: ver ConnectClicks en dbclicks.go.
func Connect(ctx context.Context) (*pgxpool.Pool, error) {
	return connect(ctx, DSN())
}
