package producer

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/sirupsen/logrus"
	"event-pipeline/internal/config"
	"event-pipeline/internal/logger"
	"event-pipeline/internal/metrics"
	"event-pipeline/internal/models"
)

// Producer wraps Kafka producer
type Producer struct {
	producer *kafka.Producer
	topic    string
}

// New creates a new Kafka producer
func New(cfg *config.KafkaConfig) (*Producer, error) {
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": cfg.Brokers,
		"client.id":         "event-producer",
		"acks":              "all",
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	logger.Log.Info("Successfully created Kafka producer")

	return &Producer{
		producer: p,
		topic:    cfg.Topic,
	}, nil
}

// Close closes the producer
func (p *Producer) Close() {
	p.producer.Flush(5000)
	p.producer.Close()
}

// PublishUserCreated publishes a UserCreated event
func (p *Producer) PublishUserCreated(event models.UserCreated) error {
	event.EventType = models.UserCreatedEvent
	return p.publish(event.GetKey(), event)
}

// PublishOrderPlaced publishes an OrderPlaced event
func (p *Producer) PublishOrderPlaced(event models.OrderPlaced) error {
	event.EventType = models.OrderPlacedEvent
	return p.publish(event.GetKey(), event)
}

// PublishPaymentSettled publishes a PaymentSettled event
func (p *Producer) PublishPaymentSettled(event models.PaymentSettled) error {
	event.EventType = models.PaymentSettledEvent
	return p.publish(event.GetKey(), event)
}

// PublishInventoryAdjusted publishes an InventoryAdjusted event
func (p *Producer) PublishInventoryAdjusted(event models.InventoryAdjusted) error {
	event.EventType = models.InventoryAdjustedEvent
	return p.publish(event.GetKey(), event)
}

// publish sends an event to Kafka
func (p *Producer) publish(key string, event interface{}) error {
	start := time.Now()
	defer func() {
		metrics.KafkaProduceLatency.Observe(time.Since(start).Seconds())
	}()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Extract eventID for logging
	var baseEvent models.BaseEvent
	if err := json.Unmarshal(data, &baseEvent); err != nil {
		return fmt.Errorf("failed to extract base event: %w", err)
	}

	deliveryChan := make(chan kafka.Event)
	
	err = p.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &p.topic, Partition: kafka.PartitionAny},
		Key:            []byte(key),
		Value:          data,
	}, deliveryChan)

	if err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	// Wait for delivery report
	e := <-deliveryChan
	m := e.(*kafka.Message)

	if m.TopicPartition.Error != nil {
		logger.WithEventID(baseEvent.EventID).WithFields(logrus.Fields{
			"eventType": baseEvent.EventType,
			"error":     m.TopicPartition.Error.Error(),
		}).Error("Failed to deliver message")
		return fmt.Errorf("delivery failed: %w", m.TopicPartition.Error)
	}

	logger.WithEventID(baseEvent.EventID).WithFields(logrus.Fields{
		"eventType": baseEvent.EventType,
		"partition": m.TopicPartition.Partition,
		"offset":    m.TopicPartition.Offset,
	}).Info("Message delivered successfully")

	return nil
}
