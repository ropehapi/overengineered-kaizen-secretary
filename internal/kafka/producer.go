package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	segmentio "github.com/segmentio/kafka-go"
)

// EventPublisher publica eventos de mensagem WhatsApp em um broker.
// Satisfeita por *Producer e permite mock em testes.
type EventPublisher interface {
	Publish(ctx context.Context, event WhatsAppMessageEvent) error
}

// writer é uma interface interna que permite injetar um mock em testes no lugar do kafka.Writer.
type writer interface {
	WriteMessages(ctx context.Context, msgs ...segmentio.Message) error
	Close() error
}

// Producer publica eventos WhatsApp no Kafka.
type Producer struct {
	w writer
}

// NewProducer cria um Producer conectado aos brokers informados, escrevendo no topic dado.
func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{
		w: &segmentio.Writer{
			Addr:                   segmentio.TCP(brokers...),
			Topic:                  topic,
			Balancer:               &segmentio.LeastBytes{},
			AllowAutoTopicCreation: true,
		},
	}
}

// Publish serializa o evento em JSON e publica no Kafka.
// A Key da mensagem é o RecipientPhone, garantindo ordering por destinatário.
func (p *Producer) Publish(ctx context.Context, event WhatsAppMessageEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return p.w.WriteMessages(ctx, segmentio.Message{
		Key:   []byte(event.RecipientPhone),
		Value: payload,
	})
}

// Close encerra o writer do Kafka.
func (p *Producer) Close() error {
	return p.w.Close()
}
