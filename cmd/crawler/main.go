package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/e-commerce/platform/internal/common/config"
	"github.com/e-commerce/platform/internal/common/db"
	"github.com/e-commerce/platform/internal/common/messaging"
	"github.com/e-commerce/platform/internal/crawler"
)

func main() {
	// Create context that listens for termination signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen for termination signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("Received signal: %v", sig)
		cancel()
	}()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database connection
	database, err := db.NewPostgresDB(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Migrate database schema
	if err := database.MigrateSchema(); err != nil {
		log.Fatalf("Failed to migrate database schema: %v", err)
	}

	// Initialize Kafka client
	kafkaClient := messaging.NewKafkaClient(&cfg.Kafka)

	// Initialize crawler service
	crawlerService := crawler.NewCrawlerService(database, kafkaClient, cfg)

	// Start crawler service
	if err := crawlerService.Start(ctx); err != nil {
		log.Fatalf("Failed to start crawler service: %v", err)
	}

	// Initialize API server
	apiServer := crawler.NewAPI(database, cfg, crawlerService)

	// Start API server
	log.Printf("Starting crawler API server on port %d", cfg.Server.Port)
	if err := apiServer.Start(ctx); err != nil {
		log.Fatalf("API server error: %v", err)
	}

	// Wait for context cancellation
	<-ctx.Done()
	log.Println("Shutting down crawler service...")
}