package relayserver

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/yourusername/crm-relay/internal/models"
)

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("Started %s %s", r.Method, r.URL.Path)

		next.ServeHTTP(w, r)

		log.Printf("Completed %s %s in %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// RecoveryMiddleware recovers from panics
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// AuthenticationMiddleware validates API key
func AuthenticationMiddleware(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip authentication for health check
			if r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}

			receivedKey := r.Header.Get("X-API-Key")
			if receivedKey == "" {
				sendErrorResponse(w, http.StatusUnauthorized, models.NewRelayError(
					models.ErrCodeAuthentication,
					"missing API key",
					nil,
				))
				return
			}

			if receivedKey != apiKey {
				sendErrorResponse(w, http.StatusUnauthorized, models.NewRelayError(
					models.ErrCodeAuthentication,
					"invalid API key",
					nil,
				))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CORSMiddleware adds CORS headers
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
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
