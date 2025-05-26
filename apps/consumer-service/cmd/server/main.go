package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"consumer-service/internal/config"
	"consumer-service/internal/infrastructure/kafka"
	"consumer-service/internal/infrastructure/metrics"
	"consumer-service/internal/usecase"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

func main() {
	// Инициализируем логгер
	logger := setupLogger()

	// Загружаем конфигурацию
	cfg, err := config.Load()
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	logger.WithFields(logrus.Fields{
		"app_name":    cfg.App.Name,
		"version":     cfg.App.Version,
		"environment": cfg.App.Environment,
	}).Info("Starting consumer service")

	// Инициализируем метрики
	consumerMetrics := metrics.NewConsumerMetrics()

	// Инициализируем обработчик событий
	eventProcessor := usecase.NewEventProcessor(logger)

	// Инициализируем Kafka consumer
	kafkaConsumer, err := kafka.NewConsumer(cfg.Kafka, eventProcessor, logger, consumerMetrics)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create Kafka consumer")
	}
	defer func() {
		if err := kafkaConsumer.Close(); err != nil {
			logger.WithError(err).Error("Failed to close Kafka consumer")
		}
	}()

	// Запускаем метрики сервер если включен
	if cfg.Metrics.Enabled {
		go startMetricsServer(cfg.Metrics, logger)
	}

	// Создаем контекст для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запускаем consumer в горутине
	go func() {
		logger.Info("Starting Kafka consumer")
		if err := kafkaConsumer.Start(ctx); err != nil {
			if err != context.Canceled {
				logger.WithError(err).Error("Kafka consumer failed")
			}
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down consumer service...")

	// Отменяем контекст для остановки consumer
	cancel()

	// Даем время на graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Ждем завершения с таймаутом
	done := make(chan struct{})
	go func() {
		defer close(done)
		// Consumer уже получит сигнал через отмененный контекст
	}()

	select {
	case <-done:
		logger.Info("Consumer service exited gracefully")
	case <-shutdownCtx.Done():
		logger.Warn("Consumer service shutdown timeout exceeded")
	}
}

// setupLogger настраивает логгер
func setupLogger() *logrus.Logger {
	logger := logrus.New()

	// Устанавливаем уровень логирования из переменной окружения
	level := os.Getenv("LOG_LEVEL")
	if level == "" {
		level = "info"
	}

	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)

	// Устанавливаем формат логирования
	format := os.Getenv("LOG_FORMAT")
	if format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
		})
	}

	return logger
}

// startMetricsServer запускает отдельный сервер для метрик
func startMetricsServer(cfg config.MetricsConfig, logger *logrus.Logger) {
	metricsPath := "/metrics"
	mux := http.NewServeMux()
	mux.Handle(metricsPath, promhttp.Handler())

	srv := &http.Server{
		Addr:    cfg.Port,
		Handler: mux,
	}

	logger.WithFields(logrus.Fields{
		"address": cfg.Port,
		"path":    metricsPath,
	}).Info("Metrics server starting")

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.WithError(err).Error("Metrics server failed")
	}
}
