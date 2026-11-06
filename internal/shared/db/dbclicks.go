package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ClicksDSN arma la cadena de conexión de la base de datos de clics (CLICKS_DB_*).
// Es independiente de la base de links para mantener los servicios desacoplados.
func ClicksDSN() string {
	return buildDSN("CLICKS_DB", "5433", "shortener_clicks")
}

// ConnectClicks abre el pool a la base de datos de clics (solo tabla clicks).
func ConnectClicks(ctx context.Context) (*pgxpool.Pool, error) {
	return connect(ctx, ClicksDSN())
}
