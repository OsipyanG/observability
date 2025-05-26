package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"producer-service/internal/config"
	"producer-service/internal/delivery/http/handlers"
	"producer-service/internal/delivery/http/middleware"
	"producer-service/internal/infrastructure/kafka"
	"producer-service/internal/infrastructure/metrics"
	"producer-service/internal/usecase"

	"github.com/gorilla/mux"
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
	}).Info("Starting producer service")

	// Инициализируем метрики
	producerMetrics := metrics.NewProducerMetrics()
	httpMetrics := metrics.NewHTTPMetrics()

	// Инициализируем Kafka producer
	kafkaProducer, err := kafka.NewProducer(cfg.Kafka, logger, producerMetrics)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create Kafka producer")
	}
	defer func() {
		if err := kafkaProducer.Close(); err != nil {
			logger.WithError(err).Error("Failed to close Kafka producer")
		}
	}()

	// Инициализируем сервисы
	eventService := usecase.NewEventService(kafkaProducer, logger)

	// Инициализируем handlers
	eventHandler := handlers.NewEventHandler(eventService, logger, httpMetrics)
	healthHandler := handlers.NewHealthHandler()

	// Настраиваем роутер
	router := mux.NewRouter()

	// Применяем middleware
	router.Use(middleware.LoggingMiddleware(logger))
	router.Use(middleware.RecoveryMiddleware(logger))
	router.Use(middleware.CORSMiddleware())

	// Регистрируем маршруты
	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/events/user", eventHandler.CreateUserEvent).Methods("POST")
	api.HandleFunc("/events/stats", eventHandler.GetEventStats).Methods("GET")

	// Системные маршруты
	router.HandleFunc("/health", healthHandler.Health).Methods("GET")
	router.HandleFunc("/ready", healthHandler.Ready).Methods("GET")

	// Запускаем метрики сервер если включен
	if cfg.Metrics.Enabled {
		go startMetricsServer(cfg.Metrics, logger)
	}

	// Настраиваем HTTP сервер
	srv := &http.Server{
		Addr:           cfg.Server.Address,
		Handler:        router,
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		IdleTimeout:    cfg.Server.IdleTimeout,
		MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
	}

	// Запускаем сервер в горутине
	go func() {
		logger.WithField("address", cfg.Server.Address).Info("HTTP server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("HTTP server failed to start")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Создаем контекст с таймаутом для graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	// Останавливаем HTTP сервер
	if err := srv.Shutdown(ctx); err != nil {
		logger.WithError(err).Error("Server forced to shutdown")
	}

	logger.Info("Server exited gracefully")
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
	mux := http.NewServeMux()
	mux.Handle(cfg.Path, promhttp.Handler())

	srv := &http.Server{
		Addr:    cfg.Port,
		Handler: mux,
	}

	logger.WithFields(logrus.Fields{
		"address": cfg.Port,
		"path":    cfg.Path,
	}).Info("Metrics server starting")

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.WithError(err).Error("Metrics server failed")
	}
}
