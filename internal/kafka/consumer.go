package kafka

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	segmentio "github.com/segmentio/kafka-go"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.uber.org/zap"
)

// Deliverer entrega um evento de mensagem WhatsApp ao serviço de envio.
// Satisfeita por *MessagingOfficerDeliverer e permite mock em testes.
type Deliverer interface {
	Deliver(ctx context.Context, event WhatsAppMessageEvent) error
}

// messageReader é uma interface interna que permite injetar um mock do kafka.Reader em testes.
type messageReader interface {
	ReadMessage(ctx context.Context) (segmentio.Message, error)
	Close() error
}

// Consumer lê eventos do topic Kafka e os entrega via Deliverer.
type Consumer struct {
	reader        messageReader
	deliverer     Deliverer
	deliveryDelay time.Duration
}

// NewConsumer cria um Consumer inscrito no topic dado.
// deliveryDelay adiciona uma pausa entre entregas — use 0 em testes.
func NewConsumer(brokers []string, topic, groupID string, deliverer Deliverer, deliveryDelay time.Duration) *Consumer {
	return &Consumer{
		reader: segmentio.NewReader(segmentio.ReaderConfig{
			Brokers: brokers,
			Topic:   topic,
			GroupID: groupID,
		}),
		deliverer:     deliverer,
		deliveryDelay: deliveryDelay,
	}
}

// ReadLoop bloqueia lendo mensagens do Kafka até ctx ser cancelado.
// Erros de unmarshal e de entrega são logados mas não interrompem o loop.
func (c *Consumer) ReadLoop(ctx context.Context) {
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return // shutdown gracioso
			}
			zap.L().Error("kafka read error", zap.Error(err))
			continue
		}

		var event WhatsAppMessageEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			zap.L().Error("failed to unmarshal kafka message",
				zap.Error(err),
				zap.Int64("offset", msg.Offset))
			continue
		}

		if err := c.deliverer.Deliver(ctx, event); err != nil {
			zap.L().Error("failed to deliver message",
				zap.Error(err),
				zap.String("recipient", event.RecipientPhone),
				zap.Int64("offset", msg.Offset))
			continue
		}

		zap.L().Info("message delivered via kafka consumer",
			zap.String("recipient", event.RecipientPhone),
			zap.Int64("offset", msg.Offset))

		if c.deliveryDelay > 0 {
			time.Sleep(c.deliveryDelay)
		}
	}
}

// Close encerra o reader do Kafka.
func (c *Consumer) Close() error {
	return c.reader.Close()
}

// MessagingOfficerDeliverer entrega eventos WhatsApp via HTTP para o serviço messaging-officer.
type MessagingOfficerDeliverer struct {
	client  *http.Client
	baseURL string
	apiKey  string
	session string
}

// NewMessagingOfficerDeliverer cria um Deliverer lendo configuração das variáveis de ambiente.
func NewMessagingOfficerDeliverer() *MessagingOfficerDeliverer {
	return newMessagingOfficerDeliverer(
		os.Getenv("MESSAGING_OFFICER_HOST")+":"+os.Getenv("MESSAGING_OFFICER_PORT"),
		os.Getenv("MESSAGING_OFFICER_API_KEY"),
		os.Getenv("MESSAGING_OFFICER_SESSION_ID"),
	)
}

func newMessagingOfficerDeliverer(baseURL, apiKey, session string) *MessagingOfficerDeliverer {
	return &MessagingOfficerDeliverer{
		client: &http.Client{
			Timeout:   10 * time.Second,
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
		baseURL: baseURL,
		apiKey:  apiKey,
		session: session,
	}
}

// Deliver faz POST para o messaging-officer com o payload do evento.
func (d *MessagingOfficerDeliverer) Deliver(ctx context.Context, event WhatsAppMessageEvent) error {
	payload, err := json.Marshal(map[string]interface{}{
		"number":  event.RecipientPhone,
		"message": event.Message,
	})
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		d.baseURL+"/api/send-message", bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", d.apiKey)
	req.Header.Set("x-Session-Id", d.session)

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("messaging officer returned status %d", resp.StatusCode)
	}
	return nil
}
