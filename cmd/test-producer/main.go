package main

import (
	"fmt"
	"strings"
	"time"

	"event-pipeline/internal/config"
	"event-pipeline/internal/logger"
	"event-pipeline/internal/models"
	"event-pipeline/internal/producer"

	"github.com/google/uuid"
)

func main() {
	logger.Log.Info("Starting Test Producer...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize producer
	prod, err := producer.New(&cfg.Kafka)
	if err != nil {
		logger.Log.Fatalf("Failed to create producer: %v", err)
	}
	defer prod.Close()

	fmt.Println("\n🚀 Generating test events...")
	fmt.Println(strings.Repeat("=", 50))

	// Create 3 users
	userIDs := make([]string, 3)
	for i := 0; i < 3; i++ {
		userID := uuid.New().String()
		userIDs[i] = userID

		event := models.UserCreated{
			BaseEvent: models.BaseEvent{
				EventID:   uuid.New().String(),
				Timestamp: time.Now(),
			},
			UserID:    userID,
			Email:     fmt.Sprintf("testuser%d@example.com", i+1),
			FirstName: fmt.Sprintf("TestUser%d", i+1),
			LastName:  "Test",
			CreatedAt: time.Now(),
		}

		if err := prod.PublishUserCreated(event); err != nil {
			logger.Log.Errorf("Failed to publish UserCreated: %v", err)
		} else {
			fmt.Printf("✅ Created User: %s (%s)\n", event.Email, userID)
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Create orders for each user
	orderIDs := make([]string, 3)
	for i, userID := range userIDs {
		orderID := uuid.New().String()
		orderIDs[i] = orderID

		event := models.OrderPlaced{
			BaseEvent: models.BaseEvent{
				EventID:   uuid.New().String(),
				Timestamp: time.Now(),
			},
			OrderID:     orderID,
			UserID:      userID,
			TotalAmount: float64((i + 1) * 100),
			Currency:    "USD",
			Items: []models.OrderItem{
				{SKU: fmt.Sprintf("LAPTOP-%03d", i+1), Quantity: i + 1, Price: 100.0},
			},
			PlacedAt: time.Now(),
		}

		if err := prod.PublishOrderPlaced(event); err != nil {
			logger.Log.Errorf("Failed to publish OrderPlaced: %v", err)
		} else {
			fmt.Printf("✅ Created Order: %s (User: %s, Amount: $%.2f)\n", orderID, userID, event.TotalAmount)
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Settle payments for orders
	for i, orderID := range orderIDs {
		event := models.PaymentSettled{
			BaseEvent: models.BaseEvent{
				EventID:   uuid.New().String(),
				Timestamp: time.Now(),
			},
			PaymentID:     uuid.New().String(),
			OrderID:       orderID,
			Amount:        float64((i + 1) * 100),
			Currency:      "USD",
			PaymentMethod: "credit_card",
			Status:        "completed",
			SettledAt:     time.Now(),
		}

		if err := prod.PublishPaymentSettled(event); err != nil {
			logger.Log.Errorf("Failed to publish PaymentSettled: %v", err)
		} else {
			fmt.Printf("✅ Settled Payment: %s (Order: %s, Amount: $%.2f)\n", event.PaymentID, orderID, event.Amount)
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Adjust inventory
	for i := 0; i < 5; i++ {
		event := models.InventoryAdjusted{
			BaseEvent: models.BaseEvent{
				EventID:   uuid.New().String(),
				Timestamp: time.Now(),
			},
			SKU:            fmt.Sprintf("LAPTOP-%03d", i+1),
			Quantity:       50,
			AdjustmentType: "add",
			Reason:         "initial_stock",
			AdjustedAt:     time.Now(),
		}

		if err := prod.PublishInventoryAdjusted(event); err != nil {
			logger.Log.Errorf("Failed to publish InventoryAdjusted: %v", err)
		} else {
			fmt.Printf("✅ Adjusted Inventory: %s (+%d)\n", event.SKU, event.Quantity)
		}
		time.Sleep(300 * time.Millisecond)
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("✅ All test events published successfully!")
	fmt.Println("\n📊 Test Data Created:")
	fmt.Println("  - 3 Users")
	fmt.Println("  - 3 Orders")
	fmt.Println("  - 3 Payments")
	fmt.Println("  - 5 Inventory Adjustments")
	fmt.Println("\n🔍 Test the API:")
	fmt.Printf("  curl http://localhost:8080/users/%s\n", userIDs[0])
	fmt.Printf("  curl http://localhost:8080/orders/%s\n", orderIDs[0])
	fmt.Println("  curl http://localhost:8080/metrics")
}
