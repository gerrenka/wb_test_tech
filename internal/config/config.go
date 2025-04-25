package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	// PostgreSQL
	DBHost      string
	DBPort      int
	DBUser      string
	DBPassword  string
	DBName      string
	DBSSLMode   string
	DBMaxConns  int
	DBIdleConns int

	// Kafka
	KafkaBrokers []string
	KafkaTopic   string
	KafkaGroupID string

	// HTTP Server
	ServerPort int

	// Cache
	CacheTTL time.Duration
}

func NewConfig() (*Config, error) {
	config := &Config{
		// PostgreSQL defaults
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBUser:      getEnv("DB_USER", "my_user"),
		DBPassword:  getEnv("DB_PASSWORD", "1"),
		DBName:      getEnv("DB_NAME", "my_database"),
		DBSSLMode:   getEnv("DB_SSL_MODE", "disable"),
		DBMaxConns:  getEnvAsInt("DB_MAX_CONNS", 25),
		DBIdleConns: getEnvAsInt("DB_IDLE_CONNS", 5),

		// Kafka defaults
		KafkaTopic:   getEnv("KAFKA_TOPIC", "orders"),
		KafkaGroupID: getEnv("KAFKA_GROUP_ID", "order-consumer-group"),

		// HTTP Server defaults
		ServerPort: getEnvAsInt("SERVER_PORT", 8081),

		// Cache defaults
		CacheTTL: getEnvAsDuration("CACHE_TTL", 30*time.Minute),
	}

	// Get DB port
	var err error
	config.DBPort, err = strconv.Atoi(getEnv("DB_PORT", "5434"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_PORT: %v", err)
	}

	// Kafka brokers
	brokersStr := getEnv("KAFKA_BROKERS", "localhost:9092")
	config.KafkaBrokers = strings.Split(brokersStr, ",")

	return config, nil
}

func (c *Config) GetDBConnString() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
