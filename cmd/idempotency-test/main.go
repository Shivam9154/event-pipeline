package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"event-pipeline/internal/config"
	"event-pipeline/internal/models"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go <user-id>")
	}

	userId := os.Args[1]

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": cfg.Kafka.Brokers,
		"client.id":         "idempotency-test",
	})
	if err != nil {
		log.Fatalf("Failed to create producer: %v", err)
	}
	defer p.Close()

	// Same event, sent 3 times
	for i := 1; i <= 3; i++ {
		user := models.UserCreated{
			BaseEvent: models.BaseEvent{
				EventID:   uuid.New().String(),
				EventType: models.UserCreatedEvent,
				Timestamp: time.Now(),
			},
			UserID:    userId,
			Email:     "duplicate.test@example.com",
			FirstName: "Duplicate",
			LastName:  "Test",
			CreatedAt: time.Now(),
		}

		data, _ := json.Marshal(user)
		msg := &kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &cfg.Kafka.Topic, Partition: kafka.PartitionAny},
			Key:            []byte(user.GetKey()),
			Value:          data,
		}
		p.Produce(msg, nil)
		fmt.Printf("  📤 Attempt %d: Sent duplicate event\n", i)
		time.Sleep(200 * time.Millisecond)
	}

	p.Flush(3000)
	fmt.Println("\n✅ Sent 3 duplicate events")
}
