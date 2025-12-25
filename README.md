# Miko 
- A music downloader (mainly for Chinese users)
- A subsonic 1.16.1 server implementation

## Features

- **Gin HTTP framework** for high-performance routing and middleware
- **Dependency Injection** using `samber/do/v2` for clean architecture
- **Request-scoped isolation** for user-specific resources (e.g., CookieJars)
- **CookieCloud Integration** for seamless authentication across devices
- **OpenAPI 3.0 documentation with Swagger UI**
- **All API endpoints under `/api/` path**
- **CORS support** for cross-origin requests
- **JSON validation** with automatic error handling
- Configuration management via environment variables and TOML
- Graceful shutdown handling
- Comprehensive testing
- Docker support

## Project Structure

```
.
├── main.go                 # Application entry point
├── config/
│   ├── config.go          # Configuration management
│   └── config.toml        # Default configuration
├── pkg/
│   ├── cookiecloud/       # CookieCloud client and identity management
│   ├── log/               # Structured logging
│   ├── netease/           # NetEase music provider implementation
│   ├── provider/          # Generic music provider interface
│   └── types/             # Common data types
├── server/
│   ├── api/               # HTTP API handlers
│   ├── models/            # API data models
│   └── subsonic/          # Subsonic API implementation
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
   # User login
   curl -X POST http://localhost:8082/api/login \
        -H "Content-Type: application/json" \
        -d '{"username": "admin", "password": "adminpassword"}'
   
   # Update CookieCloud identity (requires token)
   curl -X POST http://localhost:8082/api/cookiecloud/identity \
        -H "Authorization: Bearer <token>" \
        -H "Content-Type: application/json" \
        -d '{"key": "your-uuid", "password": "your-password"}'
   
   # Download music (requires token)
   curl -X GET "http://localhost:8082/api/download?uri=https://music.163.com/song?id=2161154646&output=./songs" \
        -H "Authorization: Bearer <token>"
   
   # View OpenAPI documentation
   open http://localhost:8082/swagger/index.html
   ```

## Configuration

Miko loads configuration in this order (later sources override earlier ones):

- Built-in defaults (`config/config.toml`)
- Optional config file (`./config.toml`, `./config/config.toml`, or `$HOME/.miko/config.toml`)
- Environment variables (`MIKO_*`)

### Config file

By default, Miko looks for a `config` file in the following locations (first match wins):

- `./config.toml`
- `./config/config.toml`
- `$HOME/.miko/config.toml`

You can also point directly to a config file path via `MIKO_CONFIG` environment variable.
Paths in the config file (like database DSN or log file) support environment variable expansion (e.g., `${HOME}/.miko/miko.db`).

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

### Authentication  
- **POST** `/api/login` - User authentication
  - Request body: `{"username": "admin", "password": "password"}`
  - Response: `{"token": "jwt-token"}`

### CookieCloud
- **GET** `/api/cookiecloud/server` - Get CookieCloud server URL
- **POST** `/api/cookiecloud/identity` - Update CookieCloud identity (key and password)
  - Request body: `{"key": "your-uuid", "password": "your-password"}`
- **POST** `/api/cookiecloud/pull` - Force pull cookies from CookieCloud

### Music Download
- **GET** `/api/download` - Download music tracks
  - Query parameters:
    - `uri`: Resource URIs (song ID, album URL, etc.)
    - `level`: Audio quality (standard, lossless, hires, etc.)
    - `output`: Output directory
    - `platform`: Music platform (e.g., netease)
  - Response: `{"summary": "..."}`

### Platform
- **GET** `/api/platform/:platform/user` - Get user information for a specific platform

### Documentation
- **GET** `/swagger/index.html` - OpenAPI/Swagger UI documentation
- **GET** `/docs/swagger.json` - OpenAPI JSON specification
- **GET** `/docs/swagger.yaml` - OpenAPI YAML specification

## License

MIT License