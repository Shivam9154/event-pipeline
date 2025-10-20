# Test Scenarios for Event Pipeline

## Scenario 1: Happy Path Flow

### Prerequisites
- All services running via `docker-compose up -d`
- Database initialized

### Steps

1. **Generate Sample Events**
   ```bash
   go run cmd/producer/main.go
   # Select option 5: Generate Sample Events
   ```

2. **Verify Events in Kafka**
   ```bash
   docker exec -it kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic events --from-beginning --max-messages 10
   ```

3. **Check Consumer Logs**
   ```bash
   docker logs -f event-consumer
   ```
   Look for:
   - `"level":"info","msg":"User upserted successfully"`
   - `"level":"info","msg":"Order upserted successfully"`
   - `"level":"info","msg":"Payment upserted successfully"`
   - `"level":"info","msg":"Inventory adjusted successfully"`

4. **Query API for User**
   ```bash
   # Get a userId from producer output, then:
   curl http://localhost:8080/users/{userId}
   ```
   Expected: User with last 5 orders

5. **Query API for Order**
   ```bash
   # Get an orderId from producer output, then:
   curl http://localhost:8080/orders/{orderId}
   ```
   Expected: Order with payment status

6. **Check Metrics**
   ```bash
   curl http://localhost:8080/metrics | grep events_processed_total
   ```
   Expected: Count of processed events by type

---

## Scenario 2: Idempotency Test

### Objective
Verify that replaying events doesn't create duplicates

### Steps

1. **Produce Initial Events**
   ```bash
   go run cmd/producer/main.go
   # Select option 1 to create a user
   # Note the userId from output
   ```

2. **Query Database - Count Records**
   ```bash
   docker exec -it mssql /opt/mssql-tools/bin/sqlcmd -S localhost -U sa -P 'YourStrong@Passw0rd' -d eventdb -Q "SELECT COUNT(*) FROM users"
   ```
   Note the count (should be 1)

3. **Stop Consumer**
   ```bash
   docker-compose stop consumer
   ```

4. **Reset Kafka Offsets to Replay**
   ```bash
   docker exec -it kafka kafka-consumer-groups --bootstrap-server localhost:9092 --group event-consumer-group --reset-offsets --to-earliest --topic events --execute
   ```

5. **Restart Consumer**
   ```bash
   docker-compose start consumer
   ```

6. **Verify Count Unchanged**
   ```bash
   docker exec -it mssql /opt/mssql-tools/bin/sqlcmd -S localhost -U sa -P 'YourStrong@Passw0rd' -d eventdb -Q "SELECT COUNT(*) FROM users"
   ```
   Expected: Same count as before (no duplicates)

7. **Verify Updated Timestamp**
   ```bash
   docker exec -it mssql /opt/mssql-tools/bin/sqlcmd -S localhost -U sa -P 'YourStrong@Passw0rd' -d eventdb -Q "SELECT user_id, updated_at FROM users ORDER BY updated_at DESC"
   ```
   Expected: updated_at should be recent

---

## Scenario 3: Dead Letter Queue (DLQ) Test

### Objective
Verify failed messages go to DLQ with error details

### Steps

1. **Produce Invalid Event**
   ```bash
   docker exec -it kafka kafka-console-producer --bootstrap-server localhost:9092 --topic events
   # Type: {"invalid": "json without closing brace
   # Press Enter, then Ctrl+C
   ```

2. **Check Consumer Logs**
   ```bash
   docker logs event-consumer | grep DLQ
   ```
   Expected: Log showing message pushed to DLQ

3. **Verify DLQ Count**
   ```bash
   docker exec -it redis redis-cli LLEN dlq:events
   ```
   Expected: Count > 0

4. **View DLQ Entry**
   ```bash
   docker exec -it redis redis-cli LINDEX dlq:events 0
   ```
   Expected: JSON with eventId, originalData, error, timestamp

5. **Check DLQ Metric**
   ```bash
   curl http://localhost:8080/metrics | grep dlq_entries_total
   ```
   Expected: Counter incremented

---

## Scenario 4: Performance & Metrics Test

### Objective
Generate load and verify metrics

### Steps

1. **Produce Multiple Events**
   ```bash
   go run cmd/producer/main.go
   # Select option 5: Generate Sample Events
   # Repeat 3-4 times
   ```

2. **Check Processing Rate**
   ```bash
   curl http://localhost:8080/metrics | grep events_processed_per_second
   ```
   Expected: Rate > 0 for each event type

3. **Check DB Latency P95**
   ```bash
   curl http://localhost:8080/metrics | grep db_operation_duration_seconds
   ```
   Look for histogram buckets and quantiles

4. **View Consumer Logs with EventID Correlation**
   ```bash
   docker logs event-consumer | grep eventId | jq .
   ```
   Expected: JSON logs with eventId field for correlation

---

## Scenario 5: Database Query Verification

### Objective
Verify data integrity in MS SQL

### Steps

1. **Connect to MS SQL**
   ```bash
   docker exec -it mssql /opt/mssql-tools/bin/sqlcmd -S localhost -U sa -P 'YourStrong@Passw0rd' -d eventdb
   ```

2. **Check Users**
   ```sql
   SELECT COUNT(*) as user_count FROM users;
   SELECT TOP 5 * FROM users ORDER BY created_at DESC;
   GO
   ```

3. **Check Orders**
   ```sql
   SELECT COUNT(*) as order_count FROM orders;
   SELECT TOP 5 o.order_id, o.user_id, o.total_amount, o.currency 
   FROM orders o ORDER BY o.placed_at DESC;
   GO
   ```

4. **Check Order Items**
   ```sql
   SELECT TOP 10 oi.order_id, oi.sku, oi.quantity, oi.price 
   FROM order_items oi;
   GO
   ```

5. **Check Payments**
   ```sql
   SELECT COUNT(*) as payment_count FROM payments;
   SELECT TOP 5 p.payment_id, p.order_id, p.status 
   FROM payments p ORDER BY p.settled_at DESC;
   GO
   ```

6. **Check Inventory**
   ```sql
   SELECT * FROM inventory ORDER BY updated_at DESC;
   GO
   ```

7. **Verify Foreign Key Relationships**
   ```sql
   SELECT u.user_id, u.email, COUNT(o.order_id) as order_count
   FROM users u
   LEFT JOIN orders o ON u.user_id = o.user_id
   GROUP BY u.user_id, u.email;
   GO
   ```

8. **Check Join: Orders with Payments**
   ```sql
   SELECT o.order_id, o.total_amount, p.payment_id, p.status
   FROM orders o
   LEFT JOIN payments p ON o.order_id = p.order_id
   ORDER BY o.placed_at DESC;
   GO
   ```

---

## Scenario 6: API Response Validation

### Prerequisites
- Have produced sample events with known IDs

### Steps

1. **Test Health Endpoint**
   ```bash
   curl -X GET http://localhost:8080/health -i
   ```
   Expected: 200 OK with JSON `{"status":"healthy"}`

2. **Test User Endpoint (Valid ID)**
   ```bash
   USER_ID="<your-user-id>"
   curl -X GET http://localhost:8080/users/$USER_ID | jq .
   ```
   Expected: User object with orders array

3. **Test User Endpoint (Invalid ID)**
   ```bash
   curl -X GET http://localhost:8080/users/non-existent-id -i
   ```
   Expected: 404 Not Found

4. **Test Order Endpoint (Valid ID)**
   ```bash
   ORDER_ID="<your-order-id>"
   curl -X GET http://localhost:8080/orders/$ORDER_ID | jq .
   ```
   Expected: Order object with optional payment object

5. **Verify Response Structure**
   ```bash
   # Check user response has required fields
   curl -X GET http://localhost:8080/users/$USER_ID | jq 'keys'
   # Expected: ["createdAt", "email", "firstName", "lastName", "orders", "updatedAt", "userId"]
   
   # Check order response
   curl -X GET http://localhost:8080/orders/$ORDER_ID | jq 'keys'
   # Expected: ["currency", "orderId", "payment", "placedAt", "totalAmount", "updatedAt", "userId"]
   ```

---

## Cleanup

After testing:

```bash
# Stop all services
docker-compose down

# Remove volumes (clean slate)
docker-compose down -v

# Or keep data and just stop
docker-compose stop
```
