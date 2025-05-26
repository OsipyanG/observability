package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"consumer-service/internal/config"
	"consumer-service/internal/infrastructure/kafka"
	"consumer-service/internal/infrastructure/logging"
	"consumer-service/internal/infrastructure/metrics"
	"consumer-service/internal/usecase"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// Application основная структура приложения
type Application struct {
	config    *config.Config
	logger    *logrus.Logger
	metrics   *metrics.ConsumerMetrics
	consumer  *kafka.Consumer
	processor *usecase.EventProcessor

	// HTTP серверы
	metricsServer *http.Server
	healthServer  *http.Server

	// Управление жизненным циклом
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func main() {
	app, err := NewApplication()
	if err != nil {
		fmt.Printf("Failed to initialize application: %v\n", err)
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		app.logger.WithError(err).Error("Application failed")
		os.Exit(1)
	}
}

// NewApplication создает новое приложение
func NewApplication() (*Application, error) {
	// Загружаем конфигурацию
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Настраиваем логгер
	logger := setupLogger(cfg.Logging)
	logger.WithField("config", cfg).Info("Configuration loaded")

	// Создаем адаптер логгера
	loggerAdapter := logging.NewLogrusAdapter(logger)

	// Инициализируем метрики
	metricsInstance := metrics.NewConsumerMetrics(cfg.Metrics.Namespace, cfg.Metrics.Subsystem)
	logger.Info("Metrics initialized")

	// Создаем процессор событий
	eventProcessor := usecase.NewEventProcessor(
		loggerAdapter,
		metricsInstance,
		cfg.Consumer.MaxConcurrency,
		cfg.Consumer.BatchSize,
		cfg.Consumer.FlushInterval,
	)

	// Создаем Kafka consumer
	consumer := kafka.NewConsumer(
		cfg.Kafka,
		eventProcessor,
		loggerAdapter,
		metricsInstance,
	)

	// Создаем контекст
	ctx, cancel := context.WithCancel(context.Background())

	app := &Application{
		config:    cfg,
		logger:    logger,
		metrics:   metricsInstance,
		consumer:  consumer,
		processor: eventProcessor,
		ctx:       ctx,
		cancel:    cancel,
	}

	// Настраиваем HTTP серверы
	app.setupServers()

	return app, nil
}

// setupLogger настраивает логгер
func setupLogger(cfg config.LoggingConfig) *logrus.Logger {
	logger := logrus.New()

	// Устанавливаем уровень логирования
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Устанавливаем формат
	switch cfg.Format {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	case "text":
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
		})
	default:
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	}

	// Устанавливаем вывод
	switch cfg.Output {
	case "stdout":
		logger.SetOutput(os.Stdout)
	case "stderr":
		logger.SetOutput(os.Stderr)
	case "file":
		if cfg.Filename != "" {
			file, err := os.OpenFile(cfg.Filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err == nil {
				logger.SetOutput(file)
			}
		}
	default:
		logger.SetOutput(os.Stdout)
	}

	return logger
}

// setupServers настраивает HTTP серверы
func (app *Application) setupServers() {
	// Metrics server
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())

	app.metricsServer = &http.Server{
		Addr:         app.config.Metrics.Port,
		Handler:      metricsMux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Health server
	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/health", app.healthHandler)
	healthMux.HandleFunc("/ready", app.readinessHandler)
	healthMux.HandleFunc("/stats", app.statsHandler)

	app.healthServer = &http.Server{
		Addr:         app.config.Server.Address,
		Handler:      healthMux,
		ReadTimeout:  app.config.Server.ReadTimeout,
		WriteTimeout: app.config.Server.WriteTimeout,
		IdleTimeout:  app.config.Server.IdleTimeout,
	}
}

// Run запускает приложение
func (app *Application) Run() error {
	app.logger.Info("Starting consumer service",
		"version", app.config.App.Version,
		"environment", app.config.App.Environment)

	// Запускаем компоненты
	if err := app.startComponents(); err != nil {
		return fmt.Errorf("failed to start components: %w", err)
	}

	// Ожидаем сигнал завершения
	app.waitForShutdown()

	// Graceful shutdown
	return app.shutdown()
}

// startComponents запускает все компоненты приложения
func (app *Application) startComponents() error {
	// Запускаем процессор событий
	if err := app.processor.Start(app.ctx); err != nil {
		return fmt.Errorf("failed to start event processor: %w", err)
	}
	app.logger.Info("Event processor started")

	// Запускаем metrics server
	app.wg.Add(1)
	go func() {
		defer app.wg.Done()
		app.startMetricsServer()
	}()

	// Запускаем health server
	app.wg.Add(1)
	go func() {
		defer app.wg.Done()
		app.startHealthServer()
	}()

	// Запускаем consumer
	app.wg.Add(1)
	go func() {
		defer app.wg.Done()
		app.startConsumer()
	}()

	// Запускаем health checker
	if app.config.Health.Enabled {
		app.wg.Add(1)
		go func() {
			defer app.wg.Done()
			app.runHealthChecker()
		}()
	}

	return nil
}

// startMetricsServer запускает сервер метрик
func (app *Application) startMetricsServer() {
	app.logger.Info("Starting metrics server", "address", app.metricsServer.Addr)

	if err := app.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		app.logger.WithError(err).Error("Metrics server error")
	}
}

// startHealthServer запускает health сервер
func (app *Application) startHealthServer() {
	app.logger.Info("Starting health server", "address", app.healthServer.Addr)

	if err := app.healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		app.logger.WithError(err).Error("Health server error")
	}
}

// startConsumer запускает Kafka consumer
func (app *Application) startConsumer() {
	app.logger.Info("Starting Kafka consumer")

	if err := app.consumer.Consume(app.ctx); err != nil && err != context.Canceled {
		app.logger.WithError(err).Error("Consumer stopped with error")
	}
}

// runHealthChecker запускает периодические health checks
func (app *Application) runHealthChecker() {
	ticker := time.NewTicker(app.config.Health.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			app.performHealthCheck()
		case <-app.ctx.Done():
			return
		}
	}
}

// performHealthCheck выполняет проверку здоровья
func (app *Application) performHealthCheck() {
	// Проверяем статистику consumer
	stats := app.consumer.Stats()

	// Проверяем, что сообщения обрабатываются
	timeSinceLastMessage := time.Since(stats.LastMessageTime)
	if timeSinceLastMessage > 5*time.Minute {
		app.logger.Warn("No messages processed recently",
			"last_message_time", stats.LastMessageTime,
			"time_since", timeSinceLastMessage)
	}

	// Проверяем количество ошибок
	if stats.Errors > 0 {
		app.logger.Warn("Consumer has errors", "error_count", stats.Errors)
	}
}

// waitForShutdown ожидает сигнал завершения
func (app *Application) waitForShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	app.logger.Info("Consumer service started successfully")
	sig := <-sigChan
	app.logger.Info("Shutdown signal received", "signal", sig)
}

// shutdown выполняет graceful shutdown
func (app *Application) shutdown() error {
	app.logger.Info("Starting graceful shutdown")

	// Отменяем контекст
	app.cancel()

	// Останавливаем процессор
	shutdownCtx, cancel := context.WithTimeout(context.Background(), app.config.Consumer.ShutdownTimeout)
	defer cancel()

	if err := app.processor.Stop(shutdownCtx); err != nil {
		app.logger.WithError(err).Error("Failed to stop event processor")
	}

	// Закрываем consumer
	if err := app.consumer.Close(); err != nil {
		app.logger.WithError(err).Error("Failed to close consumer")
	}

	// Останавливаем HTTP серверы
	app.shutdownServers()

	// Ждем завершения всех горутин
	done := make(chan struct{})
	go func() {
		app.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		app.logger.Info("All goroutines finished")
	case <-time.After(app.config.Consumer.DrainTimeout):
		app.logger.Warn("Timeout waiting for goroutines to finish")
	}

	app.logger.Info("Consumer service stopped")
	return nil
}

// shutdownServers останавливает HTTP серверы
func (app *Application) shutdownServers() {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Останавливаем metrics server
	if err := app.metricsServer.Shutdown(shutdownCtx); err != nil {
		app.logger.WithError(err).Error("Failed to shutdown metrics server")
	} else {
		app.logger.Info("Metrics server stopped")
	}

	// Останавливаем health server
	if err := app.healthServer.Shutdown(shutdownCtx); err != nil {
		app.logger.WithError(err).Error("Failed to shutdown health server")
	} else {
		app.logger.Info("Health server stopped")
	}
}

// HTTP handlers

// healthHandler обрабатывает health check запросы
func (app *Application) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"healthy","timestamp":"%s","service":"%s","version":"%s"}`,
		time.Now().UTC().Format(time.RFC3339),
		app.config.App.Name,
		app.config.App.Version)
}

// readinessHandler обрабатывает readiness check запросы
func (app *Application) readinessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Проверяем готовность компонентов
	ready := true
	details := make(map[string]interface{})

	// Проверяем consumer
	stats := app.consumer.Stats()
	details["consumer"] = map[string]interface{}{
		"messages_consumed": stats.MessagesConsumed,
		"errors":            stats.Errors,
		"last_message":      stats.LastMessageTime,
	}

	// Проверяем processor
	processorStats := app.processor.GetStats()
	details["processor"] = map[string]interface{}{
		"events_processed": processorStats.EventsProcessed,
		"events_failed":    processorStats.EventsFailed,
		"processing_rate":  processorStats.ProcessingRate,
	}

	status := "ready"
	statusCode := http.StatusOK
	if !ready {
		status = "not_ready"
		statusCode = http.StatusServiceUnavailable
	}

	w.WriteHeader(statusCode)
	fmt.Fprintf(w, `{"status":"%s","timestamp":"%s","details":%v}`,
		status,
		time.Now().UTC().Format(time.RFC3339),
		details)
}

// statsHandler возвращает статистику сервиса
func (app *Application) statsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	consumerStats := app.consumer.Stats()
	processorStats := app.processor.GetStats()

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"consumer":%v,"processor":%v,"timestamp":"%s"}`,
		consumerStats,
		processorStats,
		time.Now().UTC().Format(time.RFC3339))
}
