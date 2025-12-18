# Docker and CI/CD

This project includes automated CI/CD pipelines for building, testing, and publishing Docker images to GitHub Container Registry.

## Docker Images

Docker images are automatically built and published to GitHub Packages for every push to main/develop branches and for all tagged releases.

### Using Published Images

```bash
# Pull latest image for your architecture
docker pull ghcr.io/kevinclasky/miko:main

# Pull specific version
docker pull ghcr.io/kevinclasky/miko:v1.0.0

# Pull specific architecture
docker pull ghcr.io/kevinclasky/miko:main-linux-amd64
docker pull ghcr.io/kevinclasky/miko:main-linux-arm64
```

### Running the Container

```bash
# Basic usage
docker run -p 8082:8082 ghcr.io/kevinclasky/miko:main

# With environment variables
docker run -p 8082:8082 \
  -e ENV=production \
  -e PORT=8082 \
  ghcr.io/kevinclasky/miko:main

# With volume mount for configuration
docker run -p 8082:8082 \
  -v $(pwd)/config:/app/config \
  ghcr.io/kevinclasky/miko:main
```

### Building Locally

```bash
# Build using build.sh (recommended)
./scripts/build.sh --release linux/amd64
docker build -f docker/Dockerfile -t miko:local .

# Or build directly with Docker
docker build -f docker/Dockerfile -t miko:local .
```

## CI/CD Workflows

### 1. Build and Test (`ci.yml`)
- Runs on every push and pull request
- Tests code formatting, runs unit tests, and builds for multiple platforms
- Uploads build artifacts for download

### 2. Docker Build and Publish (`docker.yml`)
- Builds multi-architecture Docker images (amd64, arm64)
- Publishes to GitHub Container Registry
- Creates manifest lists for multi-arch support
- Uses release build for tagged versions, dev build for branches

### 3. Release (`release.yml`)
- Triggers on git tags (v*)
- Builds release binaries for all platforms
- Creates GitHub releases with binaries and checksums
- Generates automatic release notes

## Multi-Architecture Support

The Docker images support the following architectures:
- `linux/amd64` - Standard x86_64 Linux
- `linux/arm64` - ARM64 Linux (Apple Silicon, ARM servers)

Docker will automatically pull the correct architecture for your platform.

## Security Features

- Uses minimal Alpine Linux base image
- Runs as non-root user (`miko`)
- Includes health check endpoint
- Static binaries with no external dependencies (CGO_ENABLED=0)

## Container Registry

Images are published to GitHub Container Registry at:
- Registry: `ghcr.io`
- Repository: `ghcr.io/kevinclasky/miko`

### Image Tags

- `main` - Latest build from main branch
- `develop` - Latest build from develop branch  
- `v1.0.0` - Specific version tags
- `v1.0` - Major.minor version
- `v1` - Major version
- `main-1234567` - Git SHA tags
- Architecture-specific tags with `-linux-amd64` or `-linux-arm64` suffix

## Environment Variables

The container supports the following environment variables:

- `ENV` - Environment mode (development, production)
- `PORT` - Server port (default: 8082)
- `LOG_LEVEL` - Logging level (debug, info, warn, error)

## Development

For local development with Docker:

```bash
# Build development image
./scripts/build.sh --dev linux/amd64
docker build -f docker/Dockerfile -t miko:dev .

# Run with live reload (requires volume mount)
docker run -p 8082:8082 -v $(pwd):/app miko:dev
```