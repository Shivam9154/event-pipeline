# View Logs Script - Display consumer logs in real-time
# Run this in a separate terminal to monitor event processing

$ErrorActionPreference = "Continue"

Write-Host ""
Write-Host "================================================================" -ForegroundColor Cyan
Write-Host "       CONSUMER LOGS - Real-Time Monitoring                     " -ForegroundColor Cyan
Write-Host "================================================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "[*] Checking for consumer process..." -ForegroundColor Cyan

# Try to find consumer job if started with Start-Job
$consumerJob = Get-Job -Name "Consumer" -ErrorAction SilentlyContinue
if ($consumerJob) {
    Write-Host "[+] Found consumer job (ID: $($consumerJob.Id))" -ForegroundColor Green
    Write-Host "[*] Streaming logs... (Press Ctrl+C to stop)`n" -ForegroundColor Cyan
    
    while ($true) {
        $output = Receive-Job -Id $consumerJob.Id 2>$null
        if ($output) {
            foreach ($line in $output) {
                $timestamp = Get-Date -Format "HH:mm:ss"
                
                # Color code based on content
                if ($line -match "ERROR" -or $line -match '"level":"error"') {
                    Write-Host "[$timestamp] " -ForegroundColor Gray -NoNewline
                    Write-Host "[ERROR] $line" -ForegroundColor Red
                }
                elseif ($line -match "UserCreated|OrderPlaced|PaymentSettled|InventoryAdjusted") {
                    Write-Host "[$timestamp] " -ForegroundColor Gray -NoNewline
                    Write-Host "[EVENT] $line" -ForegroundColor Green
                }
                elseif ($line -match "DLQ|failed|Sent to DLQ") {
                    Write-Host "[$timestamp] " -ForegroundColor Gray -NoNewline
                    Write-Host "[DLQ]   $line" -ForegroundColor Yellow
                }
                elseif ($line -match "Processed event|successfully") {
                    Write-Host "[$timestamp] " -ForegroundColor Gray -NoNewline
                    Write-Host "[OK]    $line" -ForegroundColor Cyan
                }
                elseif ($line -match "eventId") {
                    Write-Host "[$timestamp] " -ForegroundColor Gray -NoNewline
                    Write-Host "[INFO]  $line" -ForegroundColor White
                }
                else {
                    Write-Host "[$timestamp] " -ForegroundColor Gray -NoNewline
                    Write-Host "[DEBUG] $line" -ForegroundColor DarkGray
                }
            }
        }
        Start-Sleep -Milliseconds 100
    }
}
else {
    Write-Host "[-] Consumer job not found" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "[*] Consumer might be running in another terminal." -ForegroundColor Cyan
    Write-Host "[*] To view logs, check the terminal where you ran:" -ForegroundColor Cyan
    Write-Host "    go run cmd/consumer/main.go" -ForegroundColor White
    Write-Host ""
    Write-Host "[*] Alternative: Check database for processed events:" -ForegroundColor Cyan
    Write-Host ""
    
    # Show recent database activity
    Write-Host "Recent Users (Last 5):" -ForegroundColor Yellow
    docker exec mssql /opt/mssql-tools18/bin/sqlcmd -S localhost -U sa -P 'YourStrong@Passw0rd' -C -d eventdb -Q "SELECT TOP 5 user_id, email, first_name, last_name, created_at FROM users ORDER BY created_at DESC" -h -1 -W 2>$null
    
    Write-Host ""
    Write-Host "Recent Orders (Last 5):" -ForegroundColor Yellow
    docker exec mssql /opt/mssql-tools18/bin/sqlcmd -S localhost -U sa -P 'YourStrong@Passw0rd' -C -d eventdb -Q "SELECT TOP 5 order_id, user_id, total_amount, currency, placed_at FROM orders ORDER BY placed_at DESC" -h -1 -W 2>$null
    
    Write-Host ""
    Write-Host "DLQ Entries:" -ForegroundColor Yellow
    $dlqCount = docker exec redis redis-cli LLEN dlq:events 2>$null
    Write-Host "  Total: $dlqCount entries" -ForegroundColor Cyan
    
    if ($dlqCount -gt 0) {
        Write-Host "  Latest error:" -ForegroundColor Cyan
        $latestError = docker exec redis redis-cli LINDEX dlq:events -1 2>$null | ConvertFrom-Json
        Write-Host "    Event ID: $($latestError.eventId)" -ForegroundColor Gray
        Write-Host "    Error: $($latestError.error)" -ForegroundColor Red
        Write-Host "    Timestamp: $($latestError.timestamp)" -ForegroundColor Gray
    }
    
    Write-Host ""
    Write-Host "[*] Metrics:" -ForegroundColor Yellow
    $metricsContent = curl "http://localhost:8080/metrics" -UseBasicParsing 2>$null
    if ($metricsContent) {
        $metricsContent.Content -split "`n" | Where-Object { $_ -match "events_processed_total.*status.*success" } | ForEach-Object {
            Write-Host "  $_" -ForegroundColor Green
        }
    }
}

Write-Host ""
Write-Host "================================================================" -ForegroundColor Cyan
Write-Host ""
