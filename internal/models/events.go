package models

import (
	"encoding/json"
	"time"
)

// EventType represents the type of event
type EventType string

const (
	UserCreatedEvent      EventType = "UserCreated"
	OrderPlacedEvent      EventType = "OrderPlaced"
	PaymentSettledEvent   EventType = "PaymentSettled"
	InventoryAdjustedEvent EventType = "InventoryAdjusted"
)

// BaseEvent contains common fields for all events
type BaseEvent struct {
	EventID   string    `json:"eventId"`
	EventType EventType `json:"eventType"`
	Timestamp time.Time `json:"timestamp"`
}

// UserCreated event
type UserCreated struct {
	BaseEvent
	UserID    string `json:"userId"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	CreatedAt time.Time `json:"createdAt"`
}

// GetKey returns the partition key for the event
func (e UserCreated) GetKey() string {
	return e.UserID
}

// OrderPlaced event
type OrderPlaced struct {
	BaseEvent
	OrderID     string    `json:"orderId"`
	UserID      string    `json:"userId"`
	TotalAmount float64   `json:"totalAmount"`
	Currency    string    `json:"currency"`
	Items       []OrderItem `json:"items"`
	PlacedAt    time.Time `json:"placedAt"`
}

// GetKey returns the partition key for the event
func (e OrderPlaced) GetKey() string {
	return e.OrderID
}

// OrderItem represents an item in an order
type OrderItem struct {
	SKU      string  `json:"sku"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
}

// PaymentSettled event
type PaymentSettled struct {
	BaseEvent
	PaymentID       string    `json:"paymentId"`
	OrderID         string    `json:"orderId"`
	Amount          float64   `json:"amount"`
	Currency        string    `json:"currency"`
	PaymentMethod   string    `json:"paymentMethod"`
	Status          string    `json:"status"`
	SettledAt       time.Time `json:"settledAt"`
}

// GetKey returns the partition key for the event
func (e PaymentSettled) GetKey() string {
	return e.OrderID
}

// InventoryAdjusted event
type InventoryAdjusted struct {
	BaseEvent
	SKU            string    `json:"sku"`
	Quantity       int       `json:"quantity"`
	AdjustmentType string    `json:"adjustmentType"` // "add" or "subtract"
	Reason         string    `json:"reason"`
	AdjustedAt     time.Time `json:"adjustedAt"`
}

// GetKey returns the partition key for the event
func (e InventoryAdjusted) GetKey() string {
	return e.SKU
}

// Event is a wrapper for all event types
type Event struct {
	Type    EventType       `json:"eventType"`
	Payload json.RawMessage `json:"payload"`
}

// DLQEntry represents an entry in the dead letter queue
type DLQEntry struct {
	EventID      string    `json:"eventId"`
	OriginalData string    `json:"originalData"`
	Error        string    `json:"error"`
	Timestamp    time.Time `json:"timestamp"`
	RetryCount   int       `json:"retryCount"`
}
