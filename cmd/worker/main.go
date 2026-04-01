package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
	"github.com/ropehapi/kaizen-secretary/internal/kafka"
	"github.com/ropehapi/kaizen-secretary/internal/logger"
	"github.com/ropehapi/kaizen-secretary/internal/routines"
)

func main() {
	logger.Init()

	if err := godotenv.Load(); err != nil {
		slog.Warn("arquivo .env não encontrado, usando variáveis de ambiente do sistema", "error", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown via SIGINT / SIGTERM
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		slog.Info("sinal recebido, encerrando...", "signal", sig)
		cancel()
	}()

	brokers := strings.Split(os.Getenv("KAFKA_BROKERS"), ",")
	topic := os.Getenv("KAFKA_TOPIC")

	producer := kafka.NewProducer(brokers, topic)
	defer producer.Close()

	deliverer := kafka.NewMessagingOfficerDeliverer()
	consumer := kafka.NewConsumer(brokers, topic, "kaizen-secretary", deliverer, 2*time.Second)
	defer consumer.Close()

	go consumer.ReadLoop(ctx)

	loc, _ := time.LoadLocation("America/Sao_Paulo")
	c := cron.New(cron.WithSeconds(), cron.WithLocation(loc))

	_, err := c.AddFunc("0 00 16 7 3 *", func() {
		if err := routines.PublishScoutMonthlyFees(ctx, producer); err != nil {
			slog.Error("falha ao publicar eventos de mensalidade", "error", err)
		}
	})
	if err != nil {
		slog.Error("falha ao registrar cron job", "error", err)
		panic(err)
	}

	slog.Info("kaizen-secretary iniciado",
		"brokers", brokers,
		"topic", topic)

	c.Start()
	defer c.Stop()

	<-ctx.Done()
	slog.Info("kaizen-secretary encerrado")
}
