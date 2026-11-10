package rabbitmq

import (
	"context"
	"fmt"
	"net/url"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"shortener/internal/shared/config"
)

// Delivery es un mensaje recibido de una cola. Llama a Ack() cuando lo hayas
// procesado, o Nack(requeue) si falló.
type Delivery = amqp.Delivery

// Client mantiene una conexión y un canal AMQP. Al estar embebido el canal,
// puedes usar directamente cualquier método de amqp091 si necesitas algo más.
// No es seguro para uso concurrente en publicaciones: usa un Client por goroutine
// o protege con un mutex.
type Client struct {
	conn *amqp.Connection
	*amqp.Channel
}

// URL construye la URL AMQP desde variables de entorno.
func URL() string {
	RABBIT_PORT := config.String("RABBITMQ_PORT", "5672")

	u := url.URL{
		Scheme: "amqp",
		User: url.UserPassword(
			config.String("RABBITMQ_USER", "guest"),
			config.String("RABBITMQ_PASSWORD", "guest"),
		),
		Host: config.String("RABBITMQ_ADDR", "localhost") + ":" + RABBIT_PORT,
		Path: "/",
	}
	return u.String()
}

// Connect abre la conexión y un canal usando variables de entorno.
func Connect(ctx context.Context) (*Client, error) {
	conn, err := amqp.Dial(URL())
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}
	return &Client{conn: conn, Channel: ch}, nil
}

// Close cierra el canal y la conexión.
func (c *Client) Close() error {
	c.Channel.Close()
	return c.conn.Close()
}

// DeclareQueue crea la cola si no existe (durable, así sobrevive a reinicios).
// Es idempotente: se puede llamar desde productor y consumidor.
func (c *Client) DeclareQueue(name string) error {
	_, err := c.QueueDeclare(name, true, false, false, false, nil)
	return err
}

// Publish agrega un mensaje a la cola (persistente).
func (c *Client) Publish(ctx context.Context, queue string, body []byte) error {
	return c.PublishWithContext(ctx, "", queue, false, false, amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/octet-stream",
		Body:         body,
	})
}

// Consume devuelve un canal con los mensajes de la cola. Hay que confirmar cada
// uno con Ack()/Nack(). Prefetch limita los mensajes sin confirmar en vuelo.
// El canal se cierra cuando se cierra el cliente.
func (c *Client) Consume(queue string, prefetch int) (<-chan Delivery, error) {
	if err := c.Qos(prefetch, 0, false); err != nil {
		return nil, err
	}
	return c.Channel.Consume(queue, "", false, false, false, false, nil)
}

// Get saca un solo mensaje de la cola sin bloquear. ok=false si la cola está vacía.
// Debes llamar a msg.Ack(false) tras procesarlo.
func (c *Client) Get(queue string) (msg Delivery, ok bool, err error) {
	return c.Channel.Get(queue, false)
}

// String implementa fmt.Stringer sin exponer credenciales.
func (c *Client) String() string {
	return fmt.Sprintf("rabbitmq(%s)", config.String("RABBITMQ_ADDR", "localhost:5672"))
}

func (c *Client) Check(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	done := make(chan error, 1) // 1. canal de resultado

	go func() { // 2. trabajo en segundo plano
		if c.conn.IsClosed() {
			done <- amqp.ErrClosed
			return
		}
		ch, err := c.conn.Channel()
		if err == nil {
			err = ch.Close()
		}
		done <- err
	}()

	select { // 3. espera: lo primero que ocurra
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
