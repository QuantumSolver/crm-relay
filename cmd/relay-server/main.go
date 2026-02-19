package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/QuantumSolver/crm-relay/internal/auth"
	"github.com/QuantumSolver/crm-relay/internal/config"
	relayserverpkg "github.com/QuantumSolver/crm-relay/internal/relay-server"
	"github.com/QuantumSolver/crm-relay/internal/storage"
)

func main() {
	log.Println("Starting CRM Relay Server...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Configuration loaded: ServerPort=%s, RedisURL=%s, StreamName=%s",
		cfg.ServerPort, cfg.RedisURL, cfg.StreamName)

	// Initialize Redis client
	redisClient, err := storage.NewRedisClient(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize Redis client: %v", err)
	}
	defer redisClient.Close()

	log.Println("Redis client initialized successfully")

	// Initialize JWT service
	jwtService := auth.NewJWTService(cfg.JWTSecret, cfg.JWTExpiration)

	// Generate JWT secret if not set
	if cfg.JWTSecret == "" {
		bytes := make([]byte, 32)
		if _, err := rand.Read(bytes); err != nil {
			log.Fatalf("Failed to generate JWT secret: %v", err)
		}
		cfg.JWTSecret = base64.URLEncoding.EncodeToString(bytes)
		log.Printf("Generated JWT secret: %s", cfg.JWTSecret)
	}

	// Initialize default admin user
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	adminPassword := cfg.AdminPassword
	if adminPassword == "" {
		// Generate random password
		bytes := make([]byte, 16)
		if _, err := rand.Read(bytes); err != nil {
			log.Fatalf("Failed to generate admin password: %v", err)
		}
		adminPassword = base64.URLEncoding.EncodeToString(bytes)
		log.Printf("Generated admin password: %s", adminPassword)
	}

	adminPasswordHash, err := auth.HashPassword(adminPassword)
	if err != nil {
		log.Fatalf("Failed to hash admin password: %v", err)
	}

	if err := redisClient.InitializeDefaultUser(ctx, cfg.AdminUsername, adminPasswordHash); err != nil {
		log.Fatalf("Failed to initialize default admin user: %v", err)
	}

	log.Printf("Default admin user initialized: username=%s, password=%s", cfg.AdminUsername, adminPassword)

	// Create handler
	handler := relayserverpkg.NewHandler(redisClient, cfg, jwtService)

	// Set up HTTP server with enhanced ServeMux (Go 1.22+)
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("POST /webhook", handler.HandleWebhook)
	mux.HandleFunc("POST /webhook/", handler.HandleWebhook)
	mux.HandleFunc("GET /health", handler.HandleHealth)

	// Auth endpoints
	mux.HandleFunc("POST /api/auth/login", handler.HandleLogin)
	mux.HandleFunc("GET /api/auth/me", handler.HandleGetCurrentUser)

	// API key management endpoints
	mux.HandleFunc("GET /api/keys", handler.HandleListAPIKeys)
	mux.HandleFunc("POST /api/keys", handler.HandleCreateAPIKey)
	mux.HandleFunc("PUT /api/keys/", handler.HandleUpdateAPIKey)
	mux.HandleFunc("DELETE /api/keys/", handler.HandleDeleteAPIKey)

	// Webhook endpoint management endpoints
	mux.HandleFunc("GET /api/endpoints", handler.HandleListEndpoints)
	mux.HandleFunc("POST /api/endpoints", handler.HandleCreateEndpoint)
	mux.HandleFunc("PUT /api/endpoints/", handler.HandleUpdateEndpoint)
	mux.HandleFunc("DELETE /api/endpoints/", handler.HandleDeleteEndpoint)

	// Metrics endpoints
	mux.HandleFunc("GET /api/metrics", handler.HandleGetMetrics)
	mux.HandleFunc("GET /api/queue-depth", handler.HandleGetQueueDepth)
	mux.HandleFunc("GET /api/pending-messages", handler.HandleGetPendingMessages)

	// Serve static files for UI
	uiDir := http.Dir("web/server-ui/dist")
	fileServer := http.FileServer(uiDir)
	mux.Handle("GET /assets/", fileServer)
	mux.Handle("GET /index.html", fileServer)

	// SPA fallback - serve index.html for all non-API routes
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		// Skip API routes and webhook routes
		if r.URL.Path == "/health" ||
			r.URL.Path == "/webhook" ||
			r.URL.Path == "/api/" ||
			r.URL.Path == "/api/auth/login" ||
			r.URL.Path == "/api/auth/me" ||
			r.URL.Path == "/api/keys" ||
			r.URL.Path == "/api/endpoints" ||
			r.URL.Path == "/api/metrics" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "web/server-ui/dist/index.html")
	})

	// Apply middleware
	handlerChain := relayserverpkg.CORSMiddleware(
		relayserverpkg.RecoveryMiddleware(
			relayserverpkg.LoggingMiddleware(
				relayserverpkg.JWTMiddleware(jwtService)(
					relayserverpkg.AuthenticationMiddleware(cfg.APIKey)(mux),
				),
			),
		),
	)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      handlerChain,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server listening on port %s", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}
