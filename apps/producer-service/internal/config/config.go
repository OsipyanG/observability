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
	Server  ServerConfig  `validate:"required"`
	Kafka   KafkaConfig   `validate:"required"`
	Logging LoggingConfig `validate:"required"`
	Metrics MetricsConfig `validate:"required"`
	App     AppConfig     `validate:"required"`
}

// ServerConfig содержит конфигурацию HTTP сервера
type ServerConfig struct {
	Address         string        `validate:"required" env:"SERVER_ADDRESS" default:":8080"`
	ReadTimeout     time.Duration `validate:"min=1s,max=300s" env:"SERVER_READ_TIMEOUT" default:"15s"`
	WriteTimeout    time.Duration `validate:"min=1s,max=300s" env:"SERVER_WRITE_TIMEOUT" default:"15s"`
	IdleTimeout     time.Duration `validate:"min=1s,max=600s" env:"SERVER_IDLE_TIMEOUT" default:"60s"`
	ShutdownTimeout time.Duration `validate:"min=1s,max=60s" env:"SERVER_SHUTDOWN_TIMEOUT" default:"30s"`
	MaxHeaderBytes  int           `validate:"min=1024,max=1048576" env:"SERVER_MAX_HEADER_BYTES" default:"1048576"`
}

// KafkaConfig содержит конфигурацию Kafka
type KafkaConfig struct {
	Brokers         []string      `validate:"required,min=1" env:"KAFKA_BROKER_LIST" default:"localhost:9092"`
	Topic           string        `validate:"required,min=1" env:"KAFKA_TOPIC" default:"events"`
	ClientID        string        `validate:"required" env:"KAFKA_CLIENT_ID" default:"producer-service"`
	BatchSize       int           `validate:"min=1,max=1000" env:"KAFKA_BATCH_SIZE" default:"100"`
	BatchTimeout    time.Duration `validate:"min=1ms,max=10s" env:"KAFKA_BATCH_TIMEOUT" default:"10ms"`
	MaxRetries      int           `validate:"min=0,max=10" env:"KAFKA_MAX_RETRIES" default:"3"`
	RetryBackoff    time.Duration `validate:"min=1ms,max=30s" env:"KAFKA_RETRY_BACKOFF" default:"100ms"`
	CompressionType string        `validate:"oneof=none gzip snappy lz4 zstd" env:"KAFKA_COMPRESSION" default:"snappy"`
	RequiredAcks    int           `validate:"oneof=-1 0 1" env:"KAFKA_REQUIRED_ACKS" default:"1"`
}

// LoggingConfig содержит конфигурацию логирования
type LoggingConfig struct {
	Level  string `validate:"oneof=debug info warn error" env:"LOG_LEVEL" default:"info"`
	Format string `validate:"oneof=json text" env:"LOG_FORMAT" default:"json"`
}

// MetricsConfig содержит конфигурацию метрик
type MetricsConfig struct {
	Enabled bool   `env:"METRICS_ENABLED" default:"true"`
	Port    string `validate:"required" env:"METRICS_PORT" default:":9090"`
	Path    string `validate:"required" env:"METRICS_PATH" default:"/metrics"`
}

// AppConfig содержит общие настройки приложения
type AppConfig struct {
	Name        string `validate:"required" env:"APP_NAME" default:"producer-service"`
	Version     string `validate:"required" env:"APP_VERSION" default:"1.0.0"`
	Environment string `validate:"oneof=development staging production" env:"APP_ENV" default:"development"`
	Debug       bool   `env:"APP_DEBUG" default:"false"`
}

// Load загружает и валидирует конфигурацию из переменных окружения
func Load() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Address:         getEnv("SERVER_ADDRESS", ":8080"),
			ReadTimeout:     getDurationEnv("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    getDurationEnv("SERVER_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:     getDurationEnv("SERVER_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: getDurationEnv("SERVER_SHUTDOWN_TIMEOUT", 30*time.Second),
			MaxHeaderBytes:  getIntEnv("SERVER_MAX_HEADER_BYTES", 1048576),
		},
		Kafka: KafkaConfig{
			Brokers:         getBrokersEnv("KAFKA_BROKER_LIST", []string{"localhost:9092"}),
			Topic:           getEnv("KAFKA_TOPIC", "events"),
			ClientID:        getEnv("KAFKA_CLIENT_ID", "producer-service"),
			BatchSize:       getIntEnv("KAFKA_BATCH_SIZE", 100),
			BatchTimeout:    getDurationEnv("KAFKA_BATCH_TIMEOUT", 10*time.Millisecond),
			MaxRetries:      getIntEnv("KAFKA_MAX_RETRIES", 3),
			RetryBackoff:    getDurationEnv("KAFKA_RETRY_BACKOFF", 100*time.Millisecond),
			CompressionType: getEnv("KAFKA_COMPRESSION", "snappy"),
			RequiredAcks:    getIntEnv("KAFKA_REQUIRED_ACKS", 1),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		Metrics: MetricsConfig{
			Enabled: getBoolEnv("METRICS_ENABLED", true),
			Port:    getEnv("METRICS_PORT", ":9090"),
			Path:    getEnv("METRICS_PATH", "/metrics"),
		},
		App: AppConfig{
			Name:        getEnv("APP_NAME", "producer-service"),
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
