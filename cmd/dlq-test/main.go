package main

import (
	"fmt"
	"log"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"event-pipeline/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

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
		{"Invalid JSON", `{"eventId": "test", "bad json`},
		{"Missing Fields", `{"eventId": "test", "eventType": "UserCreated"}`},
		{"Unknown Type", `{"eventId": "test", "eventType": "InvalidEvent", "timestamp": "2025-10-20T10:00:00Z"}`},
	}

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
}
