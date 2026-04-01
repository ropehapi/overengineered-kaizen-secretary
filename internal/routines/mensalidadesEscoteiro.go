package routines

import (
	"context"
	"log/slog"
	"time"

	"github.com/ropehapi/kaizen-secretary/internal/kafka"
)

// Publisher publica eventos de mensagem WhatsApp em um broker.
// Satisfeita por *kafka.Producer e permite mock em testes.
type Publisher interface {
	Publish(ctx context.Context, event kafka.WhatsAppMessageEvent) error
}

// PublishScoutMonthlyFees publica um evento Kafka por contribuinte do grupo escoteiro.
// A entrega efetiva ao WhatsApp é feita pelo Consumer do Kafka de forma assíncrona.
func PublishScoutMonthlyFees(ctx context.Context, publisher Publisher) error {
	month := monthInPortuguese(time.Now())
	members := contributors()

	slog.Info("publicando lembretes de mensalidade escoteiro",
		"month", month,
		"total", len(members))

	var errs int
	for name, phone := range members {
		event := kafka.WhatsAppMessageEvent{
			RecipientPhone: phone,
			RecipientName:  name,
			Message:        BuildMessage(name, month),
		}
		if err := publisher.Publish(ctx, event); err != nil {
			slog.Error("falha ao publicar evento kafka",
				"name", name,
				"phone", phone,
				"error", err)
			errs++
			continue
		}
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
