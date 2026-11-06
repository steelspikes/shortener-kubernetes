// Package jobs define los mensajes que viajan por las colas, compartidos entre
// el productor (api) y el consumidor (worker).
package jobs

import "time"

// ClicksQueue es la cola donde el api publica los clics y el worker los consume.
const ClicksQueue = "clicks"

// ClickJob es un clic en un link corto. Refleja las columnas de la tabla clicks.
type ClickJob struct {
	Code      string    `json:"code"`
	ClickedAt time.Time `json:"clicked_at"`
	IPHash    string    `json:"ip_hash"` // hash de la IP, nunca la IP cruda
	UserAgent string    `json:"user_agent"`
	// Browser   string    `json:"browser"` // derivado del User-Agent
}
