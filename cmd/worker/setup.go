package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/ropehapi/kaizen-secretary/internal/kafka"
	"github.com/ropehapi/kaizen-secretary/internal/telemetry"
	"go.uber.org/zap"
)

func setupMetricsServer(port string) {
	if port == "" {
		port = "9090"
	}
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(":"+port, mux); err != nil {
			zap.L().Error("metrics server failed", zap.Error(err))
			os.Exit(1)
		}
	}()
}

func mustInitTelemetry(ctx context.Context) func(context.Context) error {
	shutdown, err := telemetry.Init(ctx)
	if err != nil {
		zap.L().Error("falha ao inicializar telemetria", zap.Error(err))
		os.Exit(1)
	}
	return shutdown
}

func setupKafka(brokers []string, topic string) (*kafka.Producer, *kafka.Consumer) {
	producer := kafka.NewProducer(brokers, topic)
	deliverer := kafka.NewMessagingOfficerDeliverer()
	consumer := kafka.NewConsumer(brokers, topic, "kaizen-secretary", deliverer, 2*time.Second)
	return producer, consumer
}
