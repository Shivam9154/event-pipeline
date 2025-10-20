# Event Pipeline Architecture

## System Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         EVENT PIPELINE SYSTEM                            │
└─────────────────────────────────────────────────────────────────────────┘

┌──────────────┐
│  Producer    │  Interactive CLI for generating events
│  (Go)        │  - UserCreated
└──────┬───────┘  - OrderPlaced
       │          - PaymentSettled
       │          - InventoryAdjusted
       ▼
┌──────────────────────────────────────────┐
│            Kafka Topic: events            │
│  Partition by: userId/orderId/SKU        │
│  Retention: Configurable                 │
└──────┬───────────────────────────────────┘
       │
       │ (Consumer Group: event-consumer-group)
       ▼
┌─────────────────────────────────────────┐
│         Consumer (Go)                    │
│  - Parse event type                      │
│  - Route to handler                      │
│  - Idempotent upsert                     │
└──┬──────────────────────┬────────────────┘
   │                      │
   │ (success)            │ (error)
   ▼                      ▼
┌──────────────┐    ┌──────────────┐
│   MS SQL     │    │  Redis DLQ   │
│              │    │              │
│ - users      │    │ - Original   │
│ - orders     │    │   payload    │
│ - payments   │    │ - Error msg  │
│ - inventory  │    │ - Timestamp  │
└──────┬───────┘    └──────────────┘
       │
       │ (read operations)
       ▼
┌─────────────────────────────────────┐
│          Read API (Go)               │
│                                      │
│  GET /users/{id}                    │
│  GET /orders/{id}                   │
│  GET /health                        │
│  GET /metrics                       │
└─────────────────────────────────────┘
```

## Data Flow

### 1. Event Production Flow
```
Producer CLI
    ↓
Select Event Type
    ↓
Generate Event (with UUID)
    ↓
Marshal to JSON
    ↓
Publish to Kafka (with partition key)
    ↓
Delivery Confirmation
    ↓
Log Success (with eventId)
```

### 2. Event Consumption Flow
```
Kafka Consumer (polls)
    ↓
Read Message
    ↓
Parse Base Event (extract eventType)
    ↓
Route by Event Type
    ├─ UserCreated → UpsertUser()
    ├─ OrderPlaced → UpsertOrder()
    ├─ PaymentSettled → UpsertPayment()
    └─ InventoryAdjusted → UpsertInventory()
    ↓
Execute SQL MERGE (idempotent)
    ↓
Success? ─┬─ Yes → Commit Offset + Log Success
              │
              └─ No → Push to DLQ + Log Error + Commit Offset
```

### 3. API Query Flow
```
HTTP Request
    ↓
Route Handler
    ↓
Database Query (with JOIN)
    ↓
Marshal to JSON
    ↓
HTTP Response
```

## Event Types Detail

### UserCreated
```json
{
  "eventId": "uuid",
  "eventType": "UserCreated",
  "timestamp": "ISO8601",
  "userId": "uuid",      ← PARTITION KEY
  "email": "string",
  "firstName": "string",
  "lastName": "string",
  "createdAt": "ISO8601"
}
```

### OrderPlaced
```json
{
  "eventId": "uuid",
  "eventType": "OrderPlaced",
  "timestamp": "ISO8601",
  "orderId": "uuid",     ← PARTITION KEY
  "userId": "uuid",
  "totalAmount": 299.99,
  "currency": "USD",
  "items": [
    {"sku": "string", "quantity": 1, "price": 299.99}
  ],
  "placedAt": "ISO8601"
}
```

### PaymentSettled
```json
{
  "eventId": "uuid",
  "eventType": "PaymentSettled",
  "timestamp": "ISO8601",
  "paymentId": "uuid",
  "orderId": "uuid",     ← PARTITION KEY
  "amount": 299.99,
  "currency": "USD",
  "paymentMethod": "credit_card",
  "status": "completed",
  "settledAt": "ISO8601"
}
```

### InventoryAdjusted
```json
{
  "eventId": "uuid",
  "eventType": "InventoryAdjusted",
  "timestamp": "ISO8601",
  "sku": "LAPTOP-001",   ← PARTITION KEY
  "quantity": 10,
  "adjustmentType": "add",
  "reason": "restock",
  "adjustedAt": "ISO8601"
}
```

## Database Schema

```
┌─────────────────┐
│     users       │
├─────────────────┤
│ user_id (PK)    │◄──────┐
│ email (UQ)      │       │
│ first_name      │       │
│ last_name       │       │
│ created_at      │       │
│ updated_at      │       │
└─────────────────┘       │
                          │
                          │ (FK)
┌─────────────────┐       │
│     orders      │       │
├─────────────────┤       │
│ order_id (PK)   │◄──┐   │
│ user_id (FK)    │───┘───┘
│ total_amount    │
│ currency        │
│ placed_at       │
│ updated_at      │
└─────────────────┘
        │
        │ (FK)
        ▼
┌─────────────────┐
│  order_items    │
├─────────────────┤
│ id (PK)         │
│ order_id (FK)   │
│ sku             │
│ quantity        │
│ price           │
└─────────────────┘

┌─────────────────┐
│    payments     │
├─────────────────┤
│ payment_id (PK) │
│ order_id (FK)   │───┐
│ amount          │   │
│ currency        │   │
│ payment_method  │   │
│ status          │   │
│ settled_at      │   │
│ updated_at      │   │
└─────────────────┘   │
                      │
                      └──► Links to orders

┌─────────────────┐
│   inventory     │
├─────────────────┤
│ sku (PK)        │
│ quantity        │
│ updated_at      │
└─────────────────┘
```

## Metrics & Observability

### Prometheus Metrics
```
events_processed_total{event_type, status}
    ↓ Counter
    
events_processed_per_second{event_type}
    ↓ Gauge (calculated every 1s)
    
dlq_entries_total
    ↓ Counter
    
db_operation_duration_seconds{operation}
    ↓ Histogram (p50, p95, p99)
    
kafka_produce_duration_seconds
    ↓ Histogram
    
kafka_consume_duration_seconds
    ↓ Histogram
```

### Structured Logging
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

## Idempotency Mechanism

### SQL MERGE Example (UpsertUser)
```sql
MERGE INTO users AS target
USING (SELECT @user_id AS user_id) AS source
ON target.user_id = source.user_id
WHEN MATCHED THEN
    UPDATE SET email = @email, updated_at = @now
WHEN NOT MATCHED THEN
    INSERT (user_id, email, ...) VALUES (@user_id, @email, ...)
```

**Result**: Same event replayed multiple times → Same database state

## Error Handling & DLQ

### Error Flow
```
Event Processing Error
    ↓
Catch Exception
    ↓
Create DLQ Entry
    ├─ eventId
    ├─ originalData (full JSON)
    ├─ error (error message)
    ├─ timestamp
    └─ retryCount
    ↓
Push to Redis List (RPUSH)
    ↓
Increment DLQ Metric
    ↓
Log Warning with eventId
    ↓
Commit Kafka Offset (don't retry indefinitely)
```

### DLQ Structure in Redis
```
Key: "dlq:events"
Type: LIST (FIFO)

Entry Format:
{
  "eventId": "uuid",
  "originalData": "{...original event...}",
  "error": "failed to parse: invalid JSON",
  "timestamp": "2025-10-20T10:00:00Z",
  "retryCount": 0
}
```

## Deployment Architecture

```
┌────────────────────────────────────────────────────────────┐
│                    Docker Compose                           │
├────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │
│  │  Zookeeper   │  │    Kafka     │  │   MS SQL     │    │
│  │  :2181       │◄─┤  :9092       │  │  :1433       │    │
│  └──────────────┘  └──────────────┘  └──────────────┘    │
│                                                             │
│  ┌──────────────┐  ┌──────────────────────────────────┐  │
│  │   Redis      │  │    Consumer/API                   │  │
│  │   :6379      │◄─┤    :8080 (API)                   │  │
│  └──────────────┘  │    :9090 (Metrics)                │  │
│                    └──────────────────────────────────┘  │
│                                                             │
└────────────────────────────────────────────────────────────┘
        ▲                           │
        │                           ▼
   [Producer CLI]            [HTTP Clients]
```

## Scalability Considerations

1. **Kafka Partitions**: Scale by adding partitions to distribute load
2. **Consumer Instances**: Multiple consumer instances in same group for parallel processing
3. **Database**: Connection pooling configured (max 25 connections)
4. **Redis**: Single instance sufficient for DLQ (can cluster for HA)

## Security Notes

- Change default passwords in production
- Use Kafka SASL/SSL authentication
- Enable MS SQL encryption
- Use Redis password authentication
- Network isolation via Docker networks
- API rate limiting (not implemented, but recommended)

---

**Architecture Status**: Production-ready with observability, idempotency, and error handling
