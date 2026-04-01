package routines

import (
	"context"
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/ropehapi/kaizen-secretary/internal/kafka"
	"github.com/ropehapi/kaizen-secretary/internal/metrics"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// Publisher publica eventos de mensagem WhatsApp em um broker.
// Satisfeita por *kafka.Producer e permite mock em testes.
type Publisher interface {
	Publish(ctx context.Context, event kafka.WhatsAppMessageEvent) error
}

// PublishScoutMonthlyFees publica um evento Kafka por contribuinte do grupo escoteiro.
// A entrega efetiva ao WhatsApp é feita pelo Consumer do Kafka de forma assíncrona.
func PublishScoutMonthlyFees(ctx context.Context, publisher Publisher) error {
	const routineName = "PublishScoutMonthlyFees"

	metrics.RoutineExecutionsTotal.WithLabelValues(routineName).Inc()
	timer := prometheus.NewTimer(metrics.RoutineDurationSeconds.WithLabelValues(routineName))
	defer timer.ObserveDuration()

	month := monthInPortuguese(time.Now())
	members := contributors()

	slog.Info("publicando lembretes de mensalidade escoteiro",
		"month", month,
		"total", len(members))

	tracer := otel.Tracer("kaizen-secretary")
	var errs int
	for name, phone := range members {
		_, span := tracer.Start(ctx, "publish_whatsapp_event")
		span.SetAttributes(
			attribute.String("recipient.phone", phone),
			attribute.String("recipient.name", name),
		)

		event := kafka.WhatsAppMessageEvent{
			RecipientPhone: phone,
			RecipientName:  name,
			Message:        BuildMessage(name, month),
		}
		if err := publisher.Publish(ctx, event); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			metrics.MessagesSentTotal.WithLabelValues(routineName, "failure").Inc()
			slog.Error("falha ao publicar evento kafka",
				"name", name,
				"phone", phone,
				"error", err)
			errs++
		} else {
			metrics.MessagesSentTotal.WithLabelValues(routineName, "success").Inc()
		}
		span.End()
	}

	if errs > 0 {
		metrics.RoutineErrorsTotal.WithLabelValues(routineName).Add(float64(errs))
	}

	slog.Info("eventos publicados no kafka",
		"total", len(members),
		"publish_errors", errs)

	return nil
}

// BuildMessage constrói o texto da mensagem de lembrete de mensalidade.
// Exportada para permitir reuso e testes unitários.
func BuildMessage(name, month string) string {
	return "Olá, " + name + ", passando para lembrar sobre Contribuição mensal do Grupo Escoteiro Guarani, " +
		"referente ao mês de " + month + ". Enviar comprovante no whatsApp *PIX GRUPO GUARANI*.\n" +
		"Obs: Essa é uma mensagem automática. Caso já tenha feito o pagamento, por favor desconsidere."
}

// contributors retorna o mapa de nome → telefone dos contribuintes do grupo escoteiro.
func contributors() map[string]string {
	return map[string]string{
		"Pedrinho": "5543936180709",
	}
}

// monthInPortuguese retorna o nome do mês em português para o instante t.
func monthInPortuguese(t time.Time) string {
	names := [...]string{
		"Janeiro", "Fevereiro", "Março", "Abril",
		"Maio", "Junho", "Julho", "Agosto",
		"Setembro", "Outubro", "Novembro", "Dezembro",
	}
	return names[t.Month()-1]
}
