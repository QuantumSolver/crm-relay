package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yourusername/crm-relay/internal/models"
)

// RedisClient wraps the Redis client with stream operations
type RedisClient struct {
	client *redis.Client
	config *models.Config
}

// NewRedisClient creates a new Redis client
func NewRedisClient(cfg *models.Config) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisURL,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
		PoolSize: 10,
		MinIdleConns: 5,
		MaxRetries: 3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, models.NewRelayError(
			models.ErrCodeRedisConnection,
			"failed to connect to Redis",
			err,
		)
	}

	redisClient := &RedisClient{
		client: client,
		config: cfg,
	}

	// Initialize consumer group
	if err := redisClient.initConsumerGroup(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize consumer group: %w", err)
	}

	return redisClient, nil
}

// initConsumerGroup initializes the consumer group if it doesn't exist
func (r *RedisClient) initConsumerGroup(ctx context.Context) error {
	// Try to create consumer group
	err := r.client.XGroupCreate(ctx, r.config.StreamName, r.config.ConsumerGroup, "0").Err()
	if err != nil {
		// If group already exists, that's fine
		if strings.Contains(err.Error(), "BUSYGROUP") {
			return nil
		}
		return models.NewRelayError(
			models.ErrCodeRedisConnection,
			"failed to create consumer group",
			err,
		)
	}
	return nil
}

// AddWebhook adds a webhook to the Redis stream
func (r *RedisClient) AddWebhook(ctx context.Context, webhook *models.Webhook) (string, error) {
	// Create relay message
	message := models.RelayMessage{
		MessageID:  webhook.ID,
		Webhook:    *webhook,
		RetryCount: 0,
		CreatedAt:  time.Now(),
	}

	messageJSON, err := json.Marshal(message)
	if err != nil {
		return "", models.NewRelayError(
			models.ErrCodeStreamWrite,
			"failed to serialize relay message",
			err,
		)
	}

	// Add to stream
	id, err := r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: r.config.StreamName,
		Values: map[string]interface{}{
			"data": messageJSON,
		},
	}).Result()

	if err != nil {
		return "", models.NewRelayError(
			models.ErrCodeStreamWrite,
			"failed to add webhook to stream",
			err,
		)
	}

	// Set TTL on stream key
	r.client.Expire(ctx, r.config.StreamName, time.Duration(r.config.MessageTTL)*time.Second)

	return id, nil
}

// ReadMessages reads messages from the stream for the consumer
func (r *RedisClient) ReadMessages(ctx context.Context, count int64, block time.Duration) ([]redis.XMessage, error) {
	// Read messages from consumer group
	messages, err := r.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    r.config.ConsumerGroup,
		Consumer: r.config.ConsumerName,
		Streams:  []string{r.config.StreamName, ">"},
		Count:    count,
		Block:    block,
	}).Result()

	if err != nil && err != redis.Nil {
		return nil, models.NewRelayError(
			models.ErrCodeStreamRead,
			"failed to read messages from stream",
			err,
		)
	}

	if len(messages) == 0 {
		return nil, nil
	}

	return messages[0].Messages, nil
}

// AcknowledgeMessage acknowledges a message as processed
func (r *RedisClient) AcknowledgeMessage(ctx context.Context, messageID string) error {
	err := r.client.XAck(ctx, r.config.StreamName, r.config.ConsumerGroup, messageID).Err()
	if err != nil {
		return models.NewRelayError(
			models.ErrCodeStreamRead,
			"failed to acknowledge message",
			err,
		)
	}
	return nil
}

// MoveToDeadLetterQueue moves a message to the dead letter queue
func (r *RedisClient) MoveToDeadLetterQueue(ctx context.Context, messageID string, message *models.RelayMessage) error {
	// Serialize message
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return models.NewRelayError(
			models.ErrCodeStreamWrite,
			"failed to serialize message for DLQ",
			err,
		)
	}

	// Add to dead letter queue
	_, err = r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: r.config.DeadLetterQueue,
		Values: map[string]interface{}{
			"original_id": messageID,
			"data":        messageJSON,
			"moved_at":    time.Now().Unix(),
		},
	}).Result()

	if err != nil {
		return models.NewRelayError(
			models.ErrCodeStreamWrite,
			"failed to add message to dead letter queue",
			err,
		)
	}

	// Acknowledge the original message
	if err := r.AcknowledgeMessage(ctx, messageID); err != nil {
		return fmt.Errorf("failed to acknowledge message after moving to DLQ: %w", err)
	}

	return nil
}

// GetQueueDepth returns the current queue depth
func (r *RedisClient) GetQueueDepth(ctx context.Context) (int64, error) {
	length, err := r.client.XLen(ctx, r.config.StreamName).Result()
	if err != nil {
		return 0, models.NewRelayError(
			models.ErrCodeRedisConnection,
			"failed to get queue depth",
			err,
		)
	}
	return length, nil
}

// GetPendingMessages returns the number of pending messages for the consumer
func (r *RedisClient) GetPendingMessages(ctx context.Context) (int64, error) {
	pending, err := r.client.XPending(ctx, r.config.StreamName, r.config.ConsumerGroup).Result()
	if err != nil {
		return 0, models.NewRelayError(
			models.ErrCodeRedisConnection,
			"failed to get pending messages",
			err,
		)
	}
	return pending.Count, nil
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// ParseMessage parses a Redis stream message into a RelayMessage
func ParseMessage(message redis.XMessage) (*models.RelayMessage, error) {
	data, ok := message.Values["data"].(string)
	if !ok {
		return nil, fmt.Errorf("message data is not a string")
	}

	var relayMessage models.RelayMessage
	if err := json.Unmarshal([]byte(data), &relayMessage); err != nil {
		return nil, fmt.Errorf("failed to unmarshal relay message: %w", err)
	}

	return &relayMessage, nil
}
