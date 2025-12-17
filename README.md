# Miko Go Service

A Go service library built with **Gin HTTP framework**, featuring OpenAPI-documented endpoints, configuration management, and graceful shutdown.

## Features

- **Gin HTTP framework** for high-performance routing and middleware
- Clean architecture with separation of concerns
- **OpenAPI 3.0 documentation with Swagger UI**
- **All API endpoints under `/api/` path**
- **CORS support** for cross-origin requests
- **JSON validation** with automatic error handling
- Configuration management via environment variables
- HTTP API with health check endpoint
- Graceful shutdown handling
- Comprehensive testing
- Docker support

## Project Structure

```
.
├── main.go                 # Application entry point
├── main_test.go           # Integration tests
├── docs/                   # Generated OpenAPI documentation
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
├── internal/
│   ├── config/
│   │   └── config.go      # Configuration management
│   ├── models/
│   │   └── models.go      # API data models
│   ├── service/
│   │   └── service.go     # Business logic
│   └── handler/
│       ├── handler.go     # HTTP handlers with OpenAPI annotations
│       └── handler_test.go # Handler tests
├── docker/
│   └── Dockerfile         # Docker configuration
├── scripts/
│   ├── build.sh          # Build script
│   └── run.sh            # Run script
└── README.md
```

## Quick Start

1. **Install dependencies:**
   ```bash
   go mod tidy
   ```

2. **Run the service:**
   ```bash
   go run main.go
   ```

3. **Test the service:**
   ```bash
   # Health check
   curl http://localhost:8080/api/health
   
   # Process data
   curl -X POST http://localhost:8080/api/process \
        -H "Content-Type: application/json" \
        -d '{"data": "hello world"}'
   
   # View OpenAPI documentation
   open http://localhost:8080/swagger/index.html
   ```

## Configuration

The service can be configured using environment variables:

- `PORT`: Server port (default: 8080)
- `ENVIRONMENT`: Environment name (default: development)
- `LOG_LEVEL`: Log level (default: info)

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o bin/miko main.go
```

### Generate OpenAPI Documentation

```bash
# Install swag if not already installed
go install github.com/swaggo/swag/cmd/swag@latest

# Generate docs
~/go/bin/swag init
```

### Using Docker

```bash
# Build image
docker build -f docker/Dockerfile -t miko .

# Run container
docker run -p 8080:8080 miko
```

## API Endpoints

All API endpoints are under the `/api/` path as per OpenAPI best practices.

### Health Check
- **GET** `/api/health` - Returns service health status

### Process Data  
- **POST** `/api/process` - Processes input data
  - Request body: `{"data": "string"}`
  - Response: `{"result": "processed: string"}`

### Documentation
- **GET** `/swagger/index.html` - OpenAPI/Swagger UI documentation
- **GET** `/docs/swagger.json` - OpenAPI JSON specification
- **GET** `/docs/swagger.yaml` - OpenAPI YAML specification

## License

MIT License