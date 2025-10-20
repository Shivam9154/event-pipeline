package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	Kafka   KafkaConfig
	MSSQL   MSSQLConfig
	Redis   RedisConfig
	API     APIConfig
	Metrics MetricsConfig
}

// KafkaConfig holds Kafka configuration
type KafkaConfig struct {
	Brokers       string
	Topic         string
	ConsumerGroup string
}

// MSSQLConfig holds MS SQL configuration
type MSSQLConfig struct {
	Server   string
	Port     int
	User     string
	Password string
	Database string
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
	DLQKey   string
}

// APIConfig holds API server configuration
type APIConfig struct {
	Port string
}

// MetricsConfig holds metrics server configuration
type MetricsConfig struct {
	Port string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Try to load .env file (optional)
	_ = godotenv.Load()

	redisPort, err := strconv.Atoi(getEnv("REDIS_PORT", "6379"))
	if err != nil {
		return nil, fmt.Errorf("invalid REDIS_PORT: %w", err)
	}

	redisDB, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil {
		return nil, fmt.Errorf("invalid REDIS_DB: %w", err)
	}

	mssqlPort, err := strconv.Atoi(getEnv("MSSQL_PORT", "1433"))
	if err != nil {
		return nil, fmt.Errorf("invalid MSSQL_PORT: %w", err)
	}

	return &Config{
		Kafka: KafkaConfig{
			Brokers:       getEnv("KAFKA_BROKERS", "localhost:9092"),
			Topic:         getEnv("KAFKA_TOPIC", "events"),
			ConsumerGroup: getEnv("KAFKA_CONSUMER_GROUP", "event-consumer-group"),
		},
		MSSQL: MSSQLConfig{
			Server:   getEnv("MSSQL_SERVER", "localhost"),
			Port:     mssqlPort,
			User:     getEnv("MSSQL_USER", "sa"),
			Password: getEnv("MSSQL_PASSWORD", ""),
			Database: getEnv("MSSQL_DATABASE", "eventdb"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     redisPort,
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       redisDB,
			DLQKey:   getEnv("REDIS_DLQ_KEY", "dlq:events"),
		},
		API: APIConfig{
			Port: getEnv("API_PORT", "8080"),
		},
		Metrics: MetricsConfig{
			Port: getEnv("METRICS_PORT", "9090"),
		},
	}, nil
}

// GetConnectionString returns MS SQL connection string
func (c *MSSQLConfig) GetConnectionString() string {
	return fmt.Sprintf("server=%s;port=%d;user id=%s;password=%s;database=%s;encrypt=disable",
		c.Server, c.Port, c.User, c.Password, c.Database)
}

// GetRedisAddr returns Redis address
func (c *RedisConfig) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
