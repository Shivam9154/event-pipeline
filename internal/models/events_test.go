package models_test

import (
	"encoding/json"
	"testing"
	"time"

	"event-pipeline/internal/models"
	"github.com/google/uuid"
)

func TestUserCreatedEvent(t *testing.T) {
	event := models.UserCreated{
		BaseEvent: models.BaseEvent{
			EventID:   uuid.New().String(),
			EventType: models.UserCreatedEvent,
			Timestamp: time.Now(),
		},
		UserID:    uuid.New().String(),
		Email:     "test@example.com",
		FirstName: "John",
		LastName:  "Doe",
		CreatedAt: time.Now(),
	}

	// Test JSON marshaling
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	// Test JSON unmarshaling
	var decoded models.UserCreated
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	// Verify fields
	if decoded.UserID != event.UserID {
		t.Errorf("Expected UserID %s, got %s", event.UserID, decoded.UserID)
	}

	if decoded.Email != event.Email {
		t.Errorf("Expected Email %s, got %s", event.Email, decoded.Email)
	}

	// Test GetKey
	if event.GetKey() != event.UserID {
		t.Errorf("Expected key %s, got %s", event.UserID, event.GetKey())
	}
}

func TestOrderPlacedEvent(t *testing.T) {
	orderID := uuid.New().String()
	event := models.OrderPlaced{
		BaseEvent: models.BaseEvent{
			EventID:   uuid.New().String(),
			EventType: models.OrderPlacedEvent,
			Timestamp: time.Now(),
		},
		OrderID:     orderID,
		UserID:      uuid.New().String(),
		TotalAmount: 299.99,
		Currency:    "USD",
		Items: []models.OrderItem{
			{SKU: "LAPTOP-001", Quantity: 1, Price: 299.99},
		},
		PlacedAt: time.Now(),
	}

	// Test GetKey
	if event.GetKey() != orderID {
		t.Errorf("Expected key %s, got %s", orderID, event.GetKey())
	}

	// Test JSON marshaling
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	// Verify order items
	var decoded models.OrderPlaced
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if len(decoded.Items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(decoded.Items))
	}
}

func TestPaymentSettledEvent(t *testing.T) {
	orderID := uuid.New().String()
	event := models.PaymentSettled{
		BaseEvent: models.BaseEvent{
			EventID:   uuid.New().String(),
			EventType: models.PaymentSettledEvent,
			Timestamp: time.Now(),
		},
		PaymentID:     uuid.New().String(),
		OrderID:       orderID,
		Amount:        299.99,
		Currency:      "USD",
		PaymentMethod: "credit_card",
		Status:        "completed",
		SettledAt:     time.Now(),
	}

	// Test GetKey returns orderID
	if event.GetKey() != orderID {
		t.Errorf("Expected key %s, got %s", orderID, event.GetKey())
	}
}

func TestInventoryAdjustedEvent(t *testing.T) {
	sku := "LAPTOP-001"
	event := models.InventoryAdjusted{
		BaseEvent: models.BaseEvent{
			EventID:   uuid.New().String(),
			EventType: models.InventoryAdjustedEvent,
			Timestamp: time.Now(),
		},
		SKU:            sku,
		Quantity:       10,
		AdjustmentType: "add",
		Reason:         "restock",
		AdjustedAt:     time.Now(),
	}

	// Test GetKey returns SKU
	if event.GetKey() != sku {
		t.Errorf("Expected key %s, got %s", sku, event.GetKey())
	}
}
