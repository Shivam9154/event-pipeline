# Event Pipeline Demo Scripts

Comprehensive demonstration scripts that showcase the complete event-driven pipeline functionality.

## Available Demo Scripts

### 1. **PowerShell Demo** (`demo.ps1`) - Windows
Full-featured interactive demo for Windows with colored output and comprehensive testing.

### 2. **Bash Demo** (`demo.sh`) - Linux/Mac
Cross-platform demo script with the same functionality for Unix-based systems.

---

## What the Demo Shows

The demo scripts automatically demonstrate **6 key capabilities**:

### 🎯 **Step 1: Initial State Check**
- Shows current database records count
- Displays current Prometheus metrics
- Verifies all services are running

### 🛒 **Step 2: E-Commerce User Journey (Happy Path)**
Complete user purchase flow:
1. **UserCreated** - Alice Johnson registers
2. **OrderPlaced** - Orders laptop for $1,299.99
3. **PaymentSettled** - Payment processed successfully
4. **InventoryAdjusted** - Stock reduced for shipment

Then queries the REST API to verify data:
- `GET /users/{id}` - Shows user with orders
- `GET /orders/{id}` - Shows order with payment

### 🔄 **Step 3: Idempotency Test**
- Sends the **same event 3 times**
- Verifies only **1 record** exists in database
- Proves SQL MERGE prevents duplicates

### ❌ **Step 4: Error Handling & Dead Letter Queue**
Sends intentionally malformed events:
- Invalid JSON syntax
- Missing required fields
- Unknown event types

Verifies all failures are captured in Redis DLQ with:
- Original payload
- Error message
- Retry count

### 📊 **Step 5: Performance Metrics**
Displays real-time Prometheus metrics:
- `events_processed_total` (success/error breakdown)
- `db_operation_duration_seconds` (latency)
- `dlq_entries_total` (failed messages)

### 📈 **Step 6: Final Database State**
Shows complete system state:
- Total records per table
- Recent users created
- Recent orders placed

---

## Prerequisites

Before running the demo:

1. **Infrastructure must be running:**
   ```bash
   docker-compose up -d
   ```

2. **Consumer/API service must be running:**
   ```powershell
   # PowerShell (Windows)
   Start-Job -ScriptBlock { go run cmd/consumer/main.go }
   
   # Bash (Linux/Mac)
   go run cmd/consumer/main.go &
   ```

3. **Database must be initialized:**
   ```bash
   docker exec -it mssql /opt/mssql-tools18/bin/sqlcmd -S localhost -U sa \
     -P 'YourStrong@Passw0rd' -C -Q "CREATE DATABASE eventdb"
   
   Get-Content schema.sql | docker exec -i mssql /opt/mssql-tools18/bin/sqlcmd \
     -S localhost -U sa -P 'YourStrong@Passw0rd' -C -d eventdb
   ```

---

## Running the Demo

### Windows (PowerShell)
```powershell
# Run the demo
.\demo.ps1

# The script will automatically:
# - Check all services are running
# - Create test events
# - Verify results via API and database
# - Display colored output with emojis
```

### Linux/Mac (Bash)
```bash
# Make executable
chmod +x demo.sh

# Run the demo
./demo.sh

# The script will automatically:
# - Check all services are running
# - Create test events
# - Verify results via API and database
# - Display colored output
```

---

## Expected Output

### Successful Demo Run Example:

```
╔════════════════════════════════════════════════════════════════════╗
║          EVENT PIPELINE - WORKING PROTOTYPE DEMO                  ║
║  Go + Kafka + MS SQL + Redis + Docker Compose                     ║
╚════════════════════════════════════════════════════════════════════╝

═══════════════════════════════════════════════════════════════
  STEP 0: Prerequisites Check
═══════════════════════════════════════════════════════════════

✓ kafka is running
✓ zookeeper is running
✓ mssql is running
✓ redis is running
✓ Consumer/API service is running

✅ All services are ready!

═══════════════════════════════════════════════════════════════
  STEP 1: Initial State
═══════════════════════════════════════════════════════════════

📊 Current database state:
TableName   Count
Users       27
Orders      5
Payments    3
Inventory   6

═══════════════════════════════════════════════════════════════
  STEP 2: E-Commerce User Journey
═══════════════════════════════════════════════════════════════

📝 Scenario:
  1. New user registers (Alice Johnson)
  2. User places order for laptop ($1,299.99)
  3. Payment is processed successfully
  4. Inventory is adjusted for shipped items

🚀 Publishing events...
  ✓ Event 1: UserCreated - Alice Johnson registered
  ✓ Event 2: OrderPlaced - Laptop order $1,299.99
  ✓ Event 3: PaymentSettled - Payment completed
  ✓ Event 4: InventoryAdjusted - Stock reduced for shipment

✅ All events published successfully!

⏳ Waiting for events to be processed... ✓

📊 Verification - API Query Results:

GET /users/a1b2c3d4-e5f6-...
  User: Alice Johnson
  Email: alice.johnson@example.com
  Orders: 1

GET /orders/x1y2z3...
  Order Total: $1299.99
  Payment Status: completed
  Payment Method: credit_card

═══════════════════════════════════════════════════════════════
  STEP 3: Idempotency Test
═══════════════════════════════════════════════════════════════

  📤 Attempt 1: Sent duplicate event
  📤 Attempt 2: Sent duplicate event
  📤 Attempt 3: Sent duplicate event

🔍 Database Check:
  Records in database: 1
  ✅ Result: Only 1 record (idempotency working!)

═══════════════════════════════════════════════════════════════
  STEP 4: Error Handling & Dead Letter Queue
═══════════════════════════════════════════════════════════════

  ❌ Sent: Invalid JSON
  ❌ Sent: Missing Fields
  ❌ Sent: Unknown Type

🔍 Dead Letter Queue Status:
  DLQ Entries: 9
  
  Sample DLQ Entry:
    Event ID: test-456
    Error: unknown event type: InvalidEvent
    Retry Count: 0

  ✅ All errors captured in DLQ

═══════════════════════════════════════════════════════════════
  STEP 5: Performance Metrics
═══════════════════════════════════════════════════════════════

Event Processing Statistics:
  events_processed_total{event_type="UserCreated",status="success"} 35
  events_processed_total{event_type="OrderPlaced",status="success"} 6
  events_processed_total{event_type="PaymentSettled",status="success"} 4
  events_processed_total{event_type="InventoryAdjusted",status="success"} 7

Dead Letter Queue:
  dlq_entries_total 9

╔════════════════════════════════════════════════════════════════════╗
║                  🎉 DEMO COMPLETED SUCCESSFULLY! 🎉                ║
║           Event Pipeline is Production-Ready! ✅                  ║
╚════════════════════════════════════════════════════════════════════╝
```

---

## Demo Architecture

The demo creates temporary Go programs in `cmd/` directory:

- **`cmd/demo-producer/main.go`** - Publishes e-commerce journey events
- **`cmd/idempotency-test/main.go`** - Tests duplicate event handling
- **`cmd/dlq-test/main.go`** - Tests error scenarios

These are automatically created and executed by the demo scripts.

---

## Key Features Demonstrated

### ✅ **End-to-End Flow**
```
Producer → Kafka → Consumer → MS SQL → API
```

### ✅ **Event Types**
- UserCreated
- OrderPlaced
- PaymentSettled
- InventoryAdjusted

### ✅ **Idempotency**
- SQL MERGE prevents duplicate records
- Same event can be replayed safely

### ✅ **Error Recovery**
- Malformed events go to Dead Letter Queue (Redis)
- Original payload preserved for debugging
- Retry count tracked

### ✅ **REST API**
- `GET /users/{id}` - Returns user with orders (JOIN)
- `GET /orders/{id}` - Returns order with payment (JOIN)
- `GET /health` - Health check
- `GET /metrics` - Prometheus metrics

### ✅ **Observability**
- Structured JSON logging
- Prometheus metrics (counters, histograms, gauges)
- Database latency tracking
- Event processing rates

---

## Troubleshooting

### Demo fails at "Prerequisites Check"
**Issue:** Services not running  
**Solution:**
```bash
docker-compose up -d
sleep 10  # Wait for services to be ready
```

### "Consumer/API service is NOT running"
**Issue:** Consumer not started  
**Solution:**
```powershell
# Start in background
Start-Job -ScriptBlock { 
    Set-Location "c:\Users\Shivam Patil\OneDrive\Desktop\event-pipeline"
    go run cmd/consumer/main.go 
}
```

### API queries return 404
**Issue:** Database not initialized or data not found  
**Solution:**
```bash
# Reinitialize database
docker exec -it mssql /opt/mssql-tools18/bin/sqlcmd -S localhost -U sa \
  -P 'YourStrong@Passw0rd' -C -d eventdb -i schema.sql
```

### Events not being consumed
**Issue:** Consumer crashed or Kafka topic doesn't exist  
**Solution:**
```bash
# Check consumer logs
Get-Job | Receive-Job

# Check Kafka topic
docker exec kafka kafka-topics --list --bootstrap-server localhost:9092
```

---

## Customization

### Add More Test Scenarios

Edit the demo script and add a new step:

```powershell
# Step 7: Custom Scenario
Write-Section "STEP 7: Custom Test"
Write-Host "Running custom test..." -ForegroundColor White

# Your custom Go code here
$customTest = @"
package main
// Your custom test
"@

$customTest | Out-File -FilePath "cmd/custom-test/main.go" -Encoding UTF8
go run cmd/custom-test/main.go
```

### Change Test Data

Modify the JSON in the demo script:

```powershell
$testData = @"
{
  "userId": "your-custom-id",
  "userEmail": "custom@example.com",
  "amount": 999.99
}
"@
```

---

## Cleanup

After running the demo, cleanup temporary files:

```powershell
# Remove generated test programs
Remove-Item -Recurse -Force cmd/demo-producer, cmd/idempotency-test, cmd/dlq-test

# Clear DLQ (optional)
docker exec redis redis-cli DEL dlq:events
```

---

## Integration Testing

The demo can be integrated into CI/CD pipelines:

```yaml
# .github/workflows/demo.yml
- name: Run Demo
  run: |
    docker-compose up -d
    sleep 10
    go run cmd/consumer/main.go &
    sleep 5
    ./demo.sh
    
- name: Verify Results
  run: |
    # Check metrics
    curl http://localhost:8080/metrics | grep "events_processed_total"
    
    # Check DLQ is functional
    docker exec redis redis-cli LLEN dlq:events
```

---

## Next Steps

After successful demo:

1. **Explore the API:**
   ```bash
   curl http://localhost:8080/users/{id} | jq
   curl http://localhost:8080/orders/{id} | jq
   ```

2. **Monitor Metrics:**
   ```bash
   curl http://localhost:8080/metrics
   ```

3. **Inspect DLQ:**
   ```bash
   docker exec redis redis-cli LRANGE dlq:events 0 -1
   ```

4. **Check Database:**
   ```bash
   docker exec -it mssql /opt/mssql-tools18/bin/sqlcmd -S localhost \
     -U sa -P 'YourStrong@Passw0rd' -C -d eventdb -Q "SELECT * FROM users"
   ```

---

## Support

For issues or questions:
- Check `TEST_RESULTS.md` for comprehensive test documentation
- Review `PROJECT_SUMMARY.md` for architecture overview
- See `ARCHITECTURE.md` for system design details
