package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yourusername/crm-relay/internal/config"
	relayclientpkg "github.com/yourusername/crm-relay/internal/relay-client"
	"github.com/yourusername/crm-relay/internal/storage"
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

	// Create forwarder
	forwarder := relayclientpkg.NewForwarder(cfg)
	defer forwarder.Close()

	log.Printf("Forwarder initialized: LocalWebhookURL=%s", cfg.LocalWebhookURL)

	// Create consumer
	consumer := relayclientpkg.NewConsumer(redisClient, cfg, forwarder)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
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

	// Wait a bit for cleanup
	time.Sleep(2 * time.Second)

	log.Println("Relay client stopped")
}
