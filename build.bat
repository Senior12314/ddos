@echo off
setlocal enabledelayedexpansion

echo 🚀 Building CloudNordSP Minecraft DDoS Protection Platform

REM Check if Go is installed
go version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Go not found. Please install Go 1.21+
    exit /b 1
)

REM Check if Node.js is installed
node --version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Node.js not found. Please install Node.js 18+
    exit /b 1
)

REM Check if Docker is installed
docker --version >nul 2>&1
if errorlevel 1 (
    echo [WARNING] Docker not found. Docker builds will be skipped
    set DOCKER_AVAILABLE=0
) else (
    set DOCKER_AVAILABLE=1
)

echo [INFO] Building Go control plane...
cd cmd\control-plane
go mod tidy
go build -o ..\..\control-plane.exe .
cd ..\..

if exist "control-plane.exe" (
    echo [SUCCESS] Control plane built successfully
) else (
    echo [ERROR] Failed to build control plane
    exit /b 1
)

echo [INFO] Building React web UI...
cd web
call npm install
if errorlevel 1 (
    echo [ERROR] Failed to install npm dependencies
    exit /b 1
)

call npm run build
if errorlevel 1 (
    echo [ERROR] Failed to build web UI
    exit /b 1
)
cd ..

if exist "web\build" (
    echo [SUCCESS] Web UI built successfully
) else (
    echo [ERROR] Failed to build web UI
    exit /b 1
)

REM Build Docker images if Docker is available
if %DOCKER_AVAILABLE%==1 (
    echo [INFO] Building Docker images...
    docker build -f Dockerfile.control-plane -t cloudnordsp/control-plane:latest .
    docker build -f Dockerfile.web -t cloudnordsp/web:latest .
    echo [SUCCESS] Docker images built successfully
)

echo [SUCCESS] Build completed successfully!
echo [INFO] Next steps:
echo   1. Start control plane: control-plane.exe -config config.yaml
echo   2. Start web UI: cd web ^&^& npm start
echo   3. Or use Docker: docker-compose up -d

pause
