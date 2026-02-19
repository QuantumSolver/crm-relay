package relayclient

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/QuantumSolver/crm-relay/internal/auth"
	"github.com/QuantumSolver/crm-relay/internal/models"
	"github.com/QuantumSolver/crm-relay/internal/storage"
)

// Handler handles HTTP requests for the relay client
type Handler struct {
	redisClient *storage.RedisClient
	config      *models.Config
	metrics     *models.Metrics
	jwtService  *auth.JWTService
}

// NewHandler creates a new handler
func NewHandler(redisClient *storage.RedisClient, config *models.Config, jwtService *auth.JWTService) *Handler {
	return &Handler{
		redisClient: redisClient,
		config:      config,
		metrics:     &models.Metrics{},
		jwtService:  jwtService,
	}
}

// GetMetrics returns the current metrics
func (h *Handler) GetMetrics() *models.Metrics {
	return h.metrics
}

// Auth endpoints

// HandleLogin handles login requests
func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorResponse(w, http.StatusMethodNotAllowed, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"method not allowed",
			nil,
		))
		return
	}

	var loginReq models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		sendErrorResponse(w, http.StatusBadRequest, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"invalid request body",
			err,
		))
		return
	}

	// Validate credentials
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user, err := h.redisClient.GetUser(ctx, loginReq.Username)
	if err != nil {
		sendErrorResponse(w, http.StatusUnauthorized, models.NewRelayError(
			models.ErrCodeAuthentication,
			"invalid credentials",
			nil,
		))
		return
	}

	// Verify password
	if !auth.VerifyPassword(loginReq.Password, user.PasswordHash) {
		sendErrorResponse(w, http.StatusUnauthorized, models.NewRelayError(
			models.ErrCodeAuthentication,
			"invalid credentials",
			nil,
		))
		return
	}

	// Generate JWT token
	token, expiresAt, err := h.jwtService.GenerateToken(user)
	if err != nil {
		log.Printf("Failed to generate token: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, models.NewRelayError(
			models.ErrCodeAuthentication,
			"failed to generate token",
			err,
		))
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(models.LoginResponse{
		Token:     token,
		User:      *user,
		ExpiresAt: expiresAt,
	})

	log.Printf("User logged in: %s", user.Username)
}

// HandleGetCurrentUser handles requests to get the current user
func (h *Handler) HandleGetCurrentUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendErrorResponse(w, http.StatusMethodNotAllowed, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"method not allowed",
			nil,
		))
		return
	}

	// Get user from context
	claims, ok := r.Context().Value("user").(*models.JWTClaims)
	if !ok {
		sendErrorResponse(w, http.StatusUnauthorized, models.NewRelayError(
			models.ErrCodeAuthentication,
			"user not authenticated",
			nil,
		))
		return
	}

	// Get user from storage
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user, err := h.redisClient.GetUser(ctx, claims.Username)
	if err != nil {
		sendErrorResponse(w, http.StatusNotFound, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"user not found",
			err,
		))
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}

// Configuration endpoints

// HandleUpdateLocalEndpoint handles requests to update the local webhook endpoint
func (h *Handler) HandleUpdateLocalEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		sendErrorResponse(w, http.StatusMethodNotAllowed, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"method not allowed",
			nil,
		))
		return
	}

	var req struct {
		LocalWebhookURL string `json:"local_webhook_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, http.StatusBadRequest, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"invalid request body",
			err,
		))
		return
	}

	// Update config (in a real implementation, this would persist to storage)
	h.config.LocalWebhookURL = req.LocalWebhookURL

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":           true,
		"local_webhook_url": h.config.LocalWebhookURL,
	})

	log.Printf("Local webhook endpoint updated: %s", h.config.LocalWebhookURL)
}

// HandleUpdateRetryConfig handles requests to update retry configuration
func (h *Handler) HandleUpdateRetryConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		sendErrorResponse(w, http.StatusMethodNotAllowed, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"method not allowed",
			nil,
		))
		return
	}

	var req struct {
		MaxRetries      *int     `json:"max_retries"`
		RetryDelay      *int     `json:"retry_delay"`
		RetryMultiplier *float64 `json:"retry_multiplier"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, http.StatusBadRequest, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"invalid request body",
			err,
		))
		return
	}

	// Update config
	if req.MaxRetries != nil {
		h.config.MaxRetries = *req.MaxRetries
	}
	if req.RetryDelay != nil {
		h.config.RetryDelay = *req.RetryDelay
	}
	if req.RetryMultiplier != nil {
		h.config.RetryMultiplier = *req.RetryMultiplier
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":          true,
		"max_retries":      h.config.MaxRetries,
		"retry_delay":      h.config.RetryDelay,
		"retry_multiplier": h.config.RetryMultiplier,
	})

	log.Printf("Retry config updated: MaxRetries=%d, RetryDelay=%d, RetryMultiplier=%.2f",
		h.config.MaxRetries, h.config.RetryDelay, h.config.RetryMultiplier)
}

// Dead Letter Queue endpoints

// HandleGetDLQMessages handles requests to get DLQ messages
func (h *Handler) HandleGetDLQMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendErrorResponse(w, http.StatusMethodNotAllowed, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"method not allowed",
			nil,
		))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Read messages from DLQ
	messages, err := h.redisClient.ReadDLQMessages(ctx, 100)
	if err != nil {
		log.Printf("Failed to read DLQ messages: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, err.(*models.RelayError))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"messages": messages,
		"count":    len(messages),
	})
}

// HandleReplayDLQMessage handles requests to replay a DLQ message
func (h *Handler) HandleReplayDLQMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorResponse(w, http.StatusMethodNotAllowed, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"method not allowed",
			nil,
		))
		return
	}

	// Extract message ID from URL path
	messageID := r.URL.Path[len("/api/dlq/"):]
	if messageID == "" {
		sendErrorResponse(w, http.StatusBadRequest, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"missing message ID",
			nil,
		))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get message from DLQ
	message, err := h.redisClient.GetDLQMessage(ctx, messageID)
	if err != nil {
		sendErrorResponse(w, http.StatusNotFound, err.(*models.RelayError))
		return
	}

	// Re-add to main stream
	_, err = h.redisClient.AddWebhook(ctx, &message.Webhook)
	if err != nil {
		log.Printf("Failed to replay message: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, err.(*models.RelayError))
		return
	}

	// Remove from DLQ
	if err := h.redisClient.DeleteDLQMessage(ctx, messageID); err != nil {
		log.Printf("Failed to delete message from DLQ: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"message":    "Message replayed successfully",
		"message_id": messageID,
	})

	log.Printf("DLQ message replayed: %s", messageID)
}

// HandleDeleteDLQMessage handles requests to delete a DLQ message
func (h *Handler) HandleDeleteDLQMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		sendErrorResponse(w, http.StatusMethodNotAllowed, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"method not allowed",
			nil,
		))
		return
	}

	// Extract message ID from URL path
	messageID := r.URL.Path[len("/api/dlq/"):]
	if messageID == "" {
		sendErrorResponse(w, http.StatusBadRequest, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"missing message ID",
			nil,
		))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.redisClient.DeleteDLQMessage(ctx, messageID); err != nil {
		log.Printf("Failed to delete DLQ message: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, err.(*models.RelayError))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"message":    "Message deleted successfully",
		"message_id": messageID,
	})

	log.Printf("DLQ message deleted: %s", messageID)
}

// Metrics endpoints

// HandleGetMetrics handles requests to get metrics
func (h *Handler) HandleGetMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendErrorResponse(w, http.StatusMethodNotAllowed, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"method not allowed",
			nil,
		))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Get queue depth
	queueDepth, err := h.redisClient.GetQueueDepth(ctx)
	if err != nil {
		log.Printf("Failed to get queue depth: %v", err)
	}

	// Get pending messages
	pendingMessages, err := h.redisClient.GetPendingMessages(ctx)
	if err != nil {
		log.Printf("Failed to get pending messages: %v", err)
	}

	metrics := map[string]interface{}{
		"webhooks_received":  atomic.LoadInt64(&h.metrics.WebhooksReceived),
		"webhooks_processed": atomic.LoadInt64(&h.metrics.WebhooksProcessed),
		"webhooks_failed":     atomic.LoadInt64(&h.metrics.WebhooksFailed),
		"webhooks_retried":    atomic.LoadInt64(&h.metrics.WebhooksRetried),
		"queue_depth":         queueDepth,
		"pending_messages":    pendingMessages,
		"average_latency_ms":  atomic.LoadInt64(&h.metrics.AverageLatency),
		"last_webhook_time":   h.metrics.LastWebhookTime,
		"config": map[string]interface{}{
			"local_webhook_url": h.config.LocalWebhookURL,
			"max_retries":       h.config.MaxRetries,
			"retry_delay":       h.config.RetryDelay,
			"retry_multiplier":  h.config.RetryMultiplier,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metrics)
}

// sendErrorResponse sends an error response as JSON
func sendErrorResponse(w http.ResponseWriter, statusCode int, err *models.RelayError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    err.Code,
			"message": err.Message,
		},
	}

	if err.Err != nil {
		response["error"].(map[string]interface{})["details"] = err.Err.Error()
	}

	json.NewEncoder(w).Encode(response)
}
