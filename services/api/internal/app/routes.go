package app

import (
	"net/http"

	"shortener/internal/shared/httpserver"
	"shortener/services/api/internal/handler"
)

// Version se puede sobrescribir al compilar:
// go build -ldflags "-X shortener/services/api/internal/app.Version=1.0.0"
var Version = "0.1.0"

// Routes registra los endpoints del servicio api.
func Routes(mux *http.ServeMux, links *handler.Links) {
	mux.HandleFunc("GET /version", httpserver.VersionHandler("api", Version, "API del acortador de URLs"))
	mux.HandleFunc("POST /link", links.CreateLink)
	mux.HandleFunc("GET /links", links.ListLinks)
	mux.HandleFunc("GET /link/{code}", links.ListLink)
}
