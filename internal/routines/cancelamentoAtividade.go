package routines

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/ropehapi/kaizen-secretary/internal/kafka"
	"github.com/ropehapi/kaizen-secretary/internal/metrics"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

type CancelledActivity struct {
	Name   string
	Date   string
	Reason string
}

// PublishActivityCancellationNotice publica um evento Kafka por membro para cada
// atividade cancelada, notificando-os via WhatsApp com o motivo do cancelamento.
func PublishActivityCancellationNotice(ctx context.Context, publisher Publisher) error {
	const routineName = "PublishActivityCancellationNotice"

	metrics.RoutineExecutionsTotal.WithLabelValues(routineName).Inc()
	timer := prometheus.NewTimer(metrics.RoutineDurationSeconds.WithLabelValues(routineName))
	defer timer.ObserveDuration()

	activities := cancelledActivities()
	members := activityMembers()

	zap.L().Info("publicando avisos de cancelamento de atividade",
		zap.Int("activities", len(activities)),
		zap.Int("members", len(members)))

	tracer := otel.Tracer("kaizen-secretary")
	var errs int
	for _, activity := range activities {
		for name, phone := range members {
			_, span := tracer.Start(ctx, "publish_whatsapp_event")
			span.SetAttributes(
				attribute.String("recipient.phone", phone),
				attribute.String("recipient.name", name),
				attribute.String("activity.name", activity.Name),
				attribute.String("activity.date", activity.Date),
			)

			event := kafka.WhatsAppMessageEvent{
				RecipientPhone: phone,
				RecipientName:  name,
				Message:        BuildCancellationMessage(name, activity),
			}
			if err := publisher.Publish(ctx, event); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				metrics.MessagesSentTotal.WithLabelValues(routineName, "failure").Inc()
				zap.L().Error("falha ao publicar evento kafka",
					zap.String("recipient_name", name),
					zap.String("recipient", phone),
					zap.String("activity", activity.Name),
					zap.Error(err))
				errs++
			} else {
				metrics.MessagesSentTotal.WithLabelValues(routineName, "success").Inc()
			}
			span.End()
		}
	}

	if errs > 0 {
		metrics.RoutineErrorsTotal.WithLabelValues(routineName).Add(float64(errs))
	}

	zap.L().Info("avisos de cancelamento publicados no kafka",
		zap.Int("sent", len(activities)*len(members)-errs),
		zap.Int("failed", errs))

	return nil
}

// BuildCancellationMessage constrói o texto do aviso de cancelamento de atividade.
func BuildCancellationMessage(name string, activity CancelledActivity) string {
	return "Olá, " + name + "! Informamos que a atividade *" + activity.Name + "*" +
		" prevista para " + activity.Date + " foi cancelada.\n" +
		"Motivo: " + activity.Reason + "\n" +
		"Obs: Essa é uma mensagem automática. Em caso de dúvidas, entre em contato com a liderança."
}

// cancelledActivities retorna a lista de atividades canceladas.
func cancelledActivities() []CancelledActivity {
	return []CancelledActivity{
		{
			Name:   "Acampamento de Páscoa",
			Date:   "19/04/2025",
			Reason: "Previsão de chuvas fortes para o final de semana",
		},
	}
}

// activityMembers retorna o mapa de nome → telefone dos membros a notificar sobre atividades.
func activityMembers() map[string]string {
	return map[string]string{
		"Pedrinho": "5543936180709",
	}
}
