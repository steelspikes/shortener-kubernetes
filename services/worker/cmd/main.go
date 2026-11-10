package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"shortener/internal/shared/config"
	"shortener/internal/shared/db"
	"shortener/internal/shared/httpserver"
	"shortener/internal/shared/jobs"
	"shortener/internal/shared/logging"
	"shortener/internal/shared/rabbitmq"
	"shortener/services/worker/internal/app"
	"shortener/services/worker/internal/consumer"
	"shortener/services/worker/internal/handler"
	"shortener/services/worker/internal/store"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log := logging.New("worker")

	pool, err := db.ConnectClicks(context.Background())
	if err != nil {
		log.Error("no se pudo conectar a postgres (clicks)", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

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

	srv := httpserver.New(config.String("HTTP_ADDR", ":8082"), log)

	clickStore := store.NewClickStore(pool)

	clicks := &handler.Clicks{Store: clickStore, Log: log, Rabbit: mq}
	app.Routes(srv.Mux, clicks)

	consumer := consumer.Consumer{Rabbit: mq, Log: log, Store: clickStore}
	go consumer.Execute(ctx)

	if err := srv.Run(); err != nil {
		log.Error("servidor terminó con error", "error", err)
		os.Exit(1)
	}
}
