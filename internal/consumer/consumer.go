package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/sirupsen/logrus"
	"event-pipeline/internal/config"
	"event-pipeline/internal/database"
	"event-pipeline/internal/dlq"
	"event-pipeline/internal/logger"
	"event-pipeline/internal/metrics"
	"event-pipeline/internal/models"
)

// Consumer wraps Kafka consumer
type Consumer struct {
	consumer *kafka.Consumer
	db       *database.DB
	dlq      *dlq.DLQ
	ctx      context.Context
	cancel   context.CancelFunc
}

// New creates a new Kafka consumer
func New(cfg *config.KafkaConfig, db *database.DB, dlqClient *dlq.DLQ) (*Consumer, error) {
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  cfg.Brokers,
		"group.id":           cfg.ConsumerGroup,
		"auto.offset.reset":  "earliest",
		"enable.auto.commit": false,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	if err := c.Subscribe(cfg.Topic, nil); err != nil {
		c.Close()
		return nil, fmt.Errorf("failed to subscribe to topic: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	logger.Log.WithFields(logrus.Fields{
		"topic":         cfg.Topic,
		"consumerGroup": cfg.ConsumerGroup,
	}).Info("Successfully created Kafka consumer")

	return &Consumer{
		consumer: c,
		db:       db,
		dlq:      dlqClient,
		ctx:      ctx,
		cancel:   cancel,
	}, nil
}

// Start starts consuming messages
func (c *Consumer) Start() {
	logger.Log.Info("Starting consumer...")
	
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	processedCount := make(map[string]int)

	go func() {
		for range ticker.C {
			for eventType, count := range processedCount {
				if count > 0 {
					metrics.MessagesProcessedPerSecond.WithLabelValues(eventType).Set(float64(count))
					processedCount[eventType] = 0
				}
			}
		}
	}()

	for {
		select {
		case <-c.ctx.Done():
			logger.Log.Info("Consumer stopping...")
			return
		default:
			msg, err := c.consumer.ReadMessage(100 * time.Millisecond)
			if err != nil {
				if err.(kafka.Error).Code() == kafka.ErrTimedOut {
					continue
				}
				logger.Log.Errorf("Consumer error: %v", err)
				continue
			}

			c.processMessage(msg, processedCount)
		}
	}
}

// Stop stops the consumer
func (c *Consumer) Stop() {
	c.cancel()
	c.consumer.Close()
}

// processMessage processes a single Kafka message
func (c *Consumer) processMessage(msg *kafka.Message, processedCount map[string]int) {
	start := time.Now()
	defer func() {
		metrics.KafkaConsumeLatency.Observe(time.Since(start).Seconds())
	}()

	// Parse base event to determine type
	var baseEvent models.BaseEvent
	if err := json.Unmarshal(msg.Value, &baseEvent); err != nil {
		logger.Log.Errorf("Failed to parse base event: %v", err)
		c.sendToDLQ(baseEvent.EventID, string(msg.Value), fmt.Sprintf("Failed to parse base event: %v", err))
		c.consumer.CommitMessage(msg)
		return
	}

	// Route by event type
	var err error
	switch baseEvent.EventType {
	case models.UserCreatedEvent:
		err = c.handleUserCreated(msg.Value, baseEvent.EventID)
	case models.OrderPlacedEvent:
		err = c.handleOrderPlaced(msg.Value, baseEvent.EventID)
	case models.PaymentSettledEvent:
		err = c.handlePaymentSettled(msg.Value, baseEvent.EventID)
	case models.InventoryAdjustedEvent:
		err = c.handleInventoryAdjusted(msg.Value, baseEvent.EventID)
	default:
		err = fmt.Errorf("unknown event type: %s", baseEvent.EventType)
	}

	if err != nil {
		logger.WithEventID(baseEvent.EventID).Errorf("Failed to process event: %v", err)
		c.sendToDLQ(baseEvent.EventID, string(msg.Value), err.Error())
		metrics.MessagesProcessed.WithLabelValues(string(baseEvent.EventType), "error").Inc()
	} else {
		metrics.MessagesProcessed.WithLabelValues(string(baseEvent.EventType), "success").Inc()
		processedCount[string(baseEvent.EventType)]++
	}

	// Commit offset
	if _, err := c.consumer.CommitMessage(msg); err != nil {
		logger.Log.Errorf("Failed to commit offset: %v", err)
	}
}

// handleUserCreated processes UserCreated event
func (c *Consumer) handleUserCreated(data []byte, eventID string) error {
	var event models.UserCreated
	if err := json.Unmarshal(data, &event); err != nil {
		return fmt.Errorf("failed to unmarshal UserCreated event: %w", err)
	}

	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	return c.db.UpsertUser(ctx, event)
}

// handleOrderPlaced processes OrderPlaced event
func (c *Consumer) handleOrderPlaced(data []byte, eventID string) error {
	var event models.OrderPlaced
	if err := json.Unmarshal(data, &event); err != nil {
		return fmt.Errorf("failed to unmarshal OrderPlaced event: %w", err)
	}

	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	return c.db.UpsertOrder(ctx, event)
}

// handlePaymentSettled processes PaymentSettled event
func (c *Consumer) handlePaymentSettled(data []byte, eventID string) error {
	var event models.PaymentSettled
	if err := json.Unmarshal(data, &event); err != nil {
		return fmt.Errorf("failed to unmarshal PaymentSettled event: %w", err)
	}

	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	return c.db.UpsertPayment(ctx, event)
}

// handleInventoryAdjusted processes InventoryAdjusted event
func (c *Consumer) handleInventoryAdjusted(data []byte, eventID string) error {
	var event models.InventoryAdjusted
	if err := json.Unmarshal(data, &event); err != nil {
		return fmt.Errorf("failed to unmarshal InventoryAdjusted event: %w", err)
	}

	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	return c.db.UpsertInventory(ctx, event)
}

// sendToDLQ sends a failed message to the dead letter queue
func (c *Consumer) sendToDLQ(eventID, originalData, errorMsg string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.dlq.Push(ctx, eventID, originalData, errorMsg); err != nil {
		logger.WithEventID(eventID).Errorf("Failed to push to DLQ: %v", err)
	}
}
