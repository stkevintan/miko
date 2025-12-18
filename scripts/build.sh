#!/bin/bash

# Cross-platform build script for Miko Go service

set -e

# Build configuration
APP_NAME="miko"
VERSION=${VERSION:-"dev"}
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS="-s -w -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.gitCommit=${GIT_COMMIT}"

# Well-known OS/ARCH combinations
PLATFORMS=(
    "linux/amd64"
    "linux/arm64" 
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/arm64"
    "freebsd/amd64"
    "freebsd/arm64"
)

echo "Building ${APP_NAME} for multiple platforms..."
echo "Version: ${VERSION}"
echo "Build Time: ${BUILD_TIME}"
echo "Git Commit: ${GIT_COMMIT}"
echo ""

# Create bin directory
mkdir -p bin

# Function to build for a specific platform
build_platform() {
    local platform=$1
    local os=$(echo $platform | cut -d'/' -f1)
    local arch=$(echo $platform | cut -d'/' -f2)
    
    local output_name="${APP_NAME}-${os}-${arch}"
    if [ "$os" = "windows" ]; then
        output_name="${output_name}.exe"
    fi
    
    local output_path="bin/${output_name}"
    
    echo "Building for ${os}/${arch}..."
    
    GOOS=$os GOARCH=$arch go build \
        -ldflags="${LDFLAGS}" \
        -o "${output_path}" \
        main.go
    
    if [ $? -eq 0 ]; then
        echo "✅ Successfully built: ${output_path}"
        
        # Make executable on Unix-like systems
        if [ "$os" != "windows" ]; then
            chmod +x "${output_path}"
        fi
        
        # Show file size
        local size=$(ls -lh "${output_path}" | awk '{print $5}')
        echo "   Size: ${size}"
    else
        echo "❌ Failed to build for ${os}/${arch}"
        return 1
    fi
    
    echo ""
}

# Build for all platforms or specific ones
if [ $# -eq 0 ]; then
    # Build for all platforms
    echo "Building for all supported platforms:"
    for platform in "${PLATFORMS[@]}"; do
        build_platform "$platform"
    done
else
    # Build for specific platforms
    echo "Building for specified platforms:"
    for platform in "$@"; do
        # Validate platform format
        if [[ ! "$platform" =~ ^[a-z]+/[a-z0-9]+$ ]]; then
            echo "❌ Invalid platform format: $platform (expected: os/arch)"
            echo "   Examples: linux/amd64, darwin/arm64, windows/amd64"
            continue
        fi
        build_platform "$platform"
    done
fi

echo "Build complete! Binaries are in the bin/ directory:"
ls -la bin/

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

echo ""
echo "Usage examples:"
echo "  Build all platforms:     ./scripts/build.sh"
echo "  Build specific platform: ./scripts/build.sh linux/amd64"
echo "  Build multiple:          ./scripts/build.sh linux/amd64 darwin/arm64"