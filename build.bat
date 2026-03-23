@echo off
REM Build script for Kiddo Windows Service

setlocal enabledelayedexpansion

echo ========================================
echo Kiddo Service Build Script
echo ========================================

REM Check if Go is installed
go version >nul 2>&1
if errorlevel 1 (
    echo Error: Go is not installed or not in PATH
    exit /b 1
)

echo.
echo [1/3] Downloading dependencies...
go mod download
if errorlevel 1 (
    echo Error: Failed to download dependencies
    exit /b 1
)

echo.
echo [2/3] Running tests...
go test ./...
if errorlevel 1 (
    echo Warning: Some tests failed
)

echo.
echo [3/3] Building binary...
set OUTPUT=bin\kiddo.exe
if not exist bin mkdir bin

go build -o %OUTPUT% -ldflags="-s -w" .
if errorlevel 1 (
    echo Error: Build failed
    exit /b 1
)

echo.
echo ========================================
echo Build successful!
echo Output: %OUTPUT%
echo.
echo Next steps:
echo   1. Copy config.example.json to C:\ProgramData\Kiddo\config.json
echo   2. Edit config.json with your GitHub repository details
echo   3. Set KIDDO_GITHUB_TOKEN environment variable
echo   4. Run: %OUTPUT% install
echo   5. Run: net start Kiddo
echo ========================================
