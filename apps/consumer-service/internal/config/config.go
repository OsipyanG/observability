package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

// Config содержит конфигурацию приложения
type Config struct {
	Kafka   KafkaConfig   `validate:"required"`
	Logging LoggingConfig `validate:"required"`
	Metrics MetricsConfig `validate:"required"`
	App     AppConfig     `validate:"required"`
}

// KafkaConfig содержит конфигурацию Kafka consumer
type KafkaConfig struct {
	Brokers        []string      `validate:"required,min=1" env:"KAFKA_BROKER_LIST" default:"localhost:9092"`
	Topic          string        `validate:"required,min=1" env:"KAFKA_TOPIC" default:"events"`
	GroupID        string        `validate:"required" env:"KAFKA_GROUP_ID" default:"consumer-service"`
	ClientID       string        `validate:"required" env:"KAFKA_CLIENT_ID" default:"consumer-service"`
	MinBytes       int           `validate:"min=1" env:"KAFKA_MIN_BYTES" default:"10e3"`
	MaxBytes       int           `validate:"min=1" env:"KAFKA_MAX_BYTES" default:"10e6"`
	MaxWait        time.Duration `validate:"min=1ms" env:"KAFKA_MAX_WAIT" default:"1s"`
	CommitInterval time.Duration `validate:"min=1ms" env:"KAFKA_COMMIT_INTERVAL" default:"1s"`
	StartOffset    string        `validate:"oneof=earliest latest" env:"KAFKA_START_OFFSET" default:"latest"`
	MaxRetries     int           `validate:"min=0,max=10" env:"KAFKA_MAX_RETRIES" default:"3"`
	RetryBackoff   time.Duration `validate:"min=1ms,max=30s" env:"KAFKA_RETRY_BACKOFF" default:"100ms"`
}

// LoggingConfig содержит конфигурацию логирования
type LoggingConfig struct {
	Level  string `validate:"oneof=debug info warn error" env:"LOG_LEVEL" default:"info"`
	Format string `validate:"oneof=json text" env:"LOG_FORMAT" default:"json"`
}

// MetricsConfig содержит конфигурацию метрик
type MetricsConfig struct {
	Enabled bool   `env:"METRICS_ENABLED" default:"true"`
	Port    string `validate:"required" env:"METRICS_PORT" default:":9091"`
	Path    string `validate:"required" env:"METRICS_PATH" default:"/metrics"`
}

// AppConfig содержит общие настройки приложения
type AppConfig struct {
	Name        string `validate:"required" env:"APP_NAME" default:"consumer-service"`
	Version     string `validate:"required" env:"APP_VERSION" default:"1.0.0"`
	Environment string `validate:"oneof=development staging production" env:"APP_ENV" default:"development"`
	Debug       bool   `env:"APP_DEBUG" default:"false"`
}

// Load загружает и валидирует конфигурацию из переменных окружения
func Load() (*Config, error) {
	config := &Config{
		Kafka: KafkaConfig{
			Brokers:        getBrokersEnv("KAFKA_BROKER_LIST", []string{"localhost:9092"}),
			Topic:          getEnv("KAFKA_TOPIC", "events"),
			GroupID:        getEnv("KAFKA_GROUP_ID", "consumer-service"),
			ClientID:       getEnv("KAFKA_CLIENT_ID", "consumer-service"),
			MinBytes:       getIntEnv("KAFKA_MIN_BYTES", 10e3),
			MaxBytes:       getIntEnv("KAFKA_MAX_BYTES", 10e6),
			MaxWait:        getDurationEnv("KAFKA_MAX_WAIT", 1*time.Second),
			CommitInterval: getDurationEnv("KAFKA_COMMIT_INTERVAL", 1*time.Second),
			StartOffset:    getEnv("KAFKA_START_OFFSET", "latest"),
			MaxRetries:     getIntEnv("KAFKA_MAX_RETRIES", 3),
			RetryBackoff:   getDurationEnv("KAFKA_RETRY_BACKOFF", 100*time.Millisecond),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		Metrics: MetricsConfig{
			Enabled: getBoolEnv("METRICS_ENABLED", true),
			Port:    getEnv("METRICS_PORT", ":9091"),
			Path:    getEnv("METRICS_PATH", "/metrics"),
		},
		App: AppConfig{
			Name:        getEnv("APP_NAME", "consumer-service"),
			Version:     getEnv("APP_VERSION", "1.0.0"),
			Environment: getEnv("APP_ENV", "development"),
			Debug:       getBoolEnv("APP_DEBUG", false),
		},
	}

	// Валидируем конфигурацию
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// Validate валидирует конфигурацию
func (c *Config) Validate() error {
	validate := validator.New()
	if err := validate.Struct(c); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	return nil
}

// IsProduction проверяет, запущено ли приложение в production
func (c *Config) IsProduction() bool {
	return c.App.Environment == "production"
}

// IsDevelopment проверяет, запущено ли приложение в development
func (c *Config) IsDevelopment() bool {
	return c.App.Environment == "development"
}

// GetKafkaBrokerAddresses возвращает адреса брокеров Kafka
func (c *Config) GetKafkaBrokerAddresses() []string {
	return c.Kafka.Brokers
}

// Вспомогательные функции для получения переменных окружения

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
		if seconds, err := strconv.Atoi(value); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}
	return defaultValue
}

func getBrokersEnv(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		brokers := strings.Split(value, ",")
		// Очищаем пробелы
		for i, broker := range brokers {
			brokers[i] = strings.TrimSpace(broker)
		}
		return brokers
	}
	return defaultValue
}
