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
	// Try to create consumer group with MKSTREAM option to create stream if it doesn't exist
	err := r.client.XGroupCreateMkStream(ctx, r.config.StreamName, r.config.ConsumerGroup, "0").Err()
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

// User management methods

// StoreUser stores a user in Redis
func (r *RedisClient) StoreUser(ctx context.Context, user *models.User) error {
	userJSON, err := json.Marshal(user)
	if err != nil {
		return models.NewRelayError(
			models.ErrCodeStreamWrite,
			"failed to serialize user",
			err,
		)
	}

	key := fmt.Sprintf("user:%s", user.Username)
	if err := r.client.Set(ctx, key, userJSON, 0).Err(); err != nil {
		return models.NewRelayError(
			models.ErrCodeRedisConnection,
			"failed to store user",
			err,
		)
	}

	return nil
}

// GetUser retrieves a user by username
func (r *RedisClient) GetUser(ctx context.Context, username string) (*models.User, error) {
	key := fmt.Sprintf("user:%s", username)
	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, models.NewRelayError(
				models.ErrCodeAuthentication,
				"user not found",
				nil,
			)
		}
		return nil, models.NewRelayError(
			models.ErrCodeRedisConnection,
			"failed to retrieve user",
			err,
		)
	}

	var user models.User
	if err := json.Unmarshal([]byte(data), &user); err != nil {
		return nil, models.NewRelayError(
			models.ErrCodeStreamRead,
			"failed to unmarshal user",
			err,
		)
	}

	return &user, nil
}

// ValidateUser validates a user's credentials
func (r *RedisClient) ValidateUser(ctx context.Context, username, password string) (*models.User, error) {
	user, err := r.GetUser(ctx, username)
	if err != nil {
		return nil, err
	}

	// Password validation is done in the auth package using bcrypt
	return user, nil
}

// API Key management methods

// CreateAPIKey creates a new API key
func (r *RedisClient) CreateAPIKey(ctx context.Context, apiKey *models.APIKey) error {
	apiKeyJSON, err := json.Marshal(apiKey)
	if err != nil {
		return models.NewRelayError(
			models.ErrCodeStreamWrite,
			"failed to serialize API key",
			err,
		)
	}

	// Store by ID
	key := fmt.Sprintf("apikey:%s", apiKey.ID)
	if err := r.client.Set(ctx, key, apiKeyJSON, 0).Err(); err != nil {
		return models.NewRelayError(
			models.ErrCodeRedisConnection,
			"failed to store API key",
			err,
		)
	}

	// Store by key value for quick lookup
	lookupKey := fmt.Sprintf("apikey:lookup:%s", apiKey.Key)
	if err := r.client.Set(ctx, lookupKey, apiKey.ID, 0).Err(); err != nil {
		return models.NewRelayError(
			models.ErrCodeRedisConnection,
			"failed to store API key lookup",
			err,
		)
	}

	// Add to platform index
	platformKey := fmt.Sprintf("apikey:platform:%s", apiKey.Platform)
	if err := r.client.SAdd(ctx, platformKey, apiKey.ID).Err(); err != nil {
		return models.NewRelayError(
			models.ErrCodeRedisConnection,
			"failed to add to platform index",
			err,
		)
	}

	return nil
}

// GetAPIKey retrieves an API key by ID
func (r *RedisClient) GetAPIKey(ctx context.Context, id string) (*models.APIKey, error) {
	key := fmt.Sprintf("apikey:%s", id)
	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, models.NewRelayError(
				models.ErrCodeInvalidRequest,
				"API key not found",
				nil,
			)
		}
		return nil, models.NewRelayError(
			models.ErrCodeRedisConnection,
			"failed to retrieve API key",
			err,
		)
	}

	var apiKey models.APIKey
	if err := json.Unmarshal([]byte(data), &apiKey); err != nil {
		return nil, models.NewRelayError(
			models.ErrCodeStreamRead,
			"failed to unmarshal API key",
			err,
		)
	}

	return &apiKey, nil
}

// GetAPIKeyByValue retrieves an API key by its value
func (r *RedisClient) GetAPIKeyByValue(ctx context.Context, key string) (*models.APIKey, error) {
	lookupKey := fmt.Sprintf("apikey:lookup:%s", key)
	id, err := r.client.Get(ctx, lookupKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, models.NewRelayError(
				models.ErrCodeAuthentication,
				"API key not found",
				nil,
			)
		}
		return nil, models.NewRelayError(
			models.ErrCodeRedisConnection,
			"failed to lookup API key",
			err,
		)
	}

	return r.GetAPIKey(ctx, id)
}

// ListAPIKeys lists all API keys
func (r *RedisClient) ListAPIKeys(ctx context.Context) ([]*models.APIKey, error) {
	iter := r.client.Scan(ctx, 0, "apikey:*", 0).Iterator()
	var apiKeys []*models.APIKey

	for iter.Next(ctx) {
		key := iter.Val()
		if strings.Contains(key, ":lookup:") || strings.Contains(key, ":platform:") {
			continue
		}

		data, err := r.client.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		var apiKey models.APIKey
		if err := json.Unmarshal([]byte(data), &apiKey); err != nil {
			continue
		}

		apiKeys = append(apiKeys, &apiKey)
	}

	if err := iter.Err(); err != nil {
		return nil, models.NewRelayError(
			models.ErrCodeRedisConnection,
			"failed to scan API keys",
			err,
		)
	}

	return apiKeys, nil
}

// UpdateAPIKey updates an existing API key
func (r *RedisClient) UpdateAPIKey(ctx context.Context, apiKey *models.APIKey) error {
	apiKey.UpdatedAt = time.Now()
	return r.CreateAPIKey(ctx, apiKey)
}

// DeleteAPIKey deletes an API key
func (r *RedisClient) DeleteAPIKey(ctx context.Context, id string) error {
	// Get the API key first to get its value and platform
	apiKey, err := r.GetAPIKey(ctx, id)
	if err != nil {
		return err
	}

	// Delete by ID
	key := fmt.Sprintf("apikey:%s", id)
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return models.NewRelayError(
			models.ErrCodeRedisConnection,
			"failed to delete API key",
			err,
		)
	}

	// Delete lookup
	lookupKey := fmt.Sprintf("apikey:lookup:%s", apiKey.Key)
	if err := r.client.Del(ctx, lookupKey).Err(); err != nil {
		return models.NewRelayError(
			models.ErrCodeRedisConnection,
			"failed to delete API key lookup",
			err,
		)
	}

	// Remove from platform index
	platformKey := fmt.Sprintf("apikey:platform:%s", apiKey.Platform)
	if err := r.client.SRem(ctx, platformKey, id).Err(); err != nil {
		return models.NewRelayError(
			models.ErrCodeRedisConnection,
			"failed to remove from platform index",
			err,
		)
	}

	return nil
}

// Webhook Endpoint management methods

// CreateEndpoint creates a new webhook endpoint
func (r *RedisClient) CreateEndpoint(ctx context.Context, endpoint *models.WebhookEndpoint) error {
	endpointJSON, err := json.Marshal(endpoint)
	if err != nil {
		return models.NewRelayError(
			models.ErrCodeStreamWrite,
			"failed to serialize endpoint",
			err,
		)
	}

	// Store by ID
	key := fmt.Sprintf("endpoint:%s", endpoint.ID)
	if err := r.client.Set(ctx, key, endpointJSON, 0).Err(); err != nil {
		return models.NewRelayError(
			models.ErrCodeRedisConnection,
			"failed to store endpoint",
			err,
		)
	}

	// Store by path for routing lookup
	pathKey := fmt.Sprintf("endpoint:path:%s", endpoint.Path)
	if err := r.client.Set(ctx, pathKey, endpoint.ID, 0).Err(); err != nil {
		return models.NewRelayError(
			models.ErrCodeRedisConnection,
			"failed to store endpoint path lookup",
			err,
		)
	}

	return nil
}

// GetEndpoint retrieves an endpoint by ID
func (r *RedisClient) GetEndpoint(ctx context.Context, id string) (*models.WebhookEndpoint, error) {
	key := fmt.Sprintf("endpoint:%s", id)
	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, models.NewRelayError(
				models.ErrCodeInvalidRequest,
				"endpoint not found",
				nil,
			)
		}
		return nil, models.NewRelayError(
			models.ErrCodeRedisConnection,
			"failed to retrieve endpoint",
			err,
		)
	}

	var endpoint models.WebhookEndpoint
	if err := json.Unmarshal([]byte(data), &endpoint); err != nil {
		return nil, models.NewRelayError(
			models.ErrCodeStreamRead,
			"failed to unmarshal endpoint",
			err,
		)
	}

	return &endpoint, nil
}

// GetEndpointByPath retrieves an endpoint by its path
func (r *RedisClient) GetEndpointByPath(ctx context.Context, path string) (*models.WebhookEndpoint, error) {
	pathKey := fmt.Sprintf("endpoint:path:%s", path)
	id, err := r.client.Get(ctx, pathKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, models.NewRelayError(
				models.ErrCodeInvalidRequest,
				"endpoint not found for path",
				nil,
			)
		}
		return nil, models.NewRelayError(
			models.ErrCodeRedisConnection,
			"failed to lookup endpoint by path",
			err,
		)
	}

	return r.GetEndpoint(ctx, id)
}

// ListEndpoints lists all endpoints
func (r *RedisClient) ListEndpoints(ctx context.Context) ([]*models.WebhookEndpoint, error) {
	iter := r.client.Scan(ctx, 0, "endpoint:*", 0).Iterator()
	var endpoints []*models.WebhookEndpoint

	for iter.Next(ctx) {
		key := iter.Val()
		if strings.Contains(key, ":path:") {
			continue
		}

		data, err := r.client.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		var endpoint models.WebhookEndpoint
		if err := json.Unmarshal([]byte(data), &endpoint); err != nil {
			continue
		}

		endpoints = append(endpoints, &endpoint)
	}

	if err := iter.Err(); err != nil {
		return nil, models.NewRelayError(
			models.ErrCodeRedisConnection,
			"failed to scan endpoints",
			err,
		)
	}

	return endpoints, nil
}

// UpdateEndpoint updates an existing endpoint
func (r *RedisClient) UpdateEndpoint(ctx context.Context, endpoint *models.WebhookEndpoint) error {
	endpoint.UpdatedAt = time.Now()
	return r.CreateEndpoint(ctx, endpoint)
}

// DeleteEndpoint deletes an endpoint
func (r *RedisClient) DeleteEndpoint(ctx context.Context, id string) error {
	// Get the endpoint first to get its path
	endpoint, err := r.GetEndpoint(ctx, id)
	if err != nil {
		return err
	}

	// Delete by ID
	key := fmt.Sprintf("endpoint:%s", id)
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return models.NewRelayError(
			models.ErrCodeRedisConnection,
			"failed to delete endpoint",
			err,
		)
	}

	// Delete path lookup
	pathKey := fmt.Sprintf("endpoint:path:%s", endpoint.Path)
	if err := r.client.Del(ctx, pathKey).Err(); err != nil {
		return models.NewRelayError(
			models.ErrCodeRedisConnection,
			"failed to delete endpoint path lookup",
			err,
		)
	}

	return nil
}

// InitializeDefaultUser creates a default admin user if none exists
func (r *RedisClient) InitializeDefaultUser(ctx context.Context, username, passwordHash string) error {
	// Check if user already exists
	_, err := r.GetUser(ctx, username)
	if err == nil {
		// User already exists
		return nil
	}

	// Create default admin user
	user := &models.User{
		ID:           "admin",
		Username:     username,
		PasswordHash: passwordHash,
		Role:         "admin",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	return r.StoreUser(ctx, user)
}

// Dead Letter Queue methods

// ReadDLQMessages reads messages from the dead letter queue
func (r *RedisClient) ReadDLQMessages(ctx context.Context, count int64) ([]*models.RelayMessage, error) {
	messages, err := r.client.XRevRange(ctx, r.config.DeadLetterQueue, "+", "-").Result()
	if err != nil && err != redis.Nil {
		return nil, models.NewRelayError(
			models.ErrCodeStreamRead,
			"failed to read DLQ messages",
			err,
		)
	}

	var relayMessages []*models.RelayMessage
	for i, msg := range messages {
		if int64(i) >= count {
			break
		}

		data, ok := msg.Values["data"].(string)
		if !ok {
			continue
		}

		var relayMessage models.RelayMessage
		if err := json.Unmarshal([]byte(data), &relayMessage); err != nil {
			continue
		}

		relayMessages = append(relayMessages, &relayMessage)
	}

	return relayMessages, nil
}

// GetDLQMessage retrieves a specific message from the DLQ
func (r *RedisClient) GetDLQMessage(ctx context.Context, messageID string) (*models.RelayMessage, error) {
	// Scan DLQ for the message
	messages, err := r.client.XRange(ctx, r.config.DeadLetterQueue, "-", "+").Result()
	if err != nil {
		return nil, models.NewRelayError(
			models.ErrCodeStreamRead,
			"failed to read DLQ",
			err,
		)
	}

	for _, msg := range messages {
		if msg.ID == messageID {
			data, ok := msg.Values["data"].(string)
			if !ok {
				continue
			}

			var relayMessage models.RelayMessage
			if err := json.Unmarshal([]byte(data), &relayMessage); err != nil {
				continue
			}

			return &relayMessage, nil
		}
	}

	return nil, models.NewRelayError(
		models.ErrCodeInvalidRequest,
		"message not found in DLQ",
		nil,
	)
}

// DeleteDLQMessage deletes a message from the DLQ
func (r *RedisClient) DeleteDLQMessage(ctx context.Context, messageID string) error {
	err := r.client.XDel(ctx, r.config.DeadLetterQueue, messageID).Err()
	if err != nil {
		return models.NewRelayError(
			models.ErrCodeRedisConnection,
			"failed to delete DLQ message",
			err,
		)
	}
	return nil
}
