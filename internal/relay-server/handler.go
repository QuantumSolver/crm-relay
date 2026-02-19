package relayserver

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/crm-relay/internal/models"
	"github.com/yourusername/crm-relay/internal/storage"
)

// Handler handles HTTP requests for the relay server
type Handler struct {
	redisClient *storage.RedisClient
	config      *models.Config
	metrics     *models.Metrics
}

// NewHandler creates a new handler
func NewHandler(redisClient *storage.RedisClient, config *models.Config) *Handler {
	return &Handler{
		redisClient: redisClient,
		config:      config,
		metrics:     &models.Metrics{},
	}
}

// HandleWebhook handles incoming webhook requests
func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Validate method
	if r.Method != http.MethodPost {
		sendErrorResponse(w, http.StatusMethodNotAllowed, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"method not allowed",
			nil,
		))
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendErrorResponse(w, http.StatusBadRequest, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"failed to read request body",
			err,
		))
		return
	}
	defer r.Body.Close()

	// Validate body is not empty
	if len(body) == 0 {
		sendErrorResponse(w, http.StatusBadRequest, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"request body cannot be empty",
			nil,
		))
		return
	}

	// Collect headers
	headers := make(map[string]string)
	for key, values := range r.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// Create webhook
	webhook := &models.Webhook{
		ID:        uuid.New().String(),
		Headers:   headers,
		Body:      body,
		Timestamp: time.Now(),
		Signature: r.Header.Get("X-Hub-Signature"),
	}

	// Add to Redis stream
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	messageID, err := h.redisClient.AddWebhook(ctx, webhook)
	if err != nil {
		log.Printf("Failed to add webhook to stream: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, err.(*models.RelayError))
		return
	}

	// Update metrics
	atomic.AddInt64(&h.metrics.WebhooksReceived, 1)
	h.metrics.LastWebhookTime = time.Now()
	latency := time.Since(start).Milliseconds()
	atomic.StoreInt64(&h.metrics.AverageLatency, latency)

	// Send success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"message_id": messageID,
		"webhook_id": webhook.ID,
		"timestamp":  webhook.Timestamp,
	})

	log.Printf("Webhook received and queued: ID=%s, MessageID=%s, Latency=%dms", webhook.ID, messageID, latency)
}

// HandleHealth handles health check requests
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Check Redis connection
	redisStatus := "ok"
	queueDepth, err := h.redisClient.GetQueueDepth(ctx)
	if err != nil {
		redisStatus = "error"
	}

	// Prepare health response
	health := map[string]interface{}{
		"status": "healthy",
		"timestamp": time.Now(),
		"redis": map[string]interface{}{
			"status":      redisStatus,
			"queue_depth": queueDepth,
		},
		"metrics": map[string]interface{}{
			"webhooks_received":  atomic.LoadInt64(&h.metrics.WebhooksReceived),
			"webhooks_processed": atomic.LoadInt64(&h.metrics.WebhooksProcessed),
			"webhooks_failed":     atomic.LoadInt64(&h.metrics.WebhooksFailed),
			"webhooks_retried":    atomic.LoadInt64(&h.metrics.WebhooksRetried),
			"average_latency_ms":  atomic.LoadInt64(&h.metrics.AverageLatency),
			"last_webhook_time":   h.metrics.LastWebhookTime,
		},
	}

	if redisStatus != "ok" {
		health["status"] = "degraded"
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// GetMetrics returns the current metrics
func (h *Handler) GetMetrics() *models.Metrics {
	return h.metrics
}
