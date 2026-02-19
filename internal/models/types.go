package models

import (
	"time"
)

// Webhook represents an incoming webhook from Meta platform
type Webhook struct {
	ID        string            `json:"id"`
	Headers   map[string]string `json:"headers"`
	Body      []byte            `json:"body"`
	Timestamp time.Time         `json:"timestamp"`
	Signature string            `json:"signature,omitempty"`
}

// RelayMessage represents a message in the Redis stream
type RelayMessage struct {
	MessageID  string    `json:"message_id"`
	Webhook    Webhook   `json:"webhook"`
	RetryCount int       `json:"retry_count"`
	CreatedAt  time.Time `json:"created_at"`
}

// Config holds the configuration for both relay server and client
type Config struct {
	// Server configuration
	ServerPort string `env:"SERVER_PORT" envDefault:"8080"`

	// Redis configuration
	RedisURL      string `env:"REDIS_URL" envDefault:"localhost:6379"`
	RedisPassword string `env:"REDIS_PASSWORD" envDefault:""`
	RedisDB       int    `env:"REDIS_DB" envDefault:"0"`

	// Stream configuration
	StreamName         string `env:"STREAM_NAME" envDefault:"webhook-stream"`
	ConsumerGroup      string `env:"CONSUMER_GROUP" envDefault:"relay-group"`
	ConsumerName       string `env:"CONSUMER_NAME" envDefault:"relay-client"`
	DeadLetterQueue    string `env:"DEAD_LETTER_QUEUE" envDefault:"webhook-dlq"`
	MessageTTL         int    `env:"MESSAGE_TTL" envDefault:"86400"` // 24 hours in seconds

	// Authentication
	APIKey string `env:"API_KEY" envDefault:""`

	// Client configuration
	LocalWebhookURL string `env:"LOCAL_WEBHOOK_URL" envDefault:"http://localhost:3000/webhook"`

	// Retry configuration
	MaxRetries      int `env:"MAX_RETRIES" envDefault:"3"`
	RetryDelay      int `env:"RETRY_DELAY" envDefault:"1000"` // milliseconds
	RetryMultiplier float64 `env:"RETRY_MULTIPLIER" envDefault:"2.0"`

	// Health check
	HealthCheckInterval int `env:"HEALTH_CHECK_INTERVAL" envDefault:"30"` // seconds
}

// Metrics holds runtime metrics
type Metrics struct {
	WebhooksReceived   int64 `json:"webhooks_received"`
	WebhooksProcessed  int64 `json:"webhooks_processed"`
	WebhooksFailed     int64 `json:"webhooks_failed"`
	WebhooksRetried    int64 `json:"webhooks_retried"`
	QueueDepth         int64 `json:"queue_depth"`
	AverageLatency     int64 `json:"average_latency_ms"`
	LastWebhookTime    time.Time `json:"last_webhook_time"`
}

// Error types
type RelayError struct {
	Code    string
	Message string
	Err     error
}

func (e *RelayError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *RelayError) Unwrap() error {
	return e.Err
}

// Error codes
const (
	ErrCodeInvalidRequest   = "INVALID_REQUEST"
	ErrCodeAuthentication   = "AUTHENTICATION_FAILED"
	ErrCodeRedisConnection  = "REDIS_CONNECTION_ERROR"
	ErrCodeStreamWrite      = "STREAM_WRITE_ERROR"
	ErrCodeStreamRead       = "STREAM_READ_ERROR"
	ErrCodeWebhookForward   = "WEBHOOK_FORWARD_ERROR"
	ErrCodeMaxRetriesExceeded = "MAX_RETRIES_EXCEEDED"
	ErrCodeInvalidConfig    = "INVALID_CONFIG"
)

// NewRelayError creates a new RelayError
func NewRelayError(code, message string, err error) *RelayError {
	return &RelayError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}
