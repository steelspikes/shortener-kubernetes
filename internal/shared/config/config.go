package config

import (
	"os"
	"strconv"
	"time"
)

// String devuelve la variable de entorno key, o def si no está definida.
func String(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

// Int devuelve la variable de entorno key como entero, o def si falta o es inválida.
func Int(key string, def int) int {
	if n, err := strconv.Atoi(os.Getenv(key)); err == nil {
		return n
	}
	return def
}

// Duration devuelve la variable de entorno key como time.Duration, o def.
func Duration(key string, def time.Duration) time.Duration {
	if d, err := time.ParseDuration(os.Getenv(key)); err == nil {
		return d
	}
	return def
}
