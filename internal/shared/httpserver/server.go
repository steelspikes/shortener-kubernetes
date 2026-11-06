package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

// Server envuelve http.Server con middlewares comunes y apagado ordenado.
type Server struct {
	Mux   *http.ServeMux
	log   *slog.Logger
	srv   *http.Server
	grace time.Duration
}

// New crea un servidor escuchando en addr. Registra rutas sobre s.Mux.
func New(addr string, log *slog.Logger) *Server {
	mux := http.NewServeMux()
	s := &Server{Mux: mux, log: log, grace: 25 * time.Second}
	s.srv = &http.Server{
		Addr:              addr,
		Handler:           Chain(mux, Recover(log), RequestID, Logging(log)),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return s
}

// Run arranca el servidor y bloquea hasta SIGINT/SIGTERM; luego hace Shutdown.
func (s *Server) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		s.log.Info("servidor escuchando", "addr", s.srv.Addr)
		if err := s.srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	s.log.Info("apagando servidor")
	shutCtx, cancel := context.WithTimeout(context.Background(), s.grace)
	defer cancel()
	return s.srv.Shutdown(shutCtx)
}
