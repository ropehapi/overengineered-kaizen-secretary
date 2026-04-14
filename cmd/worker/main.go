package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"net/http"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/robfig/cron/v3"
	"github.com/ropehapi/kaizen-secretary/internal/flagsmith"
	"github.com/ropehapi/kaizen-secretary/internal/kafka"
	"github.com/ropehapi/kaizen-secretary/internal/logger"
	"github.com/ropehapi/kaizen-secretary/internal/metrics"
	"github.com/ropehapi/kaizen-secretary/internal/routines"
	"github.com/ropehapi/kaizen-secretary/internal/telemetry"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

func main() {
	logger.Init()
	defer func() { _ = zap.L().Sync() }()
	metrics.Init()

	if err := godotenv.Load(); err != nil {
		zap.L().Warn("arquivo .env não encontrado, usando variáveis de ambiente do sistema", zap.Error(err))
	}

	fs := flagsmith.NewClient(os.Getenv("FLAGSMITH_API_KEY"))

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
			zap.L().Error("metrics server failed", zap.Error(err))
			os.Exit(1)
		}
	}()

	shutdown, err := telemetry.Init(ctx)
	if err != nil {
		zap.L().Error("falha ao inicializar telemetria", zap.Error(err))
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := shutdown(shutdownCtx); err != nil {
			zap.L().Error("falha ao encerrar telemetria", zap.Error(err))
		}
	}()

	// Graceful shutdown via SIGINT / SIGTERM
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		zap.L().Info("sinal recebido, encerrando...", zap.String("signal", sig.String()))
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
		zap.L().Info("Flagsmith: "+ strconv.FormatBool(fs.IsEnable("routine_mensalidade_escoteiro")))
		spanCtx, span := otel.Tracer("kaizen-secretary").Start(ctx, "PublishScoutMonthlyFees")
		defer span.End()
		if err := routines.PublishScoutMonthlyFees(spanCtx, producer); err != nil {
			zap.L().Error("falha ao publicar eventos de mensalidade", zap.Error(err))
		}
	})
	if err != nil {
		zap.L().Error("falha ao registrar cron job", zap.Error(err))
		panic(err)
	}

	zap.L().Info("kaizen-secretary iniciado",
		zap.Strings("brokers", brokers),
		zap.String("topic", topic))

	c.Start()
	defer c.Stop()

	<-ctx.Done()
	zap.L().Info("kaizen-secretary encerrado")
}
