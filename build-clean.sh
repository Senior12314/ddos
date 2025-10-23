#!/bin/bash

echo "Cleaning Docker cache and building CloudNordSP..."
echo

echo "Stopping any running containers..."
docker-compose down

echo
echo "Removing old images..."
docker-compose down --rmi all

echo
echo "Cleaning Docker build cache..."
docker builder prune -f

echo
echo "Building with no cache..."
docker-compose build --no-cache

echo
echo "Starting services..."
docker-compose up -d

echo
echo "Build complete! Access the dashboard at: http://localhost"
echo "API available at: http://localhost:8080"
echo "Prometheus at: http://localhost:9090"
echo "Grafana at: http://localhost:3000 (admin/admin)"
echo
