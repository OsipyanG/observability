package config

import (
	"os"
	"strconv"
	"time"
)

// Config содержит конфигурацию приложения
type Config struct {
	Server ServerConfig
	Kafka  KafkaConfig
}

// ServerConfig содержит конфигурацию HTTP сервера
type ServerConfig struct {
	Address      string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// KafkaConfig содержит конфигурацию Kafka
type KafkaConfig struct {
	Brokers []string
	Topic   string
}

// Load загружает конфигурацию из переменных окружения
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Address:      getEnv("SERVER_ADDRESS", ":8080"),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:  getDurationEnv("SERVER_IDLE_TIMEOUT", 60*time.Second),
		},
		Kafka: KafkaConfig{
			Brokers: []string{getEnv("KAFKA_BROKER_LIST", "localhost:9092")},
			Topic:   getEnv("KAFKA_TOPIC", "events"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
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
