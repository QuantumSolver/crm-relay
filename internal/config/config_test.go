package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Set required environment variables
	os.Setenv("API_KEY", "test-api-key")
	os.Setenv("LOCAL_WEBHOOK_URL", "http://localhost:3000/webhook")
	defer func() {
		os.Unsetenv("API_KEY")
		os.Unsetenv("LOCAL_WEBHOOK_URL")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.APIKey != "test-api-key" {
		t.Errorf("Expected API_KEY to be 'test-api-key', got '%s'", cfg.APIKey)
	}

	if cfg.LocalWebhookURL != "http://localhost:3000/webhook" {
		t.Errorf("Expected LOCAL_WEBHOOK_URL to be 'http://localhost:3000/webhook', got '%s'", cfg.LocalWebhookURL)
	}

	// Check defaults
	if cfg.ServerPort != "8080" {
		t.Errorf("Expected default SERVER_PORT to be '8080', got '%s'", cfg.ServerPort)
	}

	if cfg.MaxRetries != 3 {
		t.Errorf("Expected default MAX_RETRIES to be 3, got %d", cfg.MaxRetries)
	}
}

func TestLoadMissingAPIKey(t *testing.T) {
	// Clear API key
	os.Setenv("API_KEY", "")
	os.Setenv("LOCAL_WEBHOOK_URL", "http://localhost:3000/webhook")
	defer func() {
		os.Unsetenv("API_KEY")
		os.Unsetenv("LOCAL_WEBHOOK_URL")
	}()

	_, err := Load()
	if err == nil {
		t.Error("Expected error when API_KEY is missing")
	}
}

func TestLoadInvalidRetryMultiplier(t *testing.T) {
	os.Setenv("API_KEY", "test-api-key")
	os.Setenv("LOCAL_WEBHOOK_URL", "http://localhost:3000/webhook")
	os.Setenv("RETRY_MULTIPLIER", "0")
	defer func() {
		os.Unsetenv("API_KEY")
		os.Unsetenv("LOCAL_WEBHOOK_URL")
		os.Unsetenv("RETRY_MULTIPLIER")
	}()

	_, err := Load()
	if err == nil {
		t.Error("Expected error when RETRY_MULTIPLIER is invalid")
	}
}
