Write-Host "== Heimdall setup ==" -ForegroundColor Cyan

Write-Host "`nFetching Go modules..."
go mod download
go mod tidy

if (-not (Test-Path ".env")) {
    Write-Host "`nCreating .env from .env.example..."
    Copy-Item ".env.example" ".env"
} else {
    Write-Host "`n.env already exists, leaving as-is."
}

if (-not (Test-Path "testlogs")) {
    Write-Host "`nCreating ./testlogs..."
    New-Item -ItemType Directory -Path "testlogs" | Out-Null
    New-Item -ItemType File -Path "testlogs\messages", "testlogs\auth.log", "testlogs\middlewared.log" | Out-Null
}

Write-Host "`nChecking Ollama..."
try {
    $tags = Invoke-RestMethod -Uri "http://localhost:11434/api/tags" -TimeoutSec 3
    $models = $tags.models | ForEach-Object { $_.name }
    Write-Host "Ollama running. Models: $($models -join ', ')" -ForegroundColor Green
    if ($models -notcontains "qwen2.5:0.5b") {
        & ollama pull qwen2.5:0.5b
    }
} catch {
    Write-Host "Could not reach Ollama — install from https://ollama.com/download" -ForegroundColor Red
}

Write-Host "`nFormatting and linting..."
& "$PSScriptRoot\format.ps1"
& "$PSScriptRoot\lint.ps1"

Write-Host "`n== Setup complete == Run: go run ./cmd/heimdall" -ForegroundColor Cyan