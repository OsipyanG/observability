package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config содержит конфигурацию приложения
type Config struct {
	Kafka    KafkaConfig    `env-prefix:"KAFKA_"`
	Consumer ConsumerConfig `env-prefix:"CONSUMER_"`
	Logging  LoggingConfig  `env-prefix:"LOG_"`
	Metrics  MetricsConfig  `env-prefix:"METRICS_"`
	App      AppConfig      `env-prefix:"APP_"`
}

// KafkaConfig содержит конфигурацию Kafka consumer
type KafkaConfig struct {
	Brokers        []string      `env:"BROKER_LIST" env-default:"localhost:9092"`
	Topic          string        `env:"TOPIC" env-default:"events"`
	GroupID        string        `env:"GROUP_ID" env-default:"consumer-service"`
	ClientID       string        `env:"CLIENT_ID" env-default:"consumer-service"`
	MinBytes       int           `env:"MIN_BYTES" env-default:"10000"`
	MaxBytes       int           `env:"MAX_BYTES" env-default:"10000000"`
	MaxWait        time.Duration `env:"MAX_WAIT" env-default:"1s"`
	CommitInterval time.Duration `env:"COMMIT_INTERVAL" env-default:"1s"`
	StartOffset    string        `env:"START_OFFSET" env-default:"latest"`
	MaxRetries     int           `env:"MAX_RETRIES" env-default:"3"`
	RetryBackoff   time.Duration `env:"RETRY_BACKOFF" env-default:"100ms"`
}

// ConsumerConfig содержит конфигурацию обработки сообщений
type ConsumerConfig struct {
	WorkerCount int `env:"WORKER_COUNT" env-default:"10"`
	BatchSize   int `env:"BATCH_SIZE" env-default:"100"`
}

// LoggingConfig содержит конфигурацию логирования
type LoggingConfig struct {
	Level  string `env:"LEVEL" env-default:"info"`
	Format string `env:"FORMAT" env-default:"json"`
}

// MetricsConfig содержит конфигурацию метрик
type MetricsConfig struct {
	Enabled bool   `env:"ENABLED" env-default:"true"`
	Port    string `env:"PORT" env-default:":9090"`
}

// AppConfig содержит общие настройки приложения
type AppConfig struct {
	Name        string `env:"NAME" env-default:"consumer-service"`
	Version     string `env:"VERSION" env-default:"1.0.0"`
	Environment string `env:"ENV" env-default:"development"`
	Debug       bool   `env:"DEBUG" env-default:"false"`
}

// Load загружает и валидирует конфигурацию из переменных окружения
func Load() (*Config, error) {
	var cfg Config

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return &cfg, nil
}
