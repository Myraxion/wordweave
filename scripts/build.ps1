$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path $PSScriptRoot -Parent
Push-Location $repoRoot
try {
    npm --prefix web ci --no-audit --no-fund
    if ($LASTEXITCODE -ne 0) { throw 'Frontend dependency installation failed.' }
    npm --prefix web run build
    if ($LASTEXITCODE -ne 0) { throw 'Frontend build failed.' }
    $previousCgo = $env:CGO_ENABLED
    try {
        $env:CGO_ENABLED = '0'
        go build -trimpath -ldflags='-s -w' -o bin/wordweave.exe ./cmd/wordweave
        if ($LASTEXITCODE -ne 0) { throw 'Go build failed.' }
    } finally {
        $env:CGO_ENABLED = $previousCgo
    }
    Write-Host 'Built bin/wordweave.exe'
} finally {
    Pop-Location
}
