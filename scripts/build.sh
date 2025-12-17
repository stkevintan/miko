#!/bin/bash

# Build script for the Go service

set -e

echo "Building Miko Go service..."

# Create bin directory if it doesn't exist
mkdir -p bin

# Build the binary
go build -o bin/miko -ldflags="-s -w" main.go

echo "Build complete! Binary created at bin/miko"

# Make it executable
chmod +x bin/miko

echo "You can run the service with: ./bin/miko"