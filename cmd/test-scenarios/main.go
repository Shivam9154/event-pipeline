package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"

	"event-pipeline/internal/config"
	"event-pipeline/internal/models"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create producer
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": cfg.Kafka.Brokers,
		"client.id":         "test-scenarios",
	})
	if err != nil {
		log.Fatalf("Failed to create producer: %v", err)
	}
	defer p.Close()

	// Run test scenarios
	fmt.Println("\n=== SCENARIO 1: Idempotency Test (Duplicate Events) ===")
	testIdempotency(p, cfg.Kafka.Topic)

	fmt.Println("\n=== SCENARIO 2: Edge Cases (Boundaries, Special Characters) ===")
	testEdgeCases(p, cfg.Kafka.Topic)

	fmt.Println("\n=== SCENARIO 3: Invalid/Malformed JSON (DLQ Test) ===")
	testMalformedJSON(p, cfg.Kafka.Topic)

	fmt.Println("\n=== SCENARIO 4: Concurrent Burst (20 events) ===")
	testConcurrentBurst(p, cfg.Kafka.Topic)

	fmt.Println("\n=== SCENARIO 5: Large Payload Test ===")
	testLargePayload(p, cfg.Kafka.Topic)

	// Wait for delivery reports
	fmt.Println("\n⏳ Waiting for all messages to be delivered...")
	p.Flush(5000)

	fmt.Println("\n✅ All test scenarios executed!")
	fmt.Println("\nNext Steps:")
	fmt.Println("1. Wait 10 seconds for consumer to process")
	fmt.Println("2. Check metrics: curl http://localhost:8080/metrics")
	fmt.Println("3. Query database to verify idempotency")
	fmt.Println("4. Check Redis DLQ: docker exec -it redis redis-cli LLEN dlq:events")
}

// SCENARIO 1: Idempotency - Send same event multiple times
func testIdempotency(p *kafka.Producer, topic string) {
	userID := uuid.New().String()

	// Create identical user event
	event := models.UserCreated{
		BaseEvent: models.BaseEvent{
			EventID:   uuid.New().String(),
			EventType: models.UserCreatedEvent,
			Timestamp: time.Now(),
		},
		UserID:    userID,
		Email:     "idempotency-test@example.com",
		FirstName: "Idempotent",
		LastName:  "User",
		CreatedAt: time.Now(),
	}

	// Send the SAME event 5 times (should only create 1 row in DB)
	for i := 1; i <= 5; i++ {
		data, _ := json.Marshal(event)
		msg := &kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Key:            []byte(event.GetKey()),
			Value:          data,
		}
		p.Produce(msg, nil)
		fmt.Printf("  📤 Attempt %d: Sent UserCreated (UserID: %s)\n", i, userID)
	}

	fmt.Printf("  ✅ Sent same event 5 times - DB should have only 1 row for user %s\n", userID)
}

// SCENARIO 2: Edge Cases
func testEdgeCases(p *kafka.Producer, topic string) {
	// Test 1: Empty/minimal strings
	user1 := models.UserCreated{
		BaseEvent: models.BaseEvent{
			EventID:   uuid.New().String(),
			EventType: models.UserCreatedEvent,
			Timestamp: time.Now(),
		},
		UserID:    uuid.New().String(),
		Email:     "a@b.c", // Minimal valid email
		FirstName: "X",     // Single character
		LastName:  "Y",
		CreatedAt: time.Now(),
	}
	sendEvent(p, topic, user1, "Edge: Minimal strings")

	// Test 2: Unicode and special characters
	user2 := models.UserCreated{
		BaseEvent: models.BaseEvent{
			EventID:   uuid.New().String(),
			EventType: models.UserCreatedEvent,
			Timestamp: time.Now(),
		},
		UserID:    uuid.New().String(),
		Email:     "unicode.测试@example.com",
		FirstName: "José-François",
		LastName:  "O'Brien-Smith 李明",
		CreatedAt: time.Now(),
	}
	sendEvent(p, topic, user2, "Edge: Unicode & special chars")

	// Test 3: Very long strings
	user3 := models.UserCreated{
		BaseEvent: models.BaseEvent{
			EventID:   uuid.New().String(),
			EventType: models.UserCreatedEvent,
			Timestamp: time.Now(),
		},
		UserID:    uuid.New().String(),
		Email:     strings.Repeat("very-long-email-", 10) + "@example.com",
		FirstName: strings.Repeat("LongFirstName", 20),
		LastName:  strings.Repeat("LongLastName", 20),
		CreatedAt: time.Now(),
	}
	sendEvent(p, topic, user3, "Edge: Very long strings")

	// Test 4: Large amounts (decimal precision)
	order := models.OrderPlaced{
		BaseEvent: models.BaseEvent{
			EventID:   uuid.New().String(),
			EventType: models.OrderPlacedEvent,
			Timestamp: time.Now(),
		},
		OrderID:     uuid.New().String(),
		UserID:      user1.UserID,
		TotalAmount: 9999999.99, // Max realistic value
		Currency:    "USD",
		Items:       []models.OrderItem{},
		PlacedAt:    time.Now(),
	}
	sendEvent(p, topic, order, "Edge: Large amount")

	// Test 5: Negative inventory adjustment
	inv := models.InventoryAdjusted{
		BaseEvent: models.BaseEvent{
			EventID:   uuid.New().String(),
			EventType: models.InventoryAdjustedEvent,
			Timestamp: time.Now(),
		},
		SKU:            "EDGE-TEST-001",
		Quantity:       -50, // Negative adjustment
		AdjustmentType: "returned",
		Reason:         "customer return",
		AdjustedAt:     time.Now(),
	}
	sendEvent(p, topic, inv, "Edge: Negative quantity")
}

// SCENARIO 3: Malformed JSON (should go to DLQ)
func testMalformedJSON(p *kafka.Producer, topic string) {
	testCases := []struct {
		name    string
		payload string
	}{
		{
			name:    "Invalid JSON syntax",
			payload: `{"eventId": "invalid", "eventType": "UserCreated", "timestamp": }`, // Missing value
		},
		{
			name:    "Missing required fields",
			payload: `{"eventId": "test-123", "eventType": "UserCreated"}`, // Missing userId, email
		},
		{
			name:    "Unknown event type",
			payload: `{"eventId": "test-456", "eventType": "UnknownEvent", "timestamp": "2025-10-20T10:00:00Z"}`,
		},
		{
			name:    "Empty payload",
			payload: `{}`,
		},
		{
			name:    "Non-JSON garbage",
			payload: `This is not JSON at all! 🚨`,
		},
	}

	for _, tc := range testCases {
		msg := &kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Key:            []byte("dlq-test"),
			Value:          []byte(tc.payload),
		}
		p.Produce(msg, nil)
		fmt.Printf("  📤 Sent %s\n", tc.name)
	}

	fmt.Printf("  ✅ Sent 5 invalid messages - should appear in Redis DLQ\n")
}

// SCENARIO 4: Concurrent burst
func testConcurrentBurst(p *kafka.Producer, topic string) {
	// Send 20 events rapidly
	for i := 1; i <= 20; i++ {
		event := models.UserCreated{
			BaseEvent: models.BaseEvent{
				EventID:   uuid.New().String(),
				EventType: models.UserCreatedEvent,
				Timestamp: time.Now(),
			},
			UserID:    uuid.New().String(),
			Email:     fmt.Sprintf("burst-user-%d@example.com", i),
			FirstName: fmt.Sprintf("Burst%d", i),
			LastName:  "Test",
			CreatedAt: time.Now(),
		}
		sendEvent(p, topic, event, "")
	}

	fmt.Printf("  ✅ Sent 20 events in rapid succession\n")
}

// SCENARIO 5: Large payload
func testLargePayload(p *kafka.Producer, topic string) {
	// Create order with many items
	items := make([]models.OrderItem, 50)
	for i := 0; i < 50; i++ {
		items[i] = models.OrderItem{
			SKU:      fmt.Sprintf("ITEM-%d", i),
			Quantity: i + 1,
			Price:    float64(i) * 10.50,
		}
	}

	order := models.OrderPlaced{
		BaseEvent: models.BaseEvent{
			EventID:   uuid.New().String(),
			EventType: models.OrderPlacedEvent,
			Timestamp: time.Now(),
		},
		OrderID:     uuid.New().String(),
		UserID:      uuid.New().String(),
		TotalAmount: 12345.67,
		Currency:    "USD",
		Items:       items,
		PlacedAt:    time.Now(),
	}

	sendEvent(p, topic, order, "Large payload test (50 items)")
}

// Helper function
func sendEvent(p *kafka.Producer, topic string, event interface{}, description string) {
	data, err := json.Marshal(event)
	if err != nil {
		fmt.Printf("  ❌ Failed to marshal: %v\n", err)
		return
	}

	var key string
	switch e := event.(type) {
	case models.UserCreated:
		key = e.GetKey()
	case models.OrderPlaced:
		key = e.GetKey()
	case models.PaymentSettled:
		key = e.GetKey()
	case models.InventoryAdjusted:
		key = e.GetKey()
	default:
		key = "unknown"
	}

	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Key:            []byte(key),
		Value:          data,
	}

	// Use delivery channel for feedback
	deliveryChan := make(chan kafka.Event, 1)
	err = p.Produce(msg, deliveryChan)
	if err != nil {
		fmt.Printf("  ❌ Failed to produce: %v\n", err)
		return
	}

	// Wait for delivery report (non-blocking with timeout)
	select {
	case e := <-deliveryChan:
		m := e.(*kafka.Message)
		if m.TopicPartition.Error != nil {
			fmt.Printf("  ❌ Delivery failed: %v\n", m.TopicPartition.Error)
		} else if description != "" {
			fmt.Printf("  ✅ %s\n", description)
		}
	case <-time.After(100 * time.Millisecond):
		// Timeout, continue
	}
}
