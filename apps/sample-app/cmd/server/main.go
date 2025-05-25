package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sample-app/internal/config"
	"sample-app/internal/delivery/http/handlers"
	"sample-app/internal/delivery/http/middleware"
	"sample-app/internal/infrastructure/kafka"
	"sample-app/internal/infrastructure/metrics"
	"sample-app/internal/usecase"

	"github.com/gorilla/mux"
)

func main() {
	// Загружаем конфигурацию
	cfg := config.Load()

	// Инициализируем метрики
	metricsCollector := metrics.NewPrometheusCollector()

	// Инициализируем Kafka producer
	kafkaProducer, err := kafka.NewProducer(cfg.Kafka)
	if err != nil {
		log.Fatalf("Failed to create Kafka producer: %v", err)
	}
	defer kafkaProducer.Close()

	// Инициализируем use cases
	eventUseCase := usecase.NewEventUseCase(kafkaProducer, metricsCollector)

	// Инициализируем handlers
	eventHandler := handlers.NewEventHandler(eventUseCase)
	healthHandler := handlers.NewHealthHandler()

	// Настраиваем роутер
	router := mux.NewRouter()

	// Применяем middleware
	router.Use(middleware.PrometheusMiddleware(metricsCollector))
	router.Use(middleware.LoggingMiddleware())
	router.Use(middleware.RecoveryMiddleware())

	// Регистрируем маршруты
	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/events/user-created", eventHandler.CreateUserEvent).Methods("POST")
	api.HandleFunc("/events/order-placed", eventHandler.CreateOrderEvent).Methods("POST")
	api.HandleFunc("/events/payment-processed", eventHandler.CreatePaymentEvent).Methods("POST")

	router.HandleFunc("/health", healthHandler.Health).Methods("GET")
	router.Handle("/metrics", metricsCollector.Handler())

	// Настраиваем HTTP сервер
	srv := &http.Server{
		Addr:         cfg.Server.Address,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Запускаем сервер в горутине
	go func() {
		log.Printf("Server starting on %s", cfg.Server.Address)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Server shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
