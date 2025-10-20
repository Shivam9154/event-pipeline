package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// MessagesProcessed tracks total messages processed
	MessagesProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "events_processed_total",
			Help: "Total number of events processed",
		},
		[]string{"event_type", "status"},
	)

	// MessagesProcessedPerSecond tracks processing rate
	MessagesProcessedPerSecond = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "events_processed_per_second",
			Help: "Number of events processed per second",
		},
		[]string{"event_type"},
	)

	// DLQCount tracks dead letter queue entries
	DLQCount = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "dlq_entries_total",
			Help: "Total number of entries in dead letter queue",
		},
	)

	// DBLatency tracks database operation latency
	DBLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_operation_duration_seconds",
			Help:    "Database operation latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	// KafkaProduceLatency tracks Kafka produce latency
	KafkaProduceLatency = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "kafka_produce_duration_seconds",
			Help:    "Kafka produce latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	// KafkaConsumeLatency tracks Kafka consume latency
	KafkaConsumeLatency = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "kafka_consume_duration_seconds",
			Help:    "Kafka consume latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)
)
