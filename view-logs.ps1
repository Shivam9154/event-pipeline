# View Logs Script - Display consumer logs in real-time
# Run this in a separate terminal to monitor event processing

$ErrorActionPreference = "Continue"

Write-Host ""
Write-Host "================================================================" -ForegroundColor Cyan
Write-Host "       CONSUMER LOGS - Real-Time Monitoring                     " -ForegroundColor Cyan
Write-Host "================================================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "[*] Checking for consumer process..." -ForegroundColor Cyan

# Helper: check consumer/API health on port 8080
function Test-ServiceHealth {
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:8080/health" -UseBasicParsing -TimeoutSec 5
        return $response.StatusCode -eq 200
    } catch {
        return $false
    }
}

# Helper: start consumer as a background job and wait for health
function Start-ConsumerJob {
    param(
        [int]$MaxAttempts = 15,
        [int]$DelaySeconds = 2
    )

    Write-Host "[*] Starting Consumer/API service as a background job..." -ForegroundColor Yellow

    # Clean up any existing job with the same name
    $existingJob = Get-Job -Name "Consumer" -ErrorAction SilentlyContinue
    if ($existingJob) {
        Stop-Job -Name "Consumer" -ErrorAction SilentlyContinue
        Remove-Job -Name "Consumer" -ErrorAction SilentlyContinue
    }

    $job = Start-Job -Name "Consumer" -ScriptBlock {
        Set-Location $using:PWD
        $env:MSSQL_PASSWORD = 'YourStrong@Passw0rd'
        $env:KAFKA_BROKERS = 'localhost:9092'
        $env:MSSQL_SERVER = 'localhost'
        $env:REDIS_HOST = 'localhost'
        go run ./cmd/consumer
    }

    Write-Host "[*] Waiting for Consumer/API to become healthy..." -ForegroundColor Yellow
    for ($i = 1; $i -le $MaxAttempts; $i++) {
        Start-Sleep -Seconds $DelaySeconds
        if (Test-ServiceHealth) {
            Write-Host "[+] Consumer/API is healthy" -ForegroundColor Green
            return $job
        }
        Write-Host "    Attempt $i/$MaxAttempts..." -ForegroundColor Gray
    }

    Write-Host "[-] Consumer/API failed to become healthy in time" -ForegroundColor Red
    return $null
}

# Try to find consumer job if started with Start-Job
$consumerJob = Get-Job -Name "Consumer" -ErrorAction SilentlyContinue
if ($consumerJob) {
    Write-Host "[+] Found consumer job (ID: $($consumerJob.Id))" -ForegroundColor Green
    Write-Host "[*] Streaming logs... (Press Ctrl+C to stop)`n" -ForegroundColor Cyan

    # Track how many lines we've already shown to avoid missing or duplicating output
    $script:__lastCount = 0

    while ($true) {
        # Use -Keep so we can compute a diff and never miss fast bursts
        $allOutput = Receive-Job -Id $consumerJob.Id -Keep 2>$null
        if ($allOutput) {
            # Normalize to individual lines even if some entries contain newlines
            $lines = $allOutput | Out-String -Stream
            $currentCount = ($lines | Measure-Object).Count

            if ($currentCount -gt $script:__lastCount) {
                $startIdx = $script:__lastCount
                $newLines = $lines[$startIdx..($currentCount - 1)]
                $script:__lastCount = $currentCount

                foreach ($line in $newLines) {
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
        }
        Start-Sleep -Milliseconds 100
    }
}
else {
    Write-Host "[-] Consumer job not found" -ForegroundColor Yellow

    if (-not (Test-ServiceHealth)) {
        # Start consumer automatically and stream logs
        $newJob = Start-ConsumerJob
        if ($newJob) {
            Write-Host "[+] Started consumer job (ID: $($newJob.Id))" -ForegroundColor Green

            Write-Host "[*] Streaming logs... (Press Ctrl+C to stop)`n" -ForegroundColor Cyan

            # Track how many lines we've already shown to avoid missing or duplicating output
            $script:__lastCount = 0

            while ($true) {
                # Use -Keep so we can compute a diff and never miss fast bursts
                $allOutput = Receive-Job -Id $newJob.Id -Keep 2>$null
                if ($allOutput) {
                    # Normalize to individual lines even if some entries contain newlines
                    $lines = $allOutput | Out-String -Stream
                    $currentCount = ($lines | Measure-Object).Count

                    if ($currentCount -gt $script:__lastCount) {
                        $startIdx = $script:__lastCount
                        $newLines = $lines[$startIdx..($currentCount - 1)]
                        $script:__lastCount = $currentCount

                        foreach ($line in $newLines) {
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
                }
                Start-Sleep -Milliseconds 100
            }
        } else {
            Write-Host "[-] Failed to start consumer automatically. Showing alternative signals..." -ForegroundColor Red
        }
    } else {
        Write-Host "[*] Consumer/API appears healthy but is running in another terminal." -ForegroundColor Cyan
        Write-Host "    Attempting to tail file logs if available..." -ForegroundColor Yellow

        $logPath = Join-Path -Path $PWD -ChildPath "app.log"
        if (Test-Path $logPath) {
            Write-Host "[+] Tailing: $logPath (press Ctrl+C to stop)" -ForegroundColor Green
            try {
                Get-Content -Path $logPath -Tail 100 -Wait | ForEach-Object {
                    $line = $_
                    $timestamp = Get-Date -Format "HH:mm:ss"
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
            } catch {
                Write-Host "[-] Failed to tail log file: $logPath" -ForegroundColor Red
            }
            return
        } else {
            Write-Host "[-] Log file not found at $logPath" -ForegroundColor Red
            Write-Host "    It will be created automatically next time the consumer starts (logger writes to app.log)." -ForegroundColor Gray
            Write-Host ""
        }
    }

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
    $metricsContent = Invoke-WebRequest -Uri "http://localhost:8080/metrics" -UseBasicParsing -ErrorAction SilentlyContinue 2>$null
    if ($metricsContent) {
        $metricsContent.Content -split "`n" | Where-Object { $_ -match "events_processed_total.*status.*success" } | ForEach-Object {
            Write-Host "  $_" -ForegroundColor Green
        }
    }
}

Write-Host ""
Write-Host "================================================================" -ForegroundColor Cyan
Write-Host ""
