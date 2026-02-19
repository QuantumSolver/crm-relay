package relayclient

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/yourusername/crm-relay/internal/models"
)

// Forwarder forwards webhooks to the local endpoint
type Forwarder struct {
	config      *models.Config
	httpClient  *http.Client
}

// NewForwarder creates a new forwarder
func NewForwarder(config *models.Config) *Forwarder {
	return &Forwarder{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// Forward forwards a webhook to the local endpoint
func (f *Forwarder) Forward(ctx context.Context, webhook *models.Webhook) error {
	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.config.LocalWebhookURL, bytes.NewReader(webhook.Body))
	if err != nil {
		return models.NewRelayError(
			models.ErrCodeWebhookForward,
			"failed to create request",
			err,
		)
	}

	// Copy headers
	for key, value := range webhook.Headers {
		// Skip hop-by-hop headers
		if key == "Connection" || key == "Keep-Alive" || key == "Proxy-Authenticate" ||
			key == "Proxy-Authorization" || key == "Te" || key == "Trailers" ||
			key == "Transfer-Encoding" || key == "Upgrade" {
			continue
		}
		req.Header.Set(key, value)
	}

	// Add relay headers
	req.Header.Set("X-Relay-Webhook-ID", webhook.ID)
	req.Header.Set("X-Relay-Timestamp", webhook.Timestamp.Format(time.RFC3339))
	if webhook.Signature != "" {
		req.Header.Set("X-Relay-Signature", webhook.Signature)
	}

	// Send request
	start := time.Now()
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return models.NewRelayError(
			models.ErrCodeWebhookForward,
			"failed to send request",
			err,
		)
	}
	defer resp.Body.Close()

	latency := time.Since(start)
	log.Printf("Webhook forwarded: ID=%s, Status=%d, Latency=%v", webhook.ID, resp.StatusCode, latency)

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v", err)
	}

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return models.NewRelayError(
			models.ErrCodeWebhookForward,
			"local webhook returned non-success status",
			nil,
		)
	}

	// Log response if available
	if len(body) > 0 {
		var respData map[string]interface{}
		if err := json.Unmarshal(body, &respData); err == nil {
			log.Printf("Local webhook response: %+v", respData)
		}
	}

	return nil
}

// Close closes the forwarder
func (f *Forwarder) Close() error {
	f.httpClient.CloseIdleConnections()
	return nil
}
