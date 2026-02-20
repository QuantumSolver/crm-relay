package relayserver

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/QuantumSolver/crm-relay/internal/auth"
	"github.com/QuantumSolver/crm-relay/internal/models"
	"github.com/QuantumSolver/crm-relay/internal/storage"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for the relay server
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

	// Extract platform from URL path
	platform := ""
	if strings.HasPrefix(r.URL.Path, "/webhook/") {
		platform = strings.TrimPrefix(r.URL.Path, "/webhook/")
	}

	// Validate API key
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		sendErrorResponse(w, http.StatusUnauthorized, models.NewRelayError(
			models.ErrCodeAuthentication,
			"missing API key",
			nil,
		))
		return
	}

	// If platform is specified, validate API key against platform
	if platform != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		storedKey, err := h.redisClient.GetAPIKeyByValue(ctx, apiKey)
		if err != nil || !storedKey.IsActive || storedKey.Platform != platform {
			sendErrorResponse(w, http.StatusUnauthorized, models.NewRelayError(
				models.ErrCodeAuthentication,
				"invalid API key for platform",
				nil,
			))
			return
		}
	} else if apiKey != h.config.APIKey {
		// Fallback to legacy API key validation
		sendErrorResponse(w, http.StatusUnauthorized, models.NewRelayError(
			models.ErrCodeAuthentication,
			"invalid API key",
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

	// Get endpoint configuration if platform is specified
	var endpointID string
	var httpMethod string

	if platform != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		endpoint, err := h.redisClient.GetEndpointByPath(ctx, "/webhook/"+platform)
		if err == nil {
			endpointID = endpoint.ID
			httpMethod = endpoint.HTTPMethod
			// Target endpoint will be set by the client based on routing metadata
		}
	}

	// Create webhook
	webhook := &models.Webhook{
		ID:         uuid.New().String(),
		Headers:    headers,
		Body:       body,
		Timestamp:  time.Now(),
		Signature:  r.Header.Get("X-Hub-Signature"),
		Platform:   platform,
		EndpointID: endpointID,
		HTTPMethod: httpMethod,
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
		"platform":   platform,
		"timestamp":  webhook.Timestamp,
	})

	log.Printf("Webhook received and queued: ID=%s, MessageID=%s, Platform=%s, Latency=%dms", webhook.ID, messageID, platform, latency)
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
		"status":    "healthy",
		"timestamp": time.Now(),
		"redis": map[string]interface{}{
			"status":      redisStatus,
			"queue_depth": queueDepth,
		},
		"metrics": map[string]interface{}{
			"webhooks_received":  atomic.LoadInt64(&h.metrics.WebhooksReceived),
			"webhooks_processed": atomic.LoadInt64(&h.metrics.WebhooksProcessed),
			"webhooks_failed":    atomic.LoadInt64(&h.metrics.WebhooksFailed),
			"webhooks_retried":   atomic.LoadInt64(&h.metrics.WebhooksRetried),
			"average_latency_ms": atomic.LoadInt64(&h.metrics.AverageLatency),
			"last_webhook_time":  h.metrics.LastWebhookTime,
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

// API Key management endpoints

// HandleListAPIKeys handles requests to list all API keys
func (h *Handler) HandleListAPIKeys(w http.ResponseWriter, r *http.Request) {
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

	apiKeys, err := h.redisClient.ListAPIKeys(ctx)
	if err != nil {
		log.Printf("Failed to list API keys: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, err.(*models.RelayError))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"api_keys": apiKeys,
	})
}

// HandleCreateAPIKey handles requests to create a new API key
func (h *Handler) HandleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorResponse(w, http.StatusMethodNotAllowed, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"method not allowed",
			nil,
		))
		return
	}

	var req struct {
		Name     string `json:"name"`
		Platform string `json:"platform"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, http.StatusBadRequest, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"invalid request body",
			err,
		))
		return
	}

	// Generate API key
	key, err := auth.GenerateAPIKey()
	if err != nil {
		log.Printf("Failed to generate API key: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"failed to generate API key",
			err,
		))
		return
	}

	// Generate ID
	id, err := auth.GenerateID()
	if err != nil {
		log.Printf("Failed to generate ID: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"failed to generate ID",
			err,
		))
		return
	}

	apiKey := &models.APIKey{
		ID:        id,
		Name:      req.Name,
		Key:       key,
		Platform:  req.Platform,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		IsActive:  true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.redisClient.CreateAPIKey(ctx, apiKey); err != nil {
		log.Printf("Failed to create API key: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, err.(*models.RelayError))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(apiKey)

	log.Printf("API key created: %s for platform %s", apiKey.Name, apiKey.Platform)
}

// HandleUpdateAPIKey handles requests to update an API key
func (h *Handler) HandleUpdateAPIKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		sendErrorResponse(w, http.StatusMethodNotAllowed, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"method not allowed",
			nil,
		))
		return
	}

	var req struct {
		Name     string `json:"name"`
		IsActive *bool  `json:"is_active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, http.StatusBadRequest, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"invalid request body",
			err,
		))
		return
	}

	// Extract ID from URL path
	id := r.URL.Path[len("/api/keys/"):]
	if id == "" {
		sendErrorResponse(w, http.StatusBadRequest, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"missing API key ID",
			nil,
		))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	apiKey, err := h.redisClient.GetAPIKey(ctx, id)
	if err != nil {
		sendErrorResponse(w, http.StatusNotFound, err.(*models.RelayError))
		return
	}

	// Update fields
	if req.Name != "" {
		apiKey.Name = req.Name
	}
	if req.IsActive != nil {
		apiKey.IsActive = *req.IsActive
	}

	if err := h.redisClient.UpdateAPIKey(ctx, apiKey); err != nil {
		log.Printf("Failed to update API key: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, err.(*models.RelayError))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(apiKey)

	log.Printf("API key updated: %s", apiKey.ID)
}

// HandleDeleteAPIKey handles requests to delete an API key
func (h *Handler) HandleDeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		sendErrorResponse(w, http.StatusMethodNotAllowed, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"method not allowed",
			nil,
		))
		return
	}

	// Extract ID from URL path
	id := r.URL.Path[len("/api/keys/"):]
	if id == "" {
		sendErrorResponse(w, http.StatusBadRequest, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"missing API key ID",
			nil,
		))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.redisClient.DeleteAPIKey(ctx, id); err != nil {
		log.Printf("Failed to delete API key: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, err.(*models.RelayError))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "API key deleted successfully",
	})

	log.Printf("API key deleted: %s", id)
}

// Webhook Endpoint management endpoints

// HandleListEndpoints handles requests to list all webhook endpoints
func (h *Handler) HandleListEndpoints(w http.ResponseWriter, r *http.Request) {
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

	endpoints, err := h.redisClient.ListEndpoints(ctx)
	if err != nil {
		log.Printf("Failed to list endpoints: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, err.(*models.RelayError))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"endpoints": endpoints,
	})
}

// HandleCreateEndpoint handles requests to create a new webhook endpoint
func (h *Handler) HandleCreateEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorResponse(w, http.StatusMethodNotAllowed, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"method not allowed",
			nil,
		))
		return
	}

	var req struct {
		Platform   string            `json:"platform"`
		Path       string            `json:"path"`
		HTTPMethod string            `json:"http_method"`
		Headers    map[string]string `json:"headers"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, http.StatusBadRequest, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"invalid request body",
			err,
		))
		return
	}

	// Generate ID
	id, err := auth.GenerateID()
	if err != nil {
		log.Printf("Failed to generate ID: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"failed to generate ID",
			err,
		))
		return
	}

	endpoint := &models.WebhookEndpoint{
		ID:         id,
		Platform:   req.Platform,
		Path:       req.Path,
		HTTPMethod: req.HTTPMethod,
		Headers:    req.Headers,
		RetryConfig: models.RetryConfig{
			MaxRetries:      h.config.MaxRetries,
			RetryDelay:      h.config.RetryDelay,
			RetryMultiplier: h.config.RetryMultiplier,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.redisClient.CreateEndpoint(ctx, endpoint); err != nil {
		log.Printf("Failed to create endpoint: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, err.(*models.RelayError))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(endpoint)

	log.Printf("Webhook endpoint created: %s for platform %s", endpoint.Path, endpoint.Platform)
}

// HandleUpdateEndpoint handles requests to update a webhook endpoint
func (h *Handler) HandleUpdateEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		sendErrorResponse(w, http.StatusMethodNotAllowed, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"method not allowed",
			nil,
		))
		return
	}

	var req struct {
		Platform   *string            `json:"platform"`
		Path       *string            `json:"path"`
		HTTPMethod *string            `json:"http_method"`
		Headers    *map[string]string `json:"headers"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, http.StatusBadRequest, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"invalid request body",
			err,
		))
		return
	}

	// Extract ID from URL path
	id := r.URL.Path[len("/api/endpoints/"):]
	if id == "" {
		sendErrorResponse(w, http.StatusBadRequest, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"missing endpoint ID",
			nil,
		))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	endpoint, err := h.redisClient.GetEndpoint(ctx, id)
	if err != nil {
		sendErrorResponse(w, http.StatusNotFound, err.(*models.RelayError))
		return
	}

	// Update fields
	if req.Platform != nil {
		endpoint.Platform = *req.Platform
	}
	if req.Path != nil {
		endpoint.Path = *req.Path
	}
	if req.HTTPMethod != nil {
		endpoint.HTTPMethod = *req.HTTPMethod
	}
	if req.Headers != nil {
		endpoint.Headers = *req.Headers
	}

	if err := h.redisClient.UpdateEndpoint(ctx, endpoint); err != nil {
		log.Printf("Failed to update endpoint: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, err.(*models.RelayError))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(endpoint)

	log.Printf("Webhook endpoint updated: %s", endpoint.ID)
}

// HandleDeleteEndpoint handles requests to delete a webhook endpoint
func (h *Handler) HandleDeleteEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		sendErrorResponse(w, http.StatusMethodNotAllowed, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"method not allowed",
			nil,
		))
		return
	}

	// Extract ID from URL path
	id := r.URL.Path[len("/api/endpoints/"):]
	if id == "" {
		sendErrorResponse(w, http.StatusBadRequest, models.NewRelayError(
			models.ErrCodeInvalidRequest,
			"missing endpoint ID",
			nil,
		))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.redisClient.DeleteEndpoint(ctx, id); err != nil {
		log.Printf("Failed to delete endpoint: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, err.(*models.RelayError))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Endpoint deleted successfully",
	})

	log.Printf("Webhook endpoint deleted: %s", id)
}

// Metrics and monitoring endpoints

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
		"webhooks_failed":    atomic.LoadInt64(&h.metrics.WebhooksFailed),
		"webhooks_retried":   atomic.LoadInt64(&h.metrics.WebhooksRetried),
		"queue_depth":        queueDepth,
		"pending_messages":   pendingMessages,
		"average_latency_ms": atomic.LoadInt64(&h.metrics.AverageLatency),
		"last_webhook_time":  h.metrics.LastWebhookTime,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metrics)
}

// HandleGetQueueDepth handles requests to get queue depth
func (h *Handler) HandleGetQueueDepth(w http.ResponseWriter, r *http.Request) {
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

	queueDepth, err := h.redisClient.GetQueueDepth(ctx)
	if err != nil {
		log.Printf("Failed to get queue depth: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, err.(*models.RelayError))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"queue_depth": queueDepth,
	})
}

// HandleGetPendingMessages handles requests to get pending messages
func (h *Handler) HandleGetPendingMessages(w http.ResponseWriter, r *http.Request) {
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

	pendingMessages, err := h.redisClient.GetPendingMessages(ctx)
	if err != nil {
		log.Printf("Failed to get pending messages: %v", err)
		sendErrorResponse(w, http.StatusInternalServerError, err.(*models.RelayError))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pending_messages": pendingMessages,
	})
}
