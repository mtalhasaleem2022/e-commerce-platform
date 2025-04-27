package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/e-commerce/platform/internal/analyzer"
	"github.com/e-commerce/platform/internal/common/config"
	"github.com/e-commerce/platform/internal/common/db"
	"github.com/e-commerce/platform/internal/common/messaging"
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

	// Initialize Kafka client
	kafkaClient := messaging.NewKafkaClient(&cfg.Kafka)

	// Initialize analyzer service
	analyzerService := analyzer.NewAnalyzerService(database, kafkaClient, cfg)

	// Start analyzer service
	if err := analyzerService.Start(ctx); err != nil {
		log.Fatalf("Failed to start analyzer service: %v", err)
	}

	// Initialize API server
	apiServer := analyzer.NewAPI(database, cfg, analyzerService)

	// Start API server
	log.Printf("Starting analyzer API server on port %d", cfg.Services.AnalyzerServicePort)
	if err := apiServer.Start(ctx); err != nil {
		log.Fatalf("API server error: %v", err)
	}

	// Wait for context cancellation
	<-ctx.Done()
	log.Println("Shutting down analyzer service...")
}