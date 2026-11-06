package main

import (
	"os"

	"shortener/internal/shared/config"
	"shortener/internal/shared/httpserver"
	"shortener/internal/shared/logging"
	"shortener/services/stats/internal/app"
)

func main() {
	log := logging.New("stats")

	srv := httpserver.New(config.String("HTTP_ADDR", ":8081"), log)
	app.Routes(srv.Mux)

	if err := srv.Run(); err != nil {
		log.Error("servidor terminó con error", "error", err)
		os.Exit(1)
	}
}
