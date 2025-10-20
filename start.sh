#!/bin/bash
# Quick Start Script for Event Pipeline

set -e

echo "🚀 Starting Event Pipeline..."
echo ""

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker and try again."
    exit 1
fi

echo "✓ Docker is running"

# Start services
echo ""
echo "📦 Starting services with Docker Compose..."
docker-compose up -d

echo ""
echo "⏳ Waiting for services to be healthy (this may take 30-60 seconds)..."
sleep 15

# Wait for Kafka
echo "   Waiting for Kafka..."
until docker exec kafka kafka-broker-api-versions --bootstrap-server localhost:9092 > /dev/null 2>&1; do
    echo "   Still waiting for Kafka..."
    sleep 5
done
echo "   ✓ Kafka is ready"

# Wait for MS SQL
echo "   Waiting for MS SQL..."
until docker exec mssql /opt/mssql-tools/bin/sqlcmd -S localhost -U sa -P 'YourStrong@Passw0rd' -Q "SELECT 1" > /dev/null 2>&1; do
    echo "   Still waiting for MS SQL..."
    sleep 5
done
echo "   ✓ MS SQL is ready"

# Wait for Redis
echo "   Waiting for Redis..."
until docker exec redis redis-cli ping > /dev/null 2>&1; do
    echo "   Still waiting for Redis..."
    sleep 2
done
echo "   ✓ Redis is ready"

# Wait for Consumer/API
echo "   Waiting for Consumer/API..."
sleep 10
until curl -s http://localhost:8080/health > /dev/null 2>&1; do
    echo "   Still waiting for API..."
    sleep 3
done
echo "   ✓ Consumer/API is ready"

echo ""
echo "✅ All services are ready!"
echo ""
echo "📊 Service URLs:"
echo "   API:     http://localhost:8080"
echo "   Metrics: http://localhost:8080/metrics"
echo "   Health:  http://localhost:8080/health"
echo ""
echo "🎯 Next Steps:"
echo "   1. Run producer: docker-compose run --rm producer"
echo "   2. Or: go run cmd/producer/main.go"
echo "   3. Select option 5 to generate sample events"
echo "   4. Test API: curl http://localhost:8080/health"
echo ""
echo "📝 View logs: docker-compose logs -f consumer"
echo "🛑 Stop services: docker-compose down"
echo ""
