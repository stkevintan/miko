#!/bin/bash

# Generate fresh Swagger documentation after successful build
echo ""
echo "🔄 Generating Swagger documentation..."
if command -v swag >/dev/null 2>&1; then
    swag init --parseDependency --parseInternal --output docs
    echo "✅ Swagger documentation generated successfully"
elif [ -f "${HOME}/go/bin/swag" ]; then
    "${HOME}/go/bin/swag" init --parseDependency --parseInternal --output docs
    echo "✅ Swagger documentation generated successfully"
else
    echo "⚠️  Warning: swag tool not found, skipping swagger generation"
    echo "   Install with: go install github.com/swaggo/swag/cmd/swag@latest"
fi
