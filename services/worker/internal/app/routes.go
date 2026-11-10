package app

import (
	"net/http"

	"shortener/internal/shared/httpserver"
	"shortener/services/worker/internal/handler"
)

// Version se puede sobrescribir al compilar:
// go build -ldflags "-X shortener/services/worker/internal/app.Version=1.0.0"
var Version = "0.1.0"

// Routes registra los endpoints del servicio worker.
func Routes(mux *http.ServeMux, clicks *handler.Clicks) {
	mux.HandleFunc("GET /version", httpserver.VersionHandler("worker", Version, "Worker que procesa los clics"))
	mux.HandleFunc("GET /clicks", clicks.ListClicks)

	mux.HandleFunc("GET /healthz", clicks.Healthz)
	mux.HandleFunc("GET /readyz", clicks.Readyz)
}
