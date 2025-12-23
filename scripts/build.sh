#!/bin/bash

# Cross-platform build script for Miko Go service

set -e

# Parse command line arguments for build mode
BUILD_MODE="dev"
PLATFORMS_TO_BUILD=()

while [[ $# -gt 0 ]]; do
    case $1 in
        --release)
            BUILD_MODE="release"
            shift
            ;;
        --dev|--development)
            BUILD_MODE="dev"
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [--release|--dev] [platform...]"
            echo ""
            echo "Build modes:"
            echo "  --dev        Development build (default)"
            echo "  --release    Release build with full optimizations"
            echo ""
            echo "Examples:"
            echo "  $0                           # Dev build for all platforms"
            echo "  $0 --release                 # Release build for all platforms"
            echo "  $0 --release linux/amd64     # Release build for Linux x64 only"
            echo "  $0 darwin/arm64 windows/amd64 # Dev build for specific platforms"
            exit 0
            ;;
        *)
            PLATFORMS_TO_BUILD+=("$1")
            shift
            ;;
    esac
done

# Build configuration
APP_NAME="miko"
VERSION=${VERSION:-"dev"}
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags based on mode
if [ "$BUILD_MODE" = "release" ]; then
    # Release mode: aggressive optimizations
    LDFLAGS="-s -w -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.gitCommit=${GIT_COMMIT}"
    BUILD_FLAGS="-a -installsuffix cgo -trimpath"
    CGO_ENABLED=0
    echo "🚀 Release build mode: Full optimizations enabled"
else
    # Development mode: faster builds, debugging symbols
    LDFLAGS="-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.gitCommit=${GIT_COMMIT}"
    BUILD_FLAGS="-trimpath"
    CGO_ENABLED=0
    echo "🔧 Development build mode: Debug symbols preserved"
fi

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
echo "Build Mode: ${BUILD_MODE}"
echo ""

# Create bin directory
mkdir -p bin

# Function to build for a specific platform
build_platform() {
    local platform=$1
    local os=$(echo $platform | cut -d'/' -f1)
    local arch=$(echo $platform | cut -d'/' -f2)
    
    local output_name="${APP_NAME}-${os}-${arch}"
    if [ "$BUILD_MODE" = "release" ]; then
        output_name="${APP_NAME}-${os}-${arch}-${VERSION}"
    fi
    if [ "$os" = "windows" ]; then
        output_name="${output_name}.exe"
    fi
    
    local output_path="bin/${output_name}"
    
    echo "Building for ${os}/${arch}..."
    
    CGO_ENABLED=$CGO_ENABLED GOOS=$os GOARCH=$arch go build \
        $BUILD_FLAGS \
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
if [ ${#PLATFORMS_TO_BUILD[@]} -eq 0 ]; then
    # Build for all platforms
    echo "Building for all supported platforms:"
    for platform in "${PLATFORMS[@]}"; do
        build_platform "$platform"
    done
else
    # Build for specific platforms
    echo "Building for specified platforms:"
    for platform in "${PLATFORMS_TO_BUILD[@]}"; do
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

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"

# call swagger.sh in the same directory as build.sh
bash "$SCRIPT_DIR/swagger.sh"

echo ""
echo "Usage examples:"
echo "  Build all platforms (dev):       ./scripts/build.sh"
echo "  Build all platforms (release):   ./scripts/build.sh --release"
echo "  Build specific platform (dev):   ./scripts/build.sh linux/amd64"
echo "  Build specific platform (release): ./scripts/build.sh --release linux/amd64"
echo "  Build multiple platforms:        ./scripts/build.sh --release linux/amd64 darwin/arm64"
echo ""
if [ "$BUILD_MODE" = "release" ]; then
    echo "🎉 Release build completed! Optimized binaries ready for production."
else
    echo "🎯 Development build completed! Debug symbols preserved for development."
fi