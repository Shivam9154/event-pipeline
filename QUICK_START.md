# Quick Start Guide

This guide will get you up and running with the Event Pipeline in under 2 minutes.

## ⚡ Super Quick Start (One Command)

```powershell
# 1. Start infrastructure
docker-compose up -d

# 2. Run the demo (automatically starts consumer)
.\demo.ps1
```

That's it! The demo script will:
- ✅ Check all services are running
- ✅ **Automatically start the consumer/API** (no need to run separately)
- ✅ Run complete e-commerce flow demonstration
- ✅ Show logs, metrics, and database state
- ✅ Ask if you want to keep consumer running after demo

## 📋 What the Demo Does

### Step 0: Prerequisites Check
- Verifies Docker services (Kafka, Zookeeper, MS SQL, Redis)
- **Auto-starts consumer/API if not running**
- Waits for health endpoint to respond

### Step 1-7: Full Demonstration
1. **Initial State** - Shows empty database
2. **E-Commerce Journey** - Creates User → Order → Payment → Inventory
3. **Idempotency Test** - Sends duplicates, verifies only 1 DB record
4. **Error Handling** - Malformed events go to DLQ
5. **Performance Metrics** - Shows Prometheus counters
6. **Final State** - Database after all events
7. **End-to-End Logs** - Complete event flow trace

## 🎯 After Demo

The consumer stays running by default so you can test the API:

```powershell
# Query a user
curl http://localhost:8080/users/<user-id-from-demo>

# Query an order
curl http://localhost:8080/orders/<order-id-from-demo>

# View metrics
curl http://localhost:8080/metrics

# Check health
curl http://localhost:8080/health
```

## 🛑 Stopping Services

```powershell
# Stop consumer (if you kept it running)
Stop-Job -Name Consumer
Remove-Job -Name Consumer

# Stop Docker services
docker-compose down
```

## 🔧 Manual Consumer Start (Optional)

If you prefer to run the consumer manually in a separate terminal:

```powershell
# Terminal 1: Start consumer
go run cmd/consumer/main.go

# Terminal 2: Run demo
.\demo.ps1
```

The demo will detect the running consumer and skip auto-start.

## 🐛 Troubleshooting

### Port Already in Use

**Redis (6379)**:
```powershell
# Find and stop conflicting Redis
docker ps -a --filter "name=redis"
docker stop <container-name>
```

**Consumer API (8080)**:
```powershell
# Check what's using port 8080
netstat -ano | findstr :8080

# Stop the consumer job if running
Stop-Job -Name Consumer
Remove-Job -Name Consumer
```

### Consumer Won't Start

```powershell
# Check consumer job logs
Receive-Job -Name Consumer

# Verify Go is installed
go version

# Verify working directory
cd "c:\Users\Shivam Patil\OneDrive\Desktop\event-pipeline"
```

### Services Not Ready

```powershell
# Restart Docker services
docker-compose down
docker-compose up -d

# Wait 30 seconds for services to initialize
Start-Sleep -Seconds 30

# Check service status
docker-compose ps
```

## 📚 Next Steps

- Read [DEMO_GUIDE.md](DEMO_GUIDE.md) for detailed explanation
- Check [README.md](README.md) for architecture and API docs
- Run `go test ./...` to execute unit tests
- Explore [api-tests.http](api-tests.http) for REST Client tests

## 🎬 Demo Output

The demo shows:
- ✅ Real-time consumer logs with color-coded events
- ✅ API query results (JSON responses)
- ✅ Database state before/after
- ✅ Prometheus metrics
- ✅ DLQ entries for failed messages
- ✅ Idempotency verification

Enjoy! 🚀
