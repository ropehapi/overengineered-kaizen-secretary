package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"net/http"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/robfig/cron/v3"
	"github.com/ropehapi/kaizen-secretary/internal/kafka"
	"github.com/ropehapi/kaizen-secretary/internal/logger"
	"github.com/ropehapi/kaizen-secretary/internal/metrics"
	"github.com/ropehapi/kaizen-secretary/internal/routines"
	"github.com/ropehapi/kaizen-secretary/internal/telemetry"
	"go.opentelemetry.io/otel"
)

func main() {
	logger.Init()
	metrics.Init()

	if err := godotenv.Load(); err != nil {
		slog.Warn("arquivo .env não encontrado, usando variáveis de ambiente do sistema", "error", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	metricsPort := os.Getenv("METRICS_PORT")
	if metricsPort == "" {
		metricsPort = "9090"
	}
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(":"+metricsPort, mux); err != nil {
			slog.Error("metrics server failed", "error", err)
			os.Exit(1)
		}
	}()

	shutdown, err := telemetry.Init(ctx)
	if err != nil {
		slog.Error("falha ao inicializar telemetria", "error", err)
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := shutdown(shutdownCtx); err != nil {
			slog.Error("falha ao encerrar telemetria", "error", err)
		}
	}()

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

	_, err = c.AddFunc("*/30 * * * * *", func() {
		spanCtx, span := otel.Tracer("kaizen-secretary").Start(ctx, "PublishScoutMonthlyFees")
		defer span.End()
		if err := routines.PublishScoutMonthlyFees(spanCtx, producer); err != nil {
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
