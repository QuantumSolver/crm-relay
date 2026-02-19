package relayclient

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yourusername/crm-relay/internal/models"
	"github.com/yourusername/crm-relay/internal/storage"
)

// Consumer consumes messages from Redis stream
type Consumer struct {
	redisClient *storage.RedisClient
	config      *models.Config
	forwarder   *Forwarder
	metrics     *models.Metrics
	running     atomic.Bool
}

// NewConsumer creates a new consumer
func NewConsumer(redisClient *storage.RedisClient, config *models.Config, forwarder *Forwarder) *Consumer {
	return &Consumer{
		redisClient: redisClient,
		config:      config,
		forwarder:   forwarder,
		metrics:     &models.Metrics{},
	}
}

// Start starts consuming messages
func (c *Consumer) Start(ctx context.Context) {
	c.running.Store(true)
	log.Printf("Consumer started: Group=%s, Consumer=%s", c.config.ConsumerGroup, c.config.ConsumerName)

	for c.running.Load() {
		select {
		case <-ctx.Done():
			log.Println("Consumer stopping due to context cancellation")
			return
		default:
			c.consumeMessages(ctx)
		}
	}
}

// Stop stops the consumer
func (c *Consumer) Stop() {
	c.running.Store(false)
	log.Println("Consumer stopped")
}

// consumeMessages reads and processes messages from the stream
func (c *Consumer) consumeMessages(ctx context.Context) {
	// Read messages with blocking
	messages, err := c.redisClient.ReadMessages(ctx, 10, 5*time.Second)
	if err != nil {
		log.Printf("Error reading messages: %v", err)
		time.Sleep(5 * time.Second)
		return
	}

	if len(messages) == 0 {
		return
	}

	log.Printf("Received %d messages from stream", len(messages))

	// Process each message
	for _, message := range messages {
		if !c.running.Load() {
			break
		}

		c.processMessage(ctx, message)
	}
}

// processMessage processes a single message
func (c *Consumer) processMessage(ctx context.Context, message interface{}) {
	// Parse message
	redisMessage, ok := message.(redis.XMessage)
	if !ok {
		log.Printf("Invalid message type")
		return
	}

	// Parse relay message
	relayMessage, err := storage.ParseMessage(redisMessage)
	if err != nil {
		log.Printf("Failed to parse message %s: %v", redisMessage.ID, err)
		return
	}

	// Log routing information
	if relayMessage.Webhook.Platform != "" {
		log.Printf("Processing message: ID=%s, WebhookID=%s, Platform=%s, EndpointID=%s, HTTPMethod=%s, RetryCount=%d",
			redisMessage.ID, relayMessage.Webhook.ID, relayMessage.Webhook.Platform,
			relayMessage.Webhook.EndpointID, relayMessage.Webhook.HTTPMethod, relayMessage.RetryCount)
	} else {
		log.Printf("Processing message: ID=%s, WebhookID=%s, RetryCount=%d",
			redisMessage.ID, relayMessage.Webhook.ID, relayMessage.RetryCount)
	}

	// Forward webhook
	err = c.forwarder.Forward(ctx, &relayMessage.Webhook)
	if err != nil {
		log.Printf("Failed to forward webhook %s: %v", relayMessage.Webhook.ID, err)
		c.handleForwardError(ctx, redisMessage.ID, relayMessage, err)
		return
	}

	// Acknowledge message
	if err := c.redisClient.AcknowledgeMessage(ctx, redisMessage.ID); err != nil {
		log.Printf("Failed to acknowledge message %s: %v", redisMessage.ID, err)
		return
	}

	// Update metrics
	atomic.AddInt64(&c.metrics.WebhooksProcessed, 1)

	log.Printf("Successfully processed and acknowledged message: ID=%s", redisMessage.ID)
}

// handleForwardError handles forwarding errors with retry logic
func (c *Consumer) handleForwardError(ctx context.Context, messageID string, relayMessage *models.RelayMessage, err error) {
	// Increment retry count
	relayMessage.RetryCount++
	atomic.AddInt64(&c.metrics.WebhooksRetried, 1)

	// Check if max retries exceeded
	if relayMessage.RetryCount >= c.config.MaxRetries {
		log.Printf("Max retries exceeded for webhook %s, moving to DLQ", relayMessage.Webhook.ID)
		atomic.AddInt64(&c.metrics.WebhooksFailed, 1)

		// Move to dead letter queue
		if dlqErr := c.redisClient.MoveToDeadLetterQueue(ctx, messageID, relayMessage); dlqErr != nil {
			log.Printf("Failed to move message to DLQ: %v", dlqErr)
		}

		return
	}

	// Calculate retry delay with exponential backoff
	delay := time.Duration(c.config.RetryDelay) * time.Millisecond
	for i := 1; i < relayMessage.RetryCount; i++ {
		delay = time.Duration(float64(delay) * c.config.RetryMultiplier)
	}

	log.Printf("Retrying webhook %s in %v (attempt %d/%d)",
		relayMessage.Webhook.ID, delay, relayMessage.RetryCount, c.config.MaxRetries)

	// Re-add to stream with updated retry count
	// Note: In production, you might want to use a separate retry queue
	// For simplicity, we'll just log and let the consumer pick it up again
	time.Sleep(delay)
}

// GetMetrics returns the current metrics
func (c *Consumer) GetMetrics() *models.Metrics {
	return c.metrics
}
