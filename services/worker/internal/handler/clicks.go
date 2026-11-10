package handler

import (
	"log/slog"
	"net/http"

	"shortener/internal/shared/httpserver"
	"shortener/internal/shared/rabbitmq"
	"shortener/services/worker/internal/store"
)

type Clicks struct {
	Store  *store.ClickStore
	Log    *slog.Logger
	Rabbit *rabbitmq.Client
}

type ServicesStatus struct {
	Database bool `json:"database"`
	Rabbit   bool `json:"rabbit"`
}

type ReadyResponse struct {
	Status string          `json:"status"`
	Checks *ServicesStatus `json:"checks,omitempty"`
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

func (h *Clicks) Healthz(w http.ResponseWriter, r *http.Request) {
	response := ReadyResponse{Status: "ok"}
	httpserver.JSON(w, http.StatusOK, response)
}

func (h *Clicks) Readyz(w http.ResponseWriter, r *http.Request) {
	statuses := ServicesStatus{Database: true, Rabbit: true}
	available := true

	if h.Store.Check(r.Context()) != nil {
		statuses.Database = false
		available = false
	}

	if h.Rabbit.Check(r.Context()) != nil {
		statuses.Rabbit = false
		available = false
	}

	status, responseCode := "unavailable", http.StatusServiceUnavailable

	if available {
		status, responseCode = "ready", http.StatusOK
	}

	response := ReadyResponse{Status: status, Checks: &statuses}

	httpserver.JSON(w, responseCode, response)
}
