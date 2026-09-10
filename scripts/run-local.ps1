Write-Host "Starting Heimdall worker + controller..." -ForegroundColor Cyan

$env:HEIMDALL_INTERNAL_TOKEN = if ($env:HEIMDALL_INTERNAL_TOKEN) { $env:HEIMDALL_INTERNAL_TOKEN } else { "local-dev-token" }
$env:HEIMDALL_WORKER_URL = "http://localhost:9090"
$env:HEIMDALL_INTERNAL_ADDR = ":9090"

$worker = Start-Process powershell -PassThru -ArgumentList "-NoExit", "-Command", "go run ./cmd/heimdall-server"
Start-Sleep -Seconds 2  # give the worker a head start so its internal API is up before the controller's first health check

$controller = Start-Process powershell -PassThru -ArgumentList "-NoExit", "-Command", "go run ./cmd/heimdall-ui"

Write-Host "Worker PID: $($worker.Id)"
Write-Host "Controller PID: $($controller.Id)"
Write-Host "Dashboard: http://localhost:8080"
Write-Host "Close both terminal windows to stop."