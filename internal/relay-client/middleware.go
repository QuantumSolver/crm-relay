package relayclient

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/yourusername/crm-relay/internal/auth"
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

// JWTMiddleware validates JWT tokens
func JWTMiddleware(jwtService *auth.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip JWT for health check and login
			if r.URL.Path == "/health" || r.URL.Path == "/api/auth/login" {
				next.ServeHTTP(w, r)
				return
			}

			// Get token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				sendErrorResponse(w, http.StatusUnauthorized, models.NewRelayError(
					models.ErrCodeAuthentication,
					"missing authorization header",
					nil,
				))
				return
			}

			// Extract token from "Bearer <token>"
			if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
				sendErrorResponse(w, http.StatusUnauthorized, models.NewRelayError(
					models.ErrCodeAuthentication,
					"invalid authorization header format",
					nil,
				))
				return
			}

			tokenString := authHeader[7:]

			// Validate token
			claims, err := jwtService.ValidateToken(tokenString)
			if err != nil {
				sendErrorResponse(w, http.StatusUnauthorized, models.NewRelayError(
					models.ErrCodeAuthentication,
					"invalid token",
					err,
				))
				return
			}

			// Add user context to request
			ctx := context.WithValue(r.Context(), "user", claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CORSMiddleware adds CORS headers
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

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
