package rabbitmq

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

// PublishJSON serializa v a JSON y lo agrega a la cola (persistente).
func (c *Client) PublishJSON(ctx context.Context, queue string, v any) error {
	body, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.PublishWithContext(ctx, "", queue, false, false, amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		Body:         body,
	})
}

// DecodeJSON deserializa el body de un mensaje al tipo T.
// Si falla, el mensaje está malformado: descártalo con msg.Nack(false, false).
//
//	job, err := rabbitmq.DecodeJSON[Job](msg)
func DecodeJSON[T any](msg Delivery) (T, error) {
	var v T
	err := json.Unmarshal(msg.Body, &v)
	return v, err
}
