@echo off
cd /d %~dp0

echo [build] working directory: %cd%

if not exist go.mod (
  echo go.mod not found - initializing module 'yaddns'
  go mod init yaddns
)

if not exist bin mkdir bin

echo [build] running go fmt (if available)
gofmt -w . 2>nul || (echo gofmt not available or skipped)

echo [build] running go build
go build -o bin\icmp_ddns.exe
if errorlevel 1 (
  echo Build failed
  exit /b 1
)

echo Build succeeded: %cd%\bin\icmp_ddns.exe
