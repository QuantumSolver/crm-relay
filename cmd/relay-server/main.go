package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yourusername/crm-relay/internal/config"
	relayserverpkg "github.com/yourusername/crm-relay/internal/relay-server"
	"github.com/yourusername/crm-relay/internal/storage"
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

	// Create handler
	handler := relayserverpkg.NewHandler(redisClient, cfg)

	// Set up HTTP server with enhanced ServeMux (Go 1.22+)
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("POST /webhook", handler.HandleWebhook)
	mux.HandleFunc("GET /health", handler.HandleHealth)

	// Apply middleware
	handlerChain := relayserverpkg.CORSMiddleware(
		relayserverpkg.RecoveryMiddleware(
			relayserverpkg.LoggingMiddleware(
				relayserverpkg.AuthenticationMiddleware(cfg.APIKey)(mux),
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}
