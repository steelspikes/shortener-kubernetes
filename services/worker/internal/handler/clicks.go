package handler

import (
	"log/slog"
	"net/http"

	"shortener/internal/shared/httpserver"
	"shortener/services/worker/internal/store"
)

type Clicks struct {
	Store *store.ClickStore
	Log   *slog.Logger
}

// ListClicks maneja GET /clicks.
func (h *Clicks) ListClicks(w http.ResponseWriter, r *http.Request) {
	clicks, err := h.Store.ListClicks(r.Context())

	if err != nil {
		httpserver.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpserver.JSON(w, http.StatusOK, clicks)
}
