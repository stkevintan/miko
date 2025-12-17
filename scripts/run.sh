#!/bin/bash

# Run script for the Go service

set -e

echo "Starting Miko Go service..."

# Set default environment variables if not already set
export PORT=${PORT:-8080}
export ENVIRONMENT=${ENVIRONMENT:-development}
export LOG_LEVEL=${LOG_LEVEL:-info}

echo "Configuration:"
echo "  Port: $PORT"
echo "  Environment: $ENVIRONMENT"
echo "  Log Level: $LOG_LEVEL"
echo ""

# Run the service
go run main.go