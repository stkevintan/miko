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
   
   # User login (requires NetEase account)
   curl -X POST http://localhost:8080/api/login \
        -H "Content-Type: application/json" \
        -d '{"uuid": "your_username", "password": "your_password"}'
   
   # Download music (requires login)
   curl -X POST http://localhost:8080/api/download \
        -H "Content-Type: application/json" \
        -d '{"song_id": "2161154646", "level": "lossless"}'
   
   # Process data
   curl -X POST http://localhost:8080/api/process \
        -H "Content-Type: application/json" \
        -d '{"data": "hello world"}'
   
   # View OpenAPI documentation
   open http://localhost:8080/swagger/index.html
   ```

## Configuration

Miko loads configuration in this order (later sources override earlier ones):

- Built-in defaults
- Optional config file
- Environment variables (`MIKO_*`)
- Legacy environment variables (`PORT`, `ENVIRONMENT`, `LOG_LEVEL`)

### Config file

By default, Miko looks for a `config` file in the following locations (first match wins):

- `./config.yaml` (also supports `yml`, `json`, `toml`)
- `./config/config.yaml`
- `$HOME/.miko/config.yaml`

You can also point directly to a config file path via:

- `MIKO_CONFIG=/path/to/config.yaml`

### Environment variables

All config keys can be set via environment variables using the `MIKO_` prefix.
Nested keys use `_` separators.

Examples:

- `MIKO_PORT=8082`
- `MIKO_ENVIRONMENT=development`
- `MIKO_LOG_LEVEL=info`
- `MIKO_NMAPI_DEBUG=false`
- `MIKO_NMAPI_COOKIE_FILEPATH=$HOME/.miko/cookie.json`

Legacy env vars are still supported for backward compatibility:

- `PORT`, `ENVIRONMENT`, `LOG_LEVEL`

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

### Authentication  
- **POST** `/api/login` - User authentication with NetEase Cloud Music
  - Request body: `{"uuid": "user_id", "password": "password", "timeout": 30000, "server": "cookiecloud_url"}`
  - Response: `{"username": "user", "user_id": 123456, "success": true}`

### Music Download
- **POST** `/api/download` - Get download URL and metadata for a song
  - Request body: `{"song_id": "123456", "level": "lossless", "output": "./downloads", "timeout": 30000}`
  - Response: `{"download_url": "https://...", "song_name": "title", "artist": "artist", "quality": "lossless"}`

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