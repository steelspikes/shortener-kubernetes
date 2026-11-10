package redis

import (
	"context"
	"errors"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"shortener/internal/shared/config"
)

// Nil es el error que devuelven los GET cuando la key no existe (cache miss).
// Úsalo así: if errors.Is(err, redis.Nil) { ... }
var Nil = goredis.Nil

// Client envuelve al cliente de go-redis. Al estar embebido, puedes llamar
// directamente cualquier comando: c.Get(ctx, k), c.Set(ctx, k, v, ttl), c.Del(ctx, k)...
type Client struct {
	*goredis.Client
}

// Connect abre el cliente usando variables de entorno y verifica que Redis responda.
func Connect(ctx context.Context) (*Client, error) {
	REDIS_PORT := config.String("REDIS_PORT", "6379")

	c := goredis.NewClient(&goredis.Options{
		Addr:     config.String("REDIS_ADDR", "localhost") + ":" + REDIS_PORT,
		Password: config.String("REDIS_PASSWORD", ""),
		DB:       config.Int("REDIS_DB", 0),
	})
	if err := c.Ping(ctx).Err(); err != nil {
		c.Close()
		return nil, err
	}
	return &Client{c}, nil
}

// DefaultTTL es el tiempo de vida por defecto de las keys de caché.
func DefaultTTL() time.Duration {
	return config.Duration("CACHE_TTL", time.Minute)
}

// IsMiss indica si el error es un cache miss (la key no existe) y no un fallo real.
func IsMiss(err error) bool {
	return errors.Is(err, goredis.Nil)
}

func (c *Client) Check(ctx context.Context) error {
	return c.Ping(ctx).Err()
}
