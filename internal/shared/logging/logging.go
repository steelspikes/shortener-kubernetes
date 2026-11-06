package logging

import (
	"log/slog"
	"os"
)

// New crea un logger JSON estructurado con el campo "service".
func New(service string) *slog.Logger {
	h := slog.NewJSONHandler(os.Stdout, nil)
	return slog.New(h).With("service", service)
}
