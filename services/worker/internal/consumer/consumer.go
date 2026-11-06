package consumer

import (
	"context"
	"fmt"
	"log/slog"
	"shortener/internal/shared/jobs"
	"shortener/internal/shared/rabbitmq"
	"shortener/services/worker/internal/store"
)

type Consumer struct {
	Rabbit *rabbitmq.Client
	Log    *slog.Logger
	Store  *store.ClickStore
}

func (c *Consumer) Process(ctx context.Context, v jobs.ClickJob) error {
	fmt.Println("Content", v)
	return c.Store.SaveClick(ctx, v)
}

func (c *Consumer) Execute(ctx context.Context) {
	msgs, err := c.Rabbit.Consume(jobs.ClicksQueue, 10)

	if err != nil {
		c.Log.Error("Error initializing consumer", "Error", err)
	}

	for msg := range msgs {
		fmt.Println("Sacando bb")

		v, err := rabbitmq.DecodeJSON[jobs.ClickJob](msg)

		if err != nil {
			msg.Nack(false, false) // mensaje malformado: descartarlo
			continue
		}

		if err := c.Process(ctx, v); err != nil {
			msg.Nack(false, true) // falló: devolverlo a la cola
			continue
		}

		msg.Ack(false)
	}
}
