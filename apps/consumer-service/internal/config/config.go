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
	App      AppConfig      `validate:"required"`
	Server   ServerConfig   `validate:"required"`
	Kafka    KafkaConfig    `validate:"required"`
	Consumer ConsumerConfig `validate:"required"`
	Metrics  MetricsConfig  `validate:"required"`
	Logging  LoggingConfig  `validate:"required"`
	Health   HealthConfig   `validate:"required"`
}

// AppConfig общие настройки приложения
type AppConfig struct {
	Name        string `validate:"required,min=1"`
	Version     string `validate:"required,min=1"`
	Environment string `validate:"required,oneof=development staging production"`
	Debug       bool
}

// ServerConfig настройки HTTP сервера
type ServerConfig struct {
	Address      string        `validate:"required"`
	ReadTimeout  time.Duration `validate:"min=1s"`
	WriteTimeout time.Duration `validate:"min=1s"`
	IdleTimeout  time.Duration `validate:"min=1s"`
}

// KafkaConfig содержит конфигурацию Kafka
type KafkaConfig struct {
	Brokers        []string      `validate:"required,min=1"`
	Topic          string        `validate:"required,min=1"`
	GroupID        string        `validate:"required,min=1"`
	MinBytes       int           `validate:"min=1"`
	MaxBytes       int           `validate:"min=1024"`
	MaxWait        time.Duration `validate:"min=100ms"`
	StartOffset    int64         `validate:"oneof=-2 -1"`
	CommitInterval time.Duration `validate:"min=100ms"`

	// Настройки безопасности
	SecurityProtocol string
	SASLMechanism    string
	SASLUsername     string
	SASLPassword     string

	// Настройки производительности
	FetchMin     int           `validate:"min=1"`
	FetchMax     int           `validate:"min=1024"`
	FetchDefault int           `validate:"min=1024"`
	MaxWaitTime  time.Duration `validate:"min=100ms"`

	// Настройки retry
	RetryBackoff time.Duration `validate:"min=100ms"`
	MaxRetries   int           `validate:"min=0"`
}

// ConsumerConfig содержит конфигурацию consumer
type ConsumerConfig struct {
	WorkerCount     int           `validate:"min=1,max=100"`
	BatchSize       int           `validate:"min=1,max=10000"`
	ProcessTimeout  time.Duration `validate:"min=1s"`
	RetryAttempts   int           `validate:"min=0,max=10"`
	RetryDelay      time.Duration `validate:"min=100ms"`
	RetryBackoffMax time.Duration `validate:"min=1s"`

	// Настройки обработки
	MaxConcurrency int           `validate:"min=1,max=1000"`
	BufferSize     int           `validate:"min=1"`
	FlushInterval  time.Duration `validate:"min=100ms"`

	// Настройки graceful shutdown
	ShutdownTimeout time.Duration `validate:"min=1s"`
	DrainTimeout    time.Duration `validate:"min=1s"`
}

// MetricsConfig содержит конфигурацию метрик
type MetricsConfig struct {
	Enabled   bool   `validate:"required"`
	Port      string `validate:"required"`
	Path      string `validate:"required"`
	Namespace string `validate:"required,min=1"`
	Subsystem string `validate:"required,min=1"`
}

// LoggingConfig настройки логирования
type LoggingConfig struct {
	Level      string `validate:"required,oneof=debug info warn error"`
	Format     string `validate:"required,oneof=json text"`
	Output     string `validate:"required,oneof=stdout stderr file"`
	Filename   string
	MaxSize    int `validate:"min=1"`
	MaxBackups int `validate:"min=0"`
	MaxAge     int `validate:"min=1"`
	Compress   bool
}

// HealthConfig настройки health checks
type HealthConfig struct {
	Enabled          bool          `validate:"required"`
	CheckInterval    time.Duration `validate:"min=1s"`
	Timeout          time.Duration `validate:"min=1s"`
	FailureThreshold int           `validate:"min=1"`
}

// Load загружает конфигурацию из переменных окружения
func Load() (*Config, error) {
	config := &Config{
		App: AppConfig{
			Name:        getEnv("APP_NAME", "consumer-service"),
			Version:     getEnv("APP_VERSION", "1.0.0"),
			Environment: getEnv("APP_ENV", "development"),
			Debug:       getBoolEnv("APP_DEBUG", false),
		},
		Server: ServerConfig{
			Address:      getEnv("SERVER_ADDRESS", ":8080"),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:  getDurationEnv("SERVER_IDLE_TIMEOUT", 60*time.Second),
		},
		Kafka: KafkaConfig{
			Brokers:          getBrokersEnv("KAFKA_BROKER_LIST", []string{"localhost:9092"}),
			Topic:            getEnv("KAFKA_TOPIC", "events"),
			GroupID:          getEnv("KAFKA_GROUP_ID", "consumer-service"),
			MinBytes:         getIntEnv("KAFKA_MIN_BYTES", 1),
			MaxBytes:         getIntEnv("KAFKA_MAX_BYTES", 10485760), // 10MB
			MaxWait:          getDurationEnv("KAFKA_MAX_WAIT", 1*time.Second),
			StartOffset:      getInt64Env("KAFKA_START_OFFSET", -1), // latest
			CommitInterval:   getDurationEnv("KAFKA_COMMIT_INTERVAL", 1*time.Second),
			SecurityProtocol: getEnv("KAFKA_SECURITY_PROTOCOL", "PLAINTEXT"),
			SASLMechanism:    getEnv("KAFKA_SASL_MECHANISM", ""),
			SASLUsername:     getEnv("KAFKA_SASL_USERNAME", ""),
			SASLPassword:     getEnv("KAFKA_SASL_PASSWORD", ""),
			FetchMin:         getIntEnv("KAFKA_FETCH_MIN", 1),
			FetchMax:         getIntEnv("KAFKA_FETCH_MAX", 1048576),     // 1MB
			FetchDefault:     getIntEnv("KAFKA_FETCH_DEFAULT", 1048576), // 1MB
			MaxWaitTime:      getDurationEnv("KAFKA_MAX_WAIT_TIME", 500*time.Millisecond),
			RetryBackoff:     getDurationEnv("KAFKA_RETRY_BACKOFF", 100*time.Millisecond),
			MaxRetries:       getIntEnv("KAFKA_MAX_RETRIES", 3),
		},
		Consumer: ConsumerConfig{
			WorkerCount:     getIntEnv("CONSUMER_WORKER_COUNT", 5),
			BatchSize:       getIntEnv("CONSUMER_BATCH_SIZE", 100),
			ProcessTimeout:  getDurationEnv("CONSUMER_PROCESS_TIMEOUT", 30*time.Second),
			RetryAttempts:   getIntEnv("CONSUMER_RETRY_ATTEMPTS", 3),
			RetryDelay:      getDurationEnv("CONSUMER_RETRY_DELAY", 1*time.Second),
			RetryBackoffMax: getDurationEnv("CONSUMER_RETRY_BACKOFF_MAX", 30*time.Second),
			MaxConcurrency:  getIntEnv("CONSUMER_MAX_CONCURRENCY", 10),
			BufferSize:      getIntEnv("CONSUMER_BUFFER_SIZE", 1000),
			FlushInterval:   getDurationEnv("CONSUMER_FLUSH_INTERVAL", 5*time.Second),
			ShutdownTimeout: getDurationEnv("CONSUMER_SHUTDOWN_TIMEOUT", 30*time.Second),
			DrainTimeout:    getDurationEnv("CONSUMER_DRAIN_TIMEOUT", 10*time.Second),
		},
		Metrics: MetricsConfig{
			Enabled:   getBoolEnv("METRICS_ENABLED", true),
			Port:      getEnv("METRICS_PORT", ":9090"),
			Path:      getEnv("METRICS_PATH", "/metrics"),
			Namespace: getEnv("METRICS_NAMESPACE", "consumer"),
			Subsystem: getEnv("METRICS_SUBSYSTEM", "service"),
		},
		Logging: LoggingConfig{
			Level:      getEnv("LOG_LEVEL", "info"),
			Format:     getEnv("LOG_FORMAT", "json"),
			Output:     getEnv("LOG_OUTPUT", "stdout"),
			Filename:   getEnv("LOG_FILENAME", "consumer-service.log"),
			MaxSize:    getIntEnv("LOG_MAX_SIZE", 100), // MB
			MaxBackups: getIntEnv("LOG_MAX_BACKUPS", 3),
			MaxAge:     getIntEnv("LOG_MAX_AGE", 28), // days
			Compress:   getBoolEnv("LOG_COMPRESS", true),
		},
		Health: HealthConfig{
			Enabled:          getBoolEnv("HEALTH_ENABLED", true),
			CheckInterval:    getDurationEnv("HEALTH_CHECK_INTERVAL", 30*time.Second),
			Timeout:          getDurationEnv("HEALTH_TIMEOUT", 5*time.Second),
			FailureThreshold: getIntEnv("HEALTH_FAILURE_THRESHOLD", 3),
		},
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

// Validate проверяет валидность конфигурации
func (c *Config) Validate() error {
	validate := validator.New()
	if err := validate.Struct(c); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Дополнительные проверки
	if c.Consumer.WorkerCount > c.Consumer.MaxConcurrency {
		return fmt.Errorf("worker count (%d) cannot exceed max concurrency (%d)",
			c.Consumer.WorkerCount, c.Consumer.MaxConcurrency)
	}

	if c.Kafka.MaxBytes < c.Kafka.MinBytes {
		return fmt.Errorf("kafka max bytes (%d) cannot be less than min bytes (%d)",
			c.Kafka.MaxBytes, c.Kafka.MinBytes)
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

// GetKafkaBrokerString возвращает строку с брокерами Kafka
func (c *Config) GetKafkaBrokerString() string {
	return strings.Join(c.Kafka.Brokers, ",")
}

// Helper functions
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

func getInt64Env(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
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
		// Fallback: try parsing as seconds
		if seconds, err := strconv.Atoi(value); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}
	return defaultValue
}

func getBrokersEnv(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		brokers := strings.Split(value, ",")
		// Trim whitespace from each broker
		for i, broker := range brokers {
			brokers[i] = strings.TrimSpace(broker)
		}
		return brokers
	}
	return defaultValue
}
