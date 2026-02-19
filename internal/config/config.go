package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/QuantumSolver/crm-relay/internal/models"
)

// Load loads configuration from environment variables
func Load() (*models.Config, error) {
	cfg := &models.Config{
		ServerPort:         getEnv("SERVER_PORT", "8080"),
		RedisURL:          getEnv("REDIS_URL", "localhost:6379"),
		RedisPassword:     getEnv("REDIS_PASSWORD", ""),
		RedisDB:           getEnvAsInt("REDIS_DB", 0),
		StreamName:        getEnv("STREAM_NAME", "webhook-stream"),
		ConsumerGroup:     getEnv("CONSUMER_GROUP", "relay-group"),
		ConsumerName:      getEnv("CONSUMER_NAME", "relay-client"),
		DeadLetterQueue:   getEnv("DEAD_LETTER_QUEUE", "webhook-dlq"),
		MessageTTL:        getEnvAsInt("MESSAGE_TTL", 86400),
		APIKey:            getEnv("API_KEY", ""),
		LocalWebhookURL:   getEnv("LOCAL_WEBHOOK_URL", "http://localhost:3000/webhook"),
		MaxRetries:        getEnvAsInt("MAX_RETRIES", 3),
		RetryDelay:        getEnvAsInt("RETRY_DELAY", 1000),
		RetryMultiplier:   getEnvAsFloat("RETRY_MULTIPLIER", 2.0),
		HealthCheckInterval: getEnvAsInt("HEALTH_CHECK_INTERVAL", 30),
	}

	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt retrieves an environment variable as an integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// getEnvAsFloat retrieves an environment variable as a float or returns a default value
func getEnvAsFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}

// validate validates the configuration
func validate(cfg *models.Config) error {
	var errors []string

	if cfg.ServerPort == "" {
		errors = append(errors, "SERVER_PORT is required")
	}

	if cfg.RedisURL == "" {
		errors = append(errors, "REDIS_URL is required")
	}

	if cfg.StreamName == "" {
		errors = append(errors, "STREAM_NAME is required")
	}

	if cfg.ConsumerGroup == "" {
		errors = append(errors, "CONSUMER_GROUP is required")
	}

	if cfg.ConsumerName == "" {
		errors = append(errors, "CONSUMER_NAME is required")
	}

	if cfg.APIKey == "" {
		errors = append(errors, "API_KEY is required")
	}

	if cfg.LocalWebhookURL == "" {
		errors = append(errors, "LOCAL_WEBHOOK_URL is required")
	}

	if cfg.MaxRetries < 0 {
		errors = append(errors, "MAX_RETRIES must be non-negative")
	}

	if cfg.RetryDelay < 0 {
		errors = append(errors, "RETRY_DELAY must be non-negative")
	}

	if cfg.RetryMultiplier <= 0 {
		errors = append(errors, "RETRY_MULTIPLIER must be positive")
	}

	if cfg.MessageTTL <= 0 {
		errors = append(errors, "MESSAGE_TTL must be positive")
	}

	if len(errors) > 0 {
		return models.NewRelayError(
			models.ErrCodeInvalidConfig,
			"configuration validation failed",
			fmt.Errorf("%s", strings.Join(errors, "; ")),
		)
	}

	return nil
}
