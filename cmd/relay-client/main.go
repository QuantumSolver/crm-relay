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
	relayclientpkg "github.com/QuantumSolver/crm-relay/internal/relay-client"
	"github.com/QuantumSolver/crm-relay/internal/storage"
)

func main() {
	log.Println("Starting CRM Relay Client...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Configuration loaded: RedisURL=%s, StreamName=%s, LocalWebhookURL=%s",
		cfg.RedisURL, cfg.StreamName, cfg.LocalWebhookURL)

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

	// Create forwarder
	forwarder := relayclientpkg.NewForwarder(cfg)
	defer forwarder.Close()

	log.Printf("Forwarder initialized: LocalWebhookURL=%s", cfg.LocalWebhookURL)

	// Create consumer
	consumer := relayclientpkg.NewConsumer(redisClient, cfg, forwarder)

	// Create handler
	handler := relayclientpkg.NewHandler(redisClient, cfg, jwtService)

	// Set up HTTP server with enhanced ServeMux (Go 1.22+)
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})

	// Auth endpoints
	mux.HandleFunc("POST /api/auth/login", handler.HandleLogin)
	mux.HandleFunc("GET /api/auth/me", handler.HandleGetCurrentUser)

	// Configuration endpoints
	mux.HandleFunc("PUT /api/config/local-endpoint", handler.HandleUpdateLocalEndpoint)
	mux.HandleFunc("PUT /api/config/retry", handler.HandleUpdateRetryConfig)

	// DLQ endpoints
	mux.HandleFunc("GET /api/dlq", handler.HandleGetDLQMessages)
	mux.HandleFunc("POST /api/dlq/", handler.HandleReplayDLQMessage)
	mux.HandleFunc("DELETE /api/dlq/", handler.HandleDeleteDLQMessage)

	// Metrics endpoints
	mux.HandleFunc("GET /api/metrics", handler.HandleGetMetrics)

	// Serve static files for UI
	uiDir := http.Dir("web/client-ui/dist")
	fileServer := http.FileServer(uiDir)
	mux.Handle("GET /assets/", fileServer)
	mux.Handle("GET /index.html", fileServer)

	// SPA fallback - serve index.html for all non-API routes
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		// Skip API routes
		if r.URL.Path == "/health" ||
			r.URL.Path == "/api/" ||
			r.URL.Path == "/api/auth/login" ||
			r.URL.Path == "/api/auth/me" ||
			r.URL.Path == "/api/config/local-endpoint" ||
			r.URL.Path == "/api/config/retry" ||
			r.URL.Path == "/api/dlq" ||
			r.URL.Path == "/api/metrics" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "web/client-ui/dist/index.html")
	})

	// Apply middleware
	handlerChain := relayclientpkg.CORSMiddleware(
		relayclientpkg.RecoveryMiddleware(
			relayclientpkg.LoggingMiddleware(
				relayclientpkg.JWTMiddleware(jwtService)(mux),
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

	// Create context for graceful shutdown
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	// Start consumer in a goroutine
	go func() {
		consumer.Start(ctx)
	}()

	// Start metrics reporter
	go func() {
		ticker := time.NewTicker(time.Duration(cfg.HealthCheckInterval) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				metrics := consumer.GetMetrics()
				log.Printf("Metrics: Received=%d, Processed=%d, Failed=%d, Retried=%d",
					metrics.WebhooksReceived,
					metrics.WebhooksProcessed,
					metrics.WebhooksFailed,
					metrics.WebhooksRetried,
				)
			}
		}
	}()

	// Start HTTP server in a goroutine
	go func() {
		log.Printf("HTTP server listening on port %s", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	log.Println("Relay client is running...")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down relay client...")

	// Stop consumer
	consumer.Stop()

	// Cancel context
	cancel()

	// Shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Wait a bit for cleanup
	time.Sleep(2 * time.Second)

	log.Println("Relay client stopped")
}
