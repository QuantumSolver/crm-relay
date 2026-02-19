package models

import (
	"errors"
	"testing"
	"time"
)

func TestRelayError(t *testing.T) {
	err := NewRelayError(ErrCodeInvalidRequest, "test error", errors.New("underlying error"))

	if err.Code != ErrCodeInvalidRequest {
		t.Errorf("Expected code to be '%s', got '%s'", ErrCodeInvalidRequest, err.Code)
	}

	if err.Message != "test error" {
		t.Errorf("Expected message to be 'test error', got '%s'", err.Message)
	}

	expectedError := "test error: underlying error"
	if err.Error() != expectedError {
		t.Errorf("Expected error string to be '%s', got '%s'", expectedError, err.Error())
	}

	if err.Unwrap() == nil {
		t.Error("Expected Unwrap to return the underlying error")
	}
}

func TestRelayErrorWithoutUnderlying(t *testing.T) {
	err := NewRelayError(ErrCodeInvalidRequest, "test error", nil)

	expectedError := "test error"
	if err.Error() != expectedError {
		t.Errorf("Expected error string to be '%s', got '%s'", expectedError, err.Error())
	}

	if err.Unwrap() != nil {
		t.Error("Expected Unwrap to return nil when there's no underlying error")
	}
}

func TestWebhook(t *testing.T) {
	webhook := Webhook{
		ID:        "test-id",
		Headers:   map[string]string{"Content-Type": "application/json"},
		Body:      []byte(`{"test": "data"}`),
		Timestamp: time.Now(),
		Signature: "test-signature",
	}

	if webhook.ID != "test-id" {
		t.Errorf("Expected ID to be 'test-id', got '%s'", webhook.ID)
	}

	if webhook.Signature != "test-signature" {
		t.Errorf("Expected Signature to be 'test-signature', got '%s'", webhook.Signature)
	}
}

func TestRelayMessage(t *testing.T) {
	webhook := Webhook{
		ID:        "test-id",
		Body:      []byte(`{"test": "data"}`),
		Timestamp: time.Now(),
	}

	message := RelayMessage{
		MessageID:  "msg-id",
		Webhook:    webhook,
		RetryCount: 0,
		CreatedAt:  time.Now(),
	}

	if message.MessageID != "msg-id" {
		t.Errorf("Expected MessageID to be 'msg-id', got '%s'", message.MessageID)
	}

	if message.RetryCount != 0 {
		t.Errorf("Expected RetryCount to be 0, got %d", message.RetryCount)
	}
}
