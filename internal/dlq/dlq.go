package dlq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"event-pipeline/internal/config"
	"event-pipeline/internal/logger"
	"event-pipeline/internal/metrics"
	"event-pipeline/internal/models"
)

// DLQ handles dead letter queue operations
type DLQ struct {
	client *redis.Client
	key    string
}

// New creates a new DLQ instance
func New(cfg *config.RedisConfig) (*DLQ, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.GetRedisAddr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Log.Info("Successfully connected to Redis")

	return &DLQ{
		client: client,
		key:    cfg.DLQKey,
	}, nil
}

// Close closes the Redis connection
func (d *DLQ) Close() error {
	return d.client.Close()
}

// Push adds a failed message to the DLQ
func (d *DLQ) Push(ctx context.Context, eventID, originalData, errorMsg string) error {
	entry := models.DLQEntry{
		EventID:      eventID,
		OriginalData: originalData,
		Error:        errorMsg,
		Timestamp:    time.Now(),
		RetryCount:   0,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal DLQ entry: %w", err)
	}

	// Push to Redis list
	if err := d.client.RPush(ctx, d.key, data).Err(); err != nil {
		return fmt.Errorf("failed to push to DLQ: %w", err)
	}

	// Increment DLQ counter
	metrics.DLQCount.Inc()

	logger.WithEventID(eventID).WithFields(logrus.Fields{
		"error": errorMsg,
	}).Warn("Message pushed to DLQ")

	return nil
}

// GetCount returns the number of entries in the DLQ
func (d *DLQ) GetCount(ctx context.Context) (int64, error) {
	return d.client.LLen(ctx, d.key).Result()
}

// GetEntries retrieves entries from the DLQ
func (d *DLQ) GetEntries(ctx context.Context, start, stop int64) ([]models.DLQEntry, error) {
	results, err := d.client.LRange(ctx, d.key, start, stop).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get DLQ entries: %w", err)
	}

	entries := make([]models.DLQEntry, 0, len(results))
	for _, result := range results {
		var entry models.DLQEntry
		if err := json.Unmarshal([]byte(result), &entry); err != nil {
			logger.Log.Errorf("Failed to unmarshal DLQ entry: %v", err)
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}
