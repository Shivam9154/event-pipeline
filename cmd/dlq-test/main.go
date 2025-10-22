package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"event-pipeline/internal/config"
	"event-pipeline/internal/dlq"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to Redis DLQ to verify results
	dlqClient, err := dlq.New(&cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer dlqClient.Close()

	// Get initial DLQ count
	ctx := context.Background()
	initialCount, err := dlqClient.GetCount(ctx)
	if err != nil {
		log.Fatalf("Failed to get initial DLQ count: %v", err)
	}
	fmt.Printf("\n[*] Initial DLQ count: %d\n\n", initialCount)

	// Create Kafka producer
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": cfg.Kafka.Brokers,
		"client.id":         "dlq-test",
	})
	if err != nil {
		log.Fatalf("Failed to create producer: %v", err)
	}
	defer p.Close()

	testCases := []struct {
		name string
		data string
	}{
		{
			name: "Invalid JSON Syntax",
			data: `{"eventId": "test-1", "eventType": "UserCreated", "bad json without closing brace`,
		},
		{
			name: "Unknown Event Type",
			data: `{"eventId": "test-2", "eventType": "InvalidEventType", "timestamp": "2025-10-22T10:00:00Z"}`,
		},
		{
			name: "Malformed Timestamp",
			data: `{"eventId": "test-3", "eventType": "UserCreated", "timestamp": "not-a-valid-timestamp", "userId": "test", "email": "test@example.com"}`,
		},
	}

	fmt.Println("Sending malformed events to Kafka...")
	for _, tc := range testCases {
		msg := &kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &cfg.Kafka.Topic, Partition: kafka.PartitionAny},
			Key:            []byte("dlq-test"),
			Value:          []byte(tc.data),
		}
		p.Produce(msg, nil)
		fmt.Printf("  ❌ Sent: %s\n", tc.name)
	}

	p.Flush(3000)

	// Wait for consumer to process and push to DLQ
	fmt.Println("\n⏳ Waiting for consumer to process (5 seconds)...")
	time.Sleep(5 * time.Second)

	// Verify messages are in DLQ
	finalCount, err := dlqClient.GetCount(ctx)
	if err != nil {
		log.Fatalf("Failed to get final DLQ count: %v", err)
	}

	newEntries := finalCount - initialCount
	fmt.Printf("\n[*] Final DLQ count: %d\n", finalCount)
	fmt.Printf("[*] New entries: %d\n\n", newEntries)

	expectedEntries := int64(len(testCases))
	if newEntries >= expectedEntries {
		fmt.Printf("✅ SUCCESS: All %d malformed events were captured in DLQ!\n", expectedEntries)

		// Show the latest DLQ entries
		fmt.Println("\n[*] Latest DLQ entries:")
		entries, err := dlqClient.GetEntries(ctx, -int64(newEntries), -1)
		if err != nil {
			log.Printf("Warning: Failed to list DLQ entries: %v", err)
		} else {
			// Only show the new entries we just added
			startIdx := len(entries) - int(newEntries)
			if startIdx < 0 {
				startIdx = 0
			}
			for i, entry := range entries[startIdx:] {
				fmt.Printf("\n  Entry %d:\n", i+1)
				fmt.Printf("    Event ID: %s\n", entry.EventID)
				fmt.Printf("    Error: %s\n", entry.Error)
				fmt.Printf("    Timestamp: %s\n", entry.Timestamp.Format(time.RFC3339))
				// Show first 100 chars of original data
				dataPreview := entry.OriginalData
				if len(dataPreview) > 100 {
					dataPreview = dataPreview[:100] + "..."
				}
				fmt.Printf("    Original Data: %s\n", dataPreview)
			}
		}
	} else {
		fmt.Printf("❌ FAILED: Expected %d new DLQ entries, but got %d\n", expectedEntries, newEntries)
		fmt.Println("    Possible causes:")
		fmt.Println("    • Consumer is not running")
		fmt.Println("    • Consumer is not processing events")
		fmt.Println("    • Events are being successfully processed (no errors)")

		// Still show what we got
		if newEntries > 0 {
			fmt.Printf("\n[*] Entries that were added:\n")
			entries, err := dlqClient.GetEntries(ctx, -int64(newEntries), -1)
			if err != nil {
				log.Printf("Failed to list DLQ entries: %v", err)
			} else {
				for i, entry := range entries {
					fmt.Printf("  %d. Event ID: %s, Error: %s\n", i+1, entry.EventID, entry.Error)
				}
			}
		}
	}

	fmt.Println()
}
