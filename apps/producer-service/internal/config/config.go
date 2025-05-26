package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config содержит конфигурацию приложения
type Config struct {
	Server  ServerConfig
	Kafka   KafkaConfig
	Logging LoggingConfig
	Metrics MetricsConfig
	App     AppConfig
}

// ServerConfig содержит конфигурацию HTTP сервера
type ServerConfig struct {
	Address         string        `env:"SERVER_ADDRESS" env-default:":8080"`
	ReadTimeout     time.Duration `env:"SERVER_READ_TIMEOUT" env-default:"15s"`
	WriteTimeout    time.Duration `env:"SERVER_WRITE_TIMEOUT" env-default:"15s"`
	IdleTimeout     time.Duration `env:"SERVER_IDLE_TIMEOUT" env-default:"60s"`
	ShutdownTimeout time.Duration `env:"SERVER_SHUTDOWN_TIMEOUT" env-default:"30s"`
	MaxHeaderBytes  int           `env:"SERVER_MAX_HEADER_BYTES" env-default:"1048576"`
}

// KafkaConfig содержит конфигурацию Kafka
type KafkaConfig struct {
	Brokers         []string      `env:"KAFKA_BROKER_LIST" env-default:"localhost:9092"`
	Topic           string        `env:"KAFKA_TOPIC" env-default:"events"`
	ClientID        string        `env:"KAFKA_CLIENT_ID" env-default:"producer-service"`
	BatchSize       int           `env:"KAFKA_BATCH_SIZE" env-default:"100"`
	BatchTimeout    time.Duration `env:"KAFKA_BATCH_TIMEOUT" env-default:"10ms"`
	MaxRetries      int           `env:"KAFKA_MAX_RETRIES" env-default:"3"`
	RetryBackoff    time.Duration `env:"KAFKA_RETRY_BACKOFF" env-default:"100ms"`
	CompressionType string        `env:"KAFKA_COMPRESSION" env-default:"snappy"`
	RequiredAcks    int           `env:"KAFKA_REQUIRED_ACKS" env-default:"1"`
}

// LoggingConfig содержит конфигурацию логирования
type LoggingConfig struct {
	Level  string `env:"LOG_LEVEL" env-default:"info"`
	Format string `env:"LOG_FORMAT" env-default:"json"`
}

// MetricsConfig содержит конфигурацию метрик
type MetricsConfig struct {
	Enabled bool   `env:"METRICS_ENABLED" env-default:"true"`
	Port    string `env:"METRICS_PORT" env-default:":9090"`
	Path    string `env:"METRICS_PATH" env-default:"/metrics"`
}

// AppConfig содержит общие настройки приложения
type AppConfig struct {
	Name        string `env:"APP_NAME" env-default:"producer-service"`
	Version     string `env:"APP_VERSION" env-default:"1.0.0"`
	Environment string `env:"APP_ENV" env-default:"development"`
	Debug       bool   `env:"APP_DEBUG" env-default:"false"`
}

// Load загружает конфигурацию из переменных окружения
func Load() (*Config, error) {
	var config Config

	if err := cleanenv.ReadEnv(&config); err != nil {
		return nil, fmt.Errorf("failed to read environment: %w", err)
	}

	return &config, nil
}
