package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"event-pipeline/internal/config"
	"event-pipeline/internal/logger"
	"event-pipeline/internal/models"
	"event-pipeline/internal/producer"
)

func main() {
	logger.Log.Info("Starting Event Pipeline Producer...")

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

	// Interactive menu
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Println("\n=== Event Producer Menu ===")
		fmt.Println("1. Create User")
		fmt.Println("2. Place Order")
		fmt.Println("3. Settle Payment")
		fmt.Println("4. Adjust Inventory")
		fmt.Println("5. Generate Sample Events")
		fmt.Println("0. Exit")
		fmt.Print("\nSelect option: ")

		if !scanner.Scan() {
			break
		}

		choice := strings.TrimSpace(scanner.Text())

		switch choice {
		case "1":
			createUser(prod)
		case "2":
			placeOrder(prod)
		case "3":
			settlePayment(prod)
		case "4":
			adjustInventory(prod)
		case "5":
			generateSampleEvents(prod)
		case "0":
			logger.Log.Info("Exiting...")
			return
		default:
			fmt.Println("Invalid option")
		}
	}
}

func createUser(prod *producer.Producer) {
	userID := uuid.New().String()
	event := models.UserCreated{
		BaseEvent: models.BaseEvent{
			EventID:   uuid.New().String(),
			Timestamp: time.Now(),
		},
		UserID:    userID,
		Email:     fmt.Sprintf("user%s@example.com", userID[:8]),
		FirstName: "John",
		LastName:  "Doe",
		CreatedAt: time.Now(),
	}

	if err := prod.PublishUserCreated(event); err != nil {
		logger.Log.Errorf("Failed to publish UserCreated: %v", err)
		return
	}

	fmt.Printf("✓ User created: %s\n", userID)
}

func placeOrder(prod *producer.Producer) {
	orderID := uuid.New().String()
	userID := uuid.New().String()
	
	event := models.OrderPlaced{
		BaseEvent: models.BaseEvent{
			EventID:   uuid.New().String(),
			Timestamp: time.Now(),
		},
		OrderID:     orderID,
		UserID:      userID,
		TotalAmount: 299.99,
		Currency:    "USD",
		Items: []models.OrderItem{
			{SKU: "LAPTOP-001", Quantity: 1, Price: 299.99},
		},
		PlacedAt: time.Now(),
	}

	if err := prod.PublishOrderPlaced(event); err != nil {
		logger.Log.Errorf("Failed to publish OrderPlaced: %v", err)
		return
	}

	fmt.Printf("✓ Order placed: %s (User: %s)\n", orderID, userID)
}

func settlePayment(prod *producer.Producer) {
	paymentID := uuid.New().String()
	orderID := uuid.New().String()
	
	event := models.PaymentSettled{
		BaseEvent: models.BaseEvent{
			EventID:   uuid.New().String(),
			Timestamp: time.Now(),
		},
		PaymentID:     paymentID,
		OrderID:       orderID,
		Amount:        299.99,
		Currency:      "USD",
		PaymentMethod: "credit_card",
		Status:        "completed",
		SettledAt:     time.Now(),
	}

	if err := prod.PublishPaymentSettled(event); err != nil {
		logger.Log.Errorf("Failed to publish PaymentSettled: %v", err)
		return
	}

	fmt.Printf("✓ Payment settled: %s (Order: %s)\n", paymentID, orderID)
}

func adjustInventory(prod *producer.Producer) {
	sku := fmt.Sprintf("LAPTOP-%03d", time.Now().Unix()%1000)
	
	event := models.InventoryAdjusted{
		BaseEvent: models.BaseEvent{
			EventID:   uuid.New().String(),
			Timestamp: time.Now(),
		},
		SKU:            sku,
		Quantity:       10,
		AdjustmentType: "add",
		Reason:         "restock",
		AdjustedAt:     time.Now(),
	}

	if err := prod.PublishInventoryAdjusted(event); err != nil {
		logger.Log.Errorf("Failed to publish InventoryAdjusted: %v", err)
		return
	}

	fmt.Printf("✓ Inventory adjusted: %s (+10)\n", sku)
}

func generateSampleEvents(prod *producer.Producer) {
	logger.Log.Info("Generating sample events...")

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
			Email:     fmt.Sprintf("user%d@example.com", i+1),
			FirstName: fmt.Sprintf("User%d", i+1),
			LastName:  "Test",
			CreatedAt: time.Now(),
		}
		
		if err := prod.PublishUserCreated(event); err != nil {
			logger.Log.Errorf("Failed to publish UserCreated: %v", err)
		}
	}
	fmt.Printf("✓ Created 3 users\n")

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
			TotalAmount: float64((i+1) * 100),
			Currency:    "USD",
			Items: []models.OrderItem{
				{SKU: fmt.Sprintf("ITEM-%03d", i+1), Quantity: i + 1, Price: 100.0},
			},
			PlacedAt: time.Now(),
		}
		
		if err := prod.PublishOrderPlaced(event); err != nil {
			logger.Log.Errorf("Failed to publish OrderPlaced: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Printf("✓ Created 3 orders\n")

	// Settle payments for orders
	for i, orderID := range orderIDs {
		event := models.PaymentSettled{
			BaseEvent: models.BaseEvent{
				EventID:   uuid.New().String(),
				Timestamp: time.Now(),
			},
			PaymentID:     uuid.New().String(),
			OrderID:       orderID,
			Amount:        float64((i+1) * 100),
			Currency:      "USD",
			PaymentMethod: "credit_card",
			Status:        "completed",
			SettledAt:     time.Now(),
		}
		
		if err := prod.PublishPaymentSettled(event); err != nil {
			logger.Log.Errorf("Failed to publish PaymentSettled: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Printf("✓ Settled 3 payments\n")

	// Adjust inventory
	for i := 0; i < 5; i++ {
		event := models.InventoryAdjusted{
			BaseEvent: models.BaseEvent{
				EventID:   uuid.New().String(),
				Timestamp: time.Now(),
			},
			SKU:            fmt.Sprintf("ITEM-%03d", i+1),
			Quantity:       50,
			AdjustmentType: "add",
			Reason:         "initial_stock",
			AdjustedAt:     time.Now(),
		}
		
		if err := prod.PublishInventoryAdjusted(event); err != nil {
			logger.Log.Errorf("Failed to publish InventoryAdjusted: %v", err)
		}
	}
	fmt.Printf("✓ Adjusted inventory for 5 items\n")

	fmt.Println("\n✅ Sample events generated successfully!")
}
