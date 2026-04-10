package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
	"github.com/ropehapi/kaizen-secretary/internal/featureflags"
	"github.com/ropehapi/kaizen-secretary/internal/kafka"
	"github.com/ropehapi/kaizen-secretary/internal/logger"
	"github.com/ropehapi/kaizen-secretary/internal/metrics"
	"github.com/ropehapi/kaizen-secretary/internal/routines"
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

	if err := featureflags.Init(); err != nil {
		zap.L().Warn("flagsmith unavailable, all flags default to false", zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupMetricsServer(os.Getenv("METRICS_PORT"))

	shutdown := mustInitTelemetry(ctx)
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := shutdown(shutdownCtx); err != nil {
			zap.L().Error("falha ao encerrar telemetria", zap.Error(err))
		}
	}()

	brokers := strings.Split(os.Getenv("KAFKA_BROKERS"), ",")
	topic := os.Getenv("KAFKA_TOPIC")

	producer, consumer := setupKafka(brokers, topic)
	defer producer.Close()
	defer consumer.Close()

	go consumer.ReadLoop(ctx)

	loc, _ := time.LoadLocation("America/Sao_Paulo")
	c := cron.New(cron.WithSeconds(), cron.WithLocation(loc))

	_, err = c.AddFunc("*/30 * * * * *", func() {
		if !featureflags.IsEnabled("scout_monthly_reminder_enabled") {
			zap.L().Info("routine disabled by feature flag",
				zap.String("routine", "PublishScoutMonthlyFees"))
			return
		}
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

	_, err = c.AddFunc("*/30 * * * * *", func() {
		spanCtx, span := otel.Tracer("kaizen-secretary").Start(ctx, "PublishActivityCancellationNotice")
		defer span.End()
		if err := routines.PublishActivityCancellationNotice(spanCtx, producer); err != nil {
			zap.L().Error("falha ao publicar avisos de cancelamento de atividade", zap.Error(err))
		}
	})
	if err != nil {
		zap.L().Error("falha ao registrar cron job", zap.Error(err))
		panic(err)
	}

	c.Start()
	defer c.Stop()

	go waitSignal(cancel)

	zap.L().Info("kaizen-secretary iniciado",
		zap.Strings("brokers", brokers),
		zap.String("topic", topic))

	<-ctx.Done()
	zap.L().Info("kaizen-secretary encerrado")
}

func waitSignal(cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	zap.L().Info("sinal recebido, encerrando...", zap.String("signal", sig.String()))
	cancel()
}
