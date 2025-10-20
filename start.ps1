# Quick Start Script for Event Pipeline (PowerShell)

Write-Host "🚀 Starting Event Pipeline..." -ForegroundColor Green
Write-Host ""

# Check if Docker is running
try {
    docker info | Out-Null
    Write-Host "✓ Docker is running" -ForegroundColor Green
} catch {
    Write-Host "❌ Docker is not running. Please start Docker and try again." -ForegroundColor Red
    exit 1
}

# Start services
Write-Host ""
Write-Host "📦 Starting services with Docker Compose..." -ForegroundColor Cyan
docker-compose up -d

Write-Host ""
Write-Host "⏳ Waiting for services to be healthy (this may take 30-60 seconds)..." -ForegroundColor Yellow
Start-Sleep -Seconds 15

# Wait for Kafka
Write-Host "   Waiting for Kafka..." -ForegroundColor Yellow
$maxAttempts = 20
$attempt = 0
while ($attempt -lt $maxAttempts) {
    try {
        docker exec kafka kafka-broker-api-versions --bootstrap-server localhost:9092 2>&1 | Out-Null
        if ($LASTEXITCODE -eq 0) { break }
    } catch {}
    Write-Host "   Still waiting for Kafka..." -ForegroundColor Yellow
    Start-Sleep -Seconds 5
    $attempt++
}
Write-Host "   ✓ Kafka is ready" -ForegroundColor Green

# Wait for MS SQL
Write-Host "   Waiting for MS SQL..." -ForegroundColor Yellow
$attempt = 0
while ($attempt -lt $maxAttempts) {
    try {
        docker exec mssql /opt/mssql-tools/bin/sqlcmd -S localhost -U sa -P 'YourStrong@Passw0rd' -Q "SELECT 1" 2>&1 | Out-Null
        if ($LASTEXITCODE -eq 0) { break }
    } catch {}
    Write-Host "   Still waiting for MS SQL..." -ForegroundColor Yellow
    Start-Sleep -Seconds 5
    $attempt++
}
Write-Host "   ✓ MS SQL is ready" -ForegroundColor Green

# Wait for Redis
Write-Host "   Waiting for Redis..." -ForegroundColor Yellow
$attempt = 0
while ($attempt -lt $maxAttempts) {
    try {
        docker exec redis redis-cli ping 2>&1 | Out-Null
        if ($LASTEXITCODE -eq 0) { break }
    } catch {}
    Write-Host "   Still waiting for Redis..." -ForegroundColor Yellow
    Start-Sleep -Seconds 2
    $attempt++
}
Write-Host "   ✓ Redis is ready" -ForegroundColor Green

# Wait for Consumer/API
Write-Host "   Waiting for Consumer/API..." -ForegroundColor Yellow
Start-Sleep -Seconds 10
$attempt = 0
while ($attempt -lt $maxAttempts) {
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:8080/health" -UseBasicParsing -TimeoutSec 2 2>&1
        if ($response.StatusCode -eq 200) { break }
    } catch {}
    Write-Host "   Still waiting for API..." -ForegroundColor Yellow
    Start-Sleep -Seconds 3
    $attempt++
}
Write-Host "   ✓ Consumer/API is ready" -ForegroundColor Green

Write-Host ""
Write-Host "✅ All services are ready!" -ForegroundColor Green
Write-Host ""
Write-Host "📊 Service URLs:" -ForegroundColor Cyan
Write-Host "   API:     http://localhost:8080"
Write-Host "   Metrics: http://localhost:8080/metrics"
Write-Host "   Health:  http://localhost:8080/health"
Write-Host ""
Write-Host "🎯 Next Steps:" -ForegroundColor Cyan
Write-Host "   1. Run producer: docker-compose run --rm producer"
Write-Host "   2. Or: go run cmd/producer/main.go"
Write-Host "   3. Select option 5 to generate sample events"
Write-Host "   4. Test API: curl http://localhost:8080/health"
Write-Host ""
Write-Host "📝 View logs: docker-compose logs -f consumer"
Write-Host "🛑 Stop services: docker-compose down"
Write-Host ""
