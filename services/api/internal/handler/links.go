package handler

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"shortener/internal/shared/httpserver"
	"shortener/internal/shared/jobs"
	"shortener/internal/shared/rabbitmq"
	"shortener/internal/shared/redis"
	"shortener/services/api/internal/store"

	"github.com/mileusna/useragent"
)

type Links struct {
	Store  *store.LinkStore
	Redis  *redis.Client
	Rabbit *rabbitmq.Client
	Log    *slog.Logger
}

// CreateLinkRequest es el body de POST /links.
type CreateLinkRequest struct {
	URL       string     `json:"url"`
	Code      string     `json:"code"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type ServicesStatus struct {
	Database bool `json:"database"`
	Rabbit   bool `json:"rabbit"`
	Redis    bool `json:"redis"`
}

type ReadyResponse struct {
	Status string          `json:"status"`
	Checks *ServicesStatus `json:"checks,omitempty"`
}

// CreateLink maneja POST /links.
func (h *Links) CreateLink(w http.ResponseWriter, r *http.Request) {
	body, err := httpserver.DecodeBody[CreateLinkRequest](w, r)
	if err != nil {
		httpserver.Error(w, http.StatusBadRequest, "json inválido")
		return
	}

	if len(body.URL) == 0 || len(body.Code) == 0 {
		httpserver.Error(w, http.StatusBadRequest, "Some fields are required")
		return
	}

	created, err := h.Store.Create(r.Context(), store.NewLink{
		URL:       body.URL,
		Code:      body.Code,
		ExpiresAt: body.ExpiresAt,
	})

	if err != nil {
		httpserver.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpserver.JSON(w, http.StatusOK, created)

}

// ListLinks maneja GET /links.
func (h *Links) ListLinks(w http.ResponseWriter, r *http.Request) {
	links, err := h.Store.ListAll(r.Context())
	if err != nil {
		httpserver.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpserver.JSON(w, http.StatusOK, links)
}

func (h *Links) EnqueueClick(ctx context.Context, code string, remoteIP string, userAgent string, referer string, browser string) {
	hashBytesIP := sha256.Sum256([]byte(remoteIP))
	hashHexIP := fmt.Sprintf("%x", hashBytesIP)

	err := h.Rabbit.PublishJSON(ctx, jobs.ClicksQueue, jobs.ClickJob{Code: code, ClickedAt: time.Now(), IPHash: hashHexIP, UserAgent: userAgent, Referer: referer, Browser: browser})

	if err != nil {
		h.Log.Error("Could not publish Click", "Error", err)
	}
}

func (h *Links) ListLink(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	redisQ := fmt.Sprintf("link:%s", code)

	userAgent := r.Header.Get("User-Agent")
	referer := r.Header.Get("Referer")
	browser := useragent.Parse(userAgent).Name

	catchedUrl, err := h.Redis.Get(r.Context(), redisQ).Result()
	if !redis.IsMiss(err) {
		h.EnqueueClick(r.Context(), code, r.RemoteAddr, userAgent, referer, browser)
		http.Redirect(w, r, catchedUrl, http.StatusFound)
		return
	}

	links, err := h.Store.List(r.Context(), code)

	if err != nil {
		httpserver.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.EnqueueClick(r.Context(), code, r.RemoteAddr, userAgent, referer, browser)

	h.Redis.Set(r.Context(), redisQ, links.URL, redis.DefaultTTL())

	http.Redirect(w, r, links.URL, http.StatusFound)
}

func (h *Links) Healthz(w http.ResponseWriter, r *http.Request) {
	response := ReadyResponse{Status: "ok"}
	httpserver.JSON(w, http.StatusOK, response)
}

func (h *Links) Readyz(w http.ResponseWriter, r *http.Request) {
	statuses := ServicesStatus{Database: true, Rabbit: true}
	available := true

	if h.Store.Check(r.Context()) != nil {
		statuses.Database = false
		available = false
	}

	if h.Redis.Check(r.Context()) != nil {
		statuses.Redis = false
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
