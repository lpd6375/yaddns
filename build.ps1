param()

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $MyInvocation.MyCommand.Definition
Set-Location $root

Write-Host "[build] working directory: $PWD"

if (-Not (Test-Path go.mod)) {
    Write-Host "go.mod not found — initializing module 'yaddns'"
    go mod init yaddns
}

if (-Not (Test-Path bin)) { New-Item -ItemType Directory -Path bin | Out-Null }

Write-Host "[build] running go fmt"
gofmt -w . || Write-Host "gofmt skipped or not available"

Write-Host "[build] running go build"
go build -o bin/icmp_ddns.exe

if ($LASTEXITCODE -ne 0) {
    Write-Error "Build failed (exit code $LASTEXITCODE)"
    exit $LASTEXITCODE
}

Write-Host "Build succeeded: $pwd\bin\icmp_ddns.exe"
