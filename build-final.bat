@echo off
echo ========================================
echo CloudNordSP - Final Build Script
echo ========================================
echo.

echo [1/6] Stopping any running containers...
docker-compose down

echo.
echo [2/6] Cleaning Docker cache and images...
docker-compose down --rmi all
docker builder prune -f

echo.
echo [3/6] Building all services with no cache...
docker-compose build --no-cache

echo.
echo [4/6] Starting all services...
docker-compose up -d

echo.
echo [5/6] Waiting for services to start...
timeout /t 30 /nobreak > nul

echo.
echo [6/6] Checking service status...
docker-compose ps

echo.
echo ========================================
echo BUILD COMPLETE!
echo ========================================
echo.
echo Access Points:
echo - Web Dashboard: http://localhost
echo - API: http://localhost:8080/api/v1
echo - Health Check: http://localhost:8080/health
echo - Prometheus: http://localhost:9090
echo - Grafana: http://localhost:3000 (admin/admin)
echo.
echo To view logs:
echo - docker-compose logs -f control-plane
echo - docker-compose logs -f web
echo - docker-compose logs -f postgres
echo.
echo To stop services:
echo - docker-compose down
echo.
pause
