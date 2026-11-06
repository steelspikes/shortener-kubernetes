package main

import (
	"context"
	"os"

	"shortener/internal/shared/config"
	"shortener/internal/shared/db"
	"shortener/internal/shared/httpserver"
	"shortener/internal/shared/jobs"
	"shortener/internal/shared/logging"
	"shortener/internal/shared/rabbitmq"
	"shortener/internal/shared/redis"
	"shortener/services/api/internal/app"
	"shortener/services/api/internal/handler"
	"shortener/services/api/internal/store"
)

func main() {
	log := logging.New("api")

	pool, err := db.Connect(context.Background())
	if err != nil {
		log.Error("no se pudo conectar a postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	rdb, err := redis.Connect(context.Background())
	if err != nil {
		log.Error("no se pudo conectar a redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()

	mq, err := rabbitmq.Connect(context.Background())
	if err != nil {
		log.Error("no se pudo conectar a rabbitmq", "error", err)
		os.Exit(1)
	}
	defer mq.Close()

	if err := mq.DeclareQueue(jobs.ClicksQueue); err != nil {
		log.Error("no se pudo declarar la cola", "queue", jobs.ClicksQueue, "error", err)
		os.Exit(1)
	}

	links := &handler.Links{Store: store.NewLinkStore(pool), Redis: rdb, Rabbit: mq, Log: log}

	srv := httpserver.New(config.String("HTTP_ADDR", ":8080"), log)
	app.Routes(srv.Mux, links)

	if err := srv.Run(); err != nil {
		log.Error("servidor terminó con error", "error", err)
		os.Exit(1)
	}
}
