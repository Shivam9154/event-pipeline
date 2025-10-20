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

type DemoData struct {
	UserID    string  `json:"userId"`
	OrderID   string  `json:"orderId"`
	PaymentID string  `json:"paymentId"`
	UserEmail string  `json:"userEmail"`
	Amount    float64 `json:"amount"`
}

func main() {
	// Load demo data
	data, err := os.ReadFile("demo-data.json")
	if err != nil {
		log.Fatalf("Failed to read demo data: %v", err)
	}

	var demo DemoData
	if err := json.Unmarshal(data, &demo); err != nil {
		log.Fatalf("Failed to parse demo data: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": cfg.Kafka.Brokers,
		"client.id":         "demo-producer",
	})
	if err != nil {
		log.Fatalf("Failed to create producer: %v", err)
	}
	defer p.Close()

	// Event 1: User Created
	fmt.Println("  ✓ Event 1: UserCreated - Alice Johnson registered")
	user := models.UserCreated{
		BaseEvent: models.BaseEvent{
			EventID:   uuid.New().String(),
			EventType: models.UserCreatedEvent,
			Timestamp: time.Now(),
		},
		UserID:    demo.UserID,
		Email:     demo.UserEmail,
		FirstName: "Alice",
		LastName:  "Johnson",
		CreatedAt: time.Now(),
	}
	sendEvent(p, cfg.Kafka.Topic, user)
	time.Sleep(500 * time.Millisecond)

	// Event 2: Order Placed
	fmt.Println("  ✓ Event 2: OrderPlaced - Laptop order $1,299.99")
	order := models.OrderPlaced{
		BaseEvent: models.BaseEvent{
			EventID:   uuid.New().String(),
			EventType: models.OrderPlacedEvent,
			Timestamp: time.Now(),
		},
		OrderID:     demo.OrderID,
		UserID:      demo.UserID,
		TotalAmount: demo.Amount,
		Currency:    "USD",
		Items: []models.OrderItem{
			{SKU: "LAPTOP-PRO-15", Quantity: 1, Price: 1299.99},
		},
		PlacedAt: time.Now(),
	}
	sendEvent(p, cfg.Kafka.Topic, order)
	time.Sleep(500 * time.Millisecond)

	// Event 3: Payment Settled
	fmt.Println("  ✓ Event 3: PaymentSettled - Payment completed")
	payment := models.PaymentSettled{
		BaseEvent: models.BaseEvent{
			EventID:   uuid.New().String(),
			EventType: models.PaymentSettledEvent,
			Timestamp: time.Now(),
		},
		PaymentID:     demo.PaymentID,
		OrderID:       demo.OrderID,
		Amount:        demo.Amount,
		Currency:      "USD",
		PaymentMethod: "credit_card",
		Status:        "completed",
		SettledAt:     time.Now(),
	}
	sendEvent(p, cfg.Kafka.Topic, payment)
	time.Sleep(500 * time.Millisecond)

	// Event 4: Inventory Adjusted
	fmt.Println("  ✓ Event 4: InventoryAdjusted - Stock reduced for shipment")
	inventory := models.InventoryAdjusted{
		BaseEvent: models.BaseEvent{
			EventID:   uuid.New().String(),
			EventType: models.InventoryAdjustedEvent,
			Timestamp: time.Now(),
		},
		SKU:            "LAPTOP-PRO-15",
		Quantity:       -1,
		AdjustmentType: "order_fulfilled",
		Reason:         fmt.Sprintf("Order %s shipped", demo.OrderID),
		AdjustedAt:     time.Now(),
	}
	sendEvent(p, cfg.Kafka.Topic, inventory)

	p.Flush(3000)
	fmt.Println("\n✅ All events published successfully!")
	fmt.Printf("\n📋 Demo IDs:\n")
	fmt.Printf("  User ID:    %s\n", demo.UserID)
	fmt.Printf("  Order ID:   %s\n", demo.OrderID)
	fmt.Printf("  Payment ID: %s\n", demo.PaymentID)
}

func sendEvent(p *kafka.Producer, topic string, event interface{}) {
	data, _ := json.Marshal(event)
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
	}

	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Key:            []byte(key),
		Value:          data,
	}
	p.Produce(msg, nil)
}
