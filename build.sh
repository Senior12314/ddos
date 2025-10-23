#!/bin/bash

# CloudNordSP Build Script
set -e

echo "🚀 Building CloudNordSP Minecraft DDoS Protection Platform"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root for XDP operations
check_root() {
    if [[ $EUID -eq 0 ]]; then
        print_warning "Running as root - XDP operations will be available"
    else
        print_warning "Not running as root - XDP operations will be limited"
    fi
}

# Build XDP program
build_xdp() {
    print_status "Building XDP eBPF program..."
    
    if ! command -v clang &> /dev/null; then
        print_error "clang not found. Please install clang and llvm"
        exit 1
    fi
    
    if ! command -v bpftool &> /dev/null; then
        print_error "bpftool not found. Please install bpftool"
        exit 1
    fi
    
    make clean
    make
    
    if [ -f "minecraft_protection.o" ]; then
        print_success "XDP program built successfully"
    else
        print_error "Failed to build XDP program"
        exit 1
    fi
}

# Build Go control plane
build_control_plane() {
    print_status "Building Go control plane..."
    
    if ! command -v go &> /dev/null; then
        print_error "Go not found. Please install Go 1.21+"
        exit 1
    fi
    
    cd cmd/control-plane
    go mod tidy
    go build -o ../../control-plane .
    cd ../..
    
    if [ -f "control-plane" ]; then
        print_success "Control plane built successfully"
    else
        print_error "Failed to build control plane"
        exit 1
    fi
}

# Build React web UI
build_web() {
    print_status "Building React web UI..."
    
    if ! command -v npm &> /dev/null; then
        print_error "npm not found. Please install Node.js 18+"
        exit 1
    fi
    
    cd web
    npm install
    npm run build
    cd ..
    
    if [ -d "web/build" ]; then
        print_success "Web UI built successfully"
    else
        print_error "Failed to build web UI"
        exit 1
    fi
}

# Build Docker images
build_docker() {
    print_status "Building Docker images..."
    
    if ! command -v docker &> /dev/null; then
        print_error "Docker not found. Please install Docker"
        exit 1
    fi
    
    # Build control plane image
    docker build -f Dockerfile.control-plane -t cloudnordsp/control-plane:latest .
    
    # Build web image
    docker build -f Dockerfile.web -t cloudnordsp/web:latest .
    
    print_success "Docker images built successfully"
}

# Run tests
run_tests() {
    print_status "Running tests..."
    
    # Go tests
    cd cmd/control-plane
    go test ./...
    cd ../..
    
    # React tests
    cd web
    npm test -- --watchAll=false
    cd ..
    
    print_success "All tests passed"
}

# Main build process
main() {
    print_status "Starting CloudNordSP build process..."
    
    check_root
    
    # Build components
    build_xdp
    build_control_plane
    build_web
    
    # Build Docker images if requested
    if [ "$1" = "--docker" ]; then
        build_docker
    fi
    
    # Run tests if requested
    if [ "$1" = "--test" ] || [ "$2" = "--test" ]; then
        run_tests
    fi
    
    print_success "Build completed successfully!"
    print_status "Next steps:"
    echo "  1. Load XDP program: sudo ./loader eth0 load minecraft_protection.o"
    echo "  2. Start control plane: ./control-plane -config config.yaml"
    echo "  3. Start web UI: cd web && npm start"
    echo "  4. Or use Docker: docker-compose up -d"
}

# Handle command line arguments
case "$1" in
    --help|-h)
        echo "CloudNordSP Build Script"
        echo "Usage: $0 [options]"
        echo ""
        echo "Options:"
        echo "  --docker    Build Docker images"
        echo "  --test      Run tests"
        echo "  --help      Show this help message"
        echo ""
        echo "Examples:"
        echo "  $0                    # Build all components"
        echo "  $0 --docker          # Build all components and Docker images"
        echo "  $0 --test            # Build all components and run tests"
        echo "  $0 --docker --test   # Build all components, Docker images, and run tests"
        ;;
    *)
        main "$@"
        ;;
esac
