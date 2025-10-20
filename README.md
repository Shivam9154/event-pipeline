# Event Pipeline - Kafka Producer/Consumer System

A production-ready event-driven pipeline built with Go, Kafka, MS SQL, Redis, and Docker Compose. Implements idempotent message processing with dead letter queue (DLQ) support and comprehensive observability.

## 🎯 Features

- **4 Event Types**: UserCreated, OrderPlaced, PaymentSettled, InventoryAdjusted
- **Kafka Producer/Consumer**: Keyed messages with partitioning
- **Idempotent Processing**: Upsert operations with unique constraints
- **Dead Letter Queue**: Redis-based DLQ for failed messages
- **Read API**: RESTful endpoints for querying data
- **Metrics & Logging**: Prometheus metrics with structured JSON logs
- **Docker Compose**: One-command deployment

## 📋 Architecture

```
┌─────────────┐      ┌─────────┐      ┌──────────────┐      ┌─────────┐
│  Producer   │─────▶│  Kafka  │─────▶│   Consumer   │─────▶│ MS SQL  │
└─────────────┘      └─────────┘      └──────────────┘      └─────────┘
                                              │
                                              │ (on error)
                                              ▼
                                       ┌─────────────┐
                                       │ Redis (DLQ) │
                                       └─────────────┘
                                              
┌──────────────┐
│   Read API   │◀──────────────────────────────────────────▶ MS SQL
└──────────────┘
```

## 🚀 Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.21+ (for local development)
- Git

### 1. Clone and Setup

```bash
git clone <repository-url>
cd event-pipeline
cp .env.example .env
```

### 2. Run the Demo

**Easiest way to see the system in action:**

```powershell
# Windows
.\demo.ps1

# Linux/Mac
chmod +x demo.sh
./demo.sh
```

The demo script automatically:
- ✅ Checks all services are running
- ✅ Creates realistic e-commerce events (User → Order → Payment → Inventory)
- ✅ Tests idempotency (duplicate detection)
- ✅ Tests error handling (DLQ)
- ✅ Queries the API to verify results
- ✅ Shows metrics and database state

**See [DEMO_GUIDE.md](DEMO_GUIDE.md) for detailed demo documentation.**

### 3. Start All Services (Manual)

```bash
docker-compose up -d
```

This starts:
- Kafka (port 9092)
- Zookeeper (port 2181)
- MS SQL Server (port 1433)
- Redis (port 6379)
- Consumer/API Service (ports 8080, 9090)

### 3. Initialize Database

The database schema is automatically created on startup. To manually run:

```bash
docker exec -it mssql /opt/mssql-tools/bin/sqlcmd -S localhost -U sa -P 'YourStrong@Passw0rd' -d eventdb -i /schema.sql
```

### 4. Run Producer (Interactive)

**Option A: Using Docker**
```bash
docker-compose run --rm producer
```

**Option B: Local Development**
```bash
go run cmd/producer/main.go
```

The producer provides an interactive menu:
```
=== Event Producer Menu ===
1. Create User
2. Place Order
3. Settle Payment
4. Adjust Inventory
5. Generate Sample Events
0. Exit
```

Select option `5` to generate sample events for testing.

### 5. Test the API

```bash
# Health check
curl http://localhost:8080/health

# Get user with last 5 orders (replace USER_ID)
curl http://localhost:8080/users/YOUR_USER_ID

# Get order with payment status (replace ORDER_ID)
curl http://localhost:8080/orders/YOUR_ORDER_ID

# View metrics
curl http://localhost:8080/metrics
```

Or use the provided `api-tests.http` file with REST Client extension in VS Code.

## 📊 Event Types

### 1. UserCreated
```json
{
  "eventId": "uuid",
  "eventType": "UserCreated",
  "timestamp": "2025-10-20T10:00:00Z",
  "userId": "user-uuid",
  "email": "user@example.com",
  "firstName": "John",
  "lastName": "Doe",
  "createdAt": "2025-10-20T10:00:00Z"
}
```
**Key**: `userId`

### 2. OrderPlaced
```json
{
  "eventId": "uuid",
  "eventType": "OrderPlaced",
  "timestamp": "2025-10-20T10:00:00Z",
  "orderId": "order-uuid",
  "userId": "user-uuid",
  "totalAmount": 299.99,
  "currency": "USD",
  "items": [
    {"sku": "LAPTOP-001", "quantity": 1, "price": 299.99}
  ],
  "placedAt": "2025-10-20T10:00:00Z"
}
```
**Key**: `orderId`

### 3. PaymentSettled
```json
{
  "eventId": "uuid",
  "eventType": "PaymentSettled",
  "timestamp": "2025-10-20T10:00:00Z",
  "paymentId": "payment-uuid",
  "orderId": "order-uuid",
  "amount": 299.99,
  "currency": "USD",
  "paymentMethod": "credit_card",
  "status": "completed",
  "settledAt": "2025-10-20T10:00:00Z"
}
```
**Key**: `orderId`

### 4. InventoryAdjusted
```json
{
  "eventId": "uuid",
  "eventType": "InventoryAdjusted",
  "timestamp": "2025-10-20T10:00:00Z",
  "sku": "LAPTOP-001",
  "quantity": 10,
  "adjustmentType": "add",
  "reason": "restock",
  "adjustedAt": "2025-10-20T10:00:00Z"
}
```
**Key**: `sku`

## 🔍 API Endpoints

### GET /health
Health check endpoint
```bash
curl http://localhost:8080/health
```

### GET /users/{id}
Retrieve user with last 5 orders
```bash
curl http://localhost:8080/users/550e8400-e29b-41d4-a716-446655440000
```

**Response:**
```json
{
  "userId": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "firstName": "John",
  "lastName": "Doe",
  "createdAt": "2025-10-20T10:00:00Z",
  "updatedAt": "2025-10-20T10:00:00Z",
  "orders": [
    {
      "orderId": "order-uuid",
      "userId": "user-uuid",
      "totalAmount": 299.99,
      "currency": "USD",
      "placedAt": "2025-10-20T10:00:00Z",
      "updatedAt": "2025-10-20T10:00:00Z"
    }
  ]
}
```

### GET /orders/{id}
Retrieve order with payment status
```bash
curl http://localhost:8080/orders/650e8400-e29b-41d4-a716-446655440000
```

**Response:**
```json
{
  "orderId": "650e8400-e29b-41d4-a716-446655440000",
  "userId": "user-uuid",
  "totalAmount": 299.99,
  "currency": "USD",
  "placedAt": "2025-10-20T10:00:00Z",
  "updatedAt": "2025-10-20T10:00:00Z",
  "payment": {
    "paymentId": "payment-uuid",
    "amount": 299.99,
    "paymentMethod": "credit_card",
    "status": "completed",
    "settledAt": "2025-10-20T10:00:00Z"
  }
}
```

### GET /metrics
Prometheus metrics endpoint
```bash
curl http://localhost:8080/metrics
```

## 📈 Metrics & Monitoring

### Key Metrics

1. **events_processed_total** - Counter of processed events by type and status
2. **events_processed_per_second** - Gauge of processing rate by event type
3. **dlq_entries_total** - Counter of DLQ entries
4. **db_operation_duration_seconds** - Histogram of DB latency (p50, p95, p99)
5. **kafka_produce_duration_seconds** - Histogram of Kafka produce latency
6. **kafka_consume_duration_seconds** - Histogram of Kafka consume latency

### Viewing Metrics

```bash
# View all metrics
curl http://localhost:8080/metrics

# Filter specific metric
curl http://localhost:8080/metrics | grep events_processed_total
```

### Structured Logging

All logs are in JSON format with `eventId` correlation:

```json
{
  "level": "info",
  "msg": "Message delivered successfully",
  "eventId": "550e8400-e29b-41d4-a716-446655440000",
  "eventType": "UserCreated",
  "partition": 0,
  "offset": 123,
  "time": "2025-10-20T10:00:00Z"
}
```

View consumer logs:
```bash
docker logs -f event-consumer
```

## 🔄 Idempotency

The system ensures idempotent processing through:

1. **SQL MERGE statements** - Upsert operations using unique keys
2. **Unique constraints** - Primary keys on userId, orderId, paymentId, sku
3. **Manual offset commits** - Only commit after successful processing
4. **Retry safety** - Replaying events produces same result

### Testing Idempotency

```bash
# Produce the same event twice
# Run producer and create a user
# Stop consumer, replay Kafka from beginning
# Verify database has only one record
```

## ❌ Dead Letter Queue (DLQ)

Failed messages are pushed to Redis with error details:

```json
{
  "eventId": "uuid",
  "originalData": "{...}",
  "error": "failed to parse event: invalid JSON",
  "timestamp": "2025-10-20T10:00:00Z",
  "retryCount": 0
}
```

### Inspecting DLQ

```bash
# Connect to Redis
docker exec -it redis redis-cli

# Check DLQ size
LLEN dlq:events

# View DLQ entries
LRANGE dlq:events 0 -1

# View specific entry
LINDEX dlq:events 0
```

### DLQ Triggers

Messages go to DLQ when:
- JSON parsing fails
- Database constraint violations
- Unexpected errors during processing
- Event type is unknown

## 🛠️ Local Development

### Setup

```bash
# Install dependencies
go mod download

# Copy environment file
cp .env.example .env

# Start infrastructure only
docker-compose up -d kafka mssql redis

# Wait for services to be healthy
docker-compose ps

# Initialize database
docker exec -it mssql /opt/mssql-tools/bin/sqlcmd -S localhost -U sa -P 'YourStrong@Passw0rd' -d master -Q 'CREATE DATABASE eventdb'
docker exec -it mssql /opt/mssql-tools/bin/sqlcmd -S localhost -U sa -P 'YourStrong@Passw0rd' -d eventdb -i schema.sql
```

### Run Services Locally

```bash
# Terminal 1: Run consumer
go run cmd/consumer/main.go

# Terminal 2: Run producer
go run cmd/producer/main.go
```

### Run Tests

```bash
go test ./...
```

## 🐛 Troubleshooting

### Kafka Not Starting
```bash
# Check Zookeeper
docker logs zookeeper

# Restart Kafka
docker-compose restart kafka
```

### MS SQL Connection Issues
```bash
# Verify MS SQL is running
docker exec -it mssql /opt/mssql-tools/bin/sqlcmd -S localhost -U sa -P 'YourStrong@Passw0rd' -Q "SELECT @@VERSION"

# Check database exists
docker exec -it mssql /opt/mssql-tools/bin/sqlcmd -S localhost -U sa -P 'YourStrong@Passw0rd' -Q "SELECT name FROM sys.databases"
```

### Consumer Not Processing
```bash
# Check consumer logs
docker logs -f event-consumer

# Verify Kafka topic exists
docker exec -it kafka kafka-topics --bootstrap-server localhost:9092 --list

# Check consumer group
docker exec -it kafka kafka-consumer-groups --bootstrap-server localhost:9092 --group event-consumer-group --describe
```

### View All Logs
```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f consumer
```

## 🧪 Testing Scenarios

### Scenario 1: Happy Path
1. Start all services: `docker-compose up -d`
2. Run producer: `docker-compose run --rm producer`
3. Select option `5` (Generate Sample Events)
4. Query API: `curl http://localhost:8080/users/{userId}`
5. Verify data in database and metrics

### Scenario 2: Retry & Idempotency
1. Produce events
2. Stop consumer: `docker-compose stop consumer`
3. Reset Kafka offsets: 
   ```bash
   docker exec -it kafka kafka-consumer-groups --bootstrap-server localhost:9092 --group event-consumer-group --reset-offsets --to-earliest --topic events --execute
   ```
4. Start consumer: `docker-compose start consumer`
5. Verify no duplicate records in database

### Scenario 3: DLQ Testing
1. Manually publish invalid JSON to Kafka:
   ```bash
   docker exec -it kafka kafka-console-producer --bootstrap-server localhost:9092 --topic events
   > {"invalid": "json"
   ```
2. Check DLQ: `docker exec -it redis redis-cli LLEN dlq:events`
3. Verify DLQ metrics increased

## 📁 Project Structure

```
event-pipeline/
├── cmd/
│   ├── consumer/
│   │   └── main.go          # Consumer/API entry point
│   └── producer/
│       └── main.go          # Producer entry point
├── internal/                 # Private application code
│   ├── api/
│   │   └── api.go           # REST API handlers
│   ├── config/
│   │   └── config.go        # Configuration management
│   ├── consumer/
│   │   └── consumer.go      # Kafka consumer logic
│   ├── database/
│   │   └── database.go      # MS SQL operations
│   ├── dlq/
│   │   └── dlq.go           # Dead letter queue
│   ├── logger/
│   │   └── logger.go        # Structured logging
│   ├── metrics/
│   │   └── metrics.go       # Prometheus metrics
│   ├── models/
│   │   └── events.go        # Event type definitions
│   └── producer/
│       └── producer.go      # Kafka producer logic
├── .env.example              # Example environment file
├── api-tests.http            # HTTP test file
├── docker-compose.yml        # Docker Compose config
├── Dockerfile.consumer       # Consumer Dockerfile
├── Dockerfile.producer       # Producer Dockerfile
├── go.mod                    # Go dependencies
├── go.sum                    # Go checksums
├── README.md                 # This file
└── schema.sql                # Database schema
```

## 🔒 Security Notes

- Change default passwords in production
- Use secrets management for sensitive data
- Enable Kafka authentication (SASL/SSL)
- Restrict network access with firewalls
- Use TLS for all connections

## 📝 License

MIT License - See LICENSE file for details

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## 📧 Support

For issues and questions, please open a GitHub issue.
