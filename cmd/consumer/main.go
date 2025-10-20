package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"event-pipeline/internal/api"
	"event-pipeline/internal/config"
	"event-pipeline/internal/consumer"
	"event-pipeline/internal/database"
	"event-pipeline/internal/dlq"
	"event-pipeline/internal/logger"
)

func main() {
	logger.Log.Info("Starting Event Pipeline Consumer...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	db, err := database.New(&cfg.MSSQL)
	if err != nil {
		logger.Log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize DLQ
	dlqClient, err := dlq.New(&cfg.Redis)
	if err != nil {
		logger.Log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer dlqClient.Close()

	// Initialize consumer
	kafkaConsumer, err := consumer.New(&cfg.Kafka, db, dlqClient)
	if err != nil {
		logger.Log.Fatalf("Failed to create consumer: %v", err)
	}
	defer kafkaConsumer.Stop()

	// Initialize API server
	apiServer := api.New(&cfg.API, db)

	// Start consumer in goroutine
	go kafkaConsumer.Start()

	// Start API server in goroutine
	go func() {
		if err := apiServer.Start(); err != nil && err != context.Canceled {
			logger.Log.Fatalf("API server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Log.Info("Shutting down gracefully...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := apiServer.Stop(ctx); err != nil {
		logger.Log.Errorf("Error stopping API server: %v", err)
	}

	logger.Log.Info("Shutdown complete")
}
