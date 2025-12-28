# Miko 

Miko is a lightweight, high-performance music service that combines a **Subsonic-compatible server** with a powerful **music downloader**. It is designed to be self-hosted, CGO-free, and highly portable.

## Features

- **Subsonic API (v1.16.1)**: Compatible with most Subsonic clients (DSub, Play:Sub, Symfonium, etc.).
- **OpenSubsonic Extensions**: Supports modern extensions like `songLyrics` (synced/unsynced LRC).
- **Music Downloader**: Integrated downloader for music platforms (primarily NetEase Cloud Music).
- **CookieCloud Integration**: Seamlessly sync authentication cookies from your browser to the server.
- **CGO-Free & Portable**: Built with a pure-Go SQLite driver, making it easy to run in lightweight Docker containers (Alpine).
- **Multi-Arch Support**: Native binaries and Docker images for `amd64` and `arm64` (Apple Silicon/Raspberry Pi).
- **Clean Architecture**: Built with Chi, GORM, and Dependency Injection (`samber/do`).
- **Centralized Data Management**: All database, cache, and user data are stored in a single, configurable `dataDir`.

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

All config keys can be set via environment variables using the `MIKO_` prefix. Nested keys use `_` separators.

| Variable | Description | Default |
| :--- | :--- | :--- |
| `MIKO_PORT` | Server port | `8082` |
| `MIKO_SUBSONIC_DATADIR` | Directory for DB, cache, and avatars | `./data` |
| `MIKO_SUBSONIC_FOLDERS` | Comma-separated list of music folders | |
| `MIKO_LOG_LEVEL` | Log level (debug, info, warn, error) | `info` |

## Subsonic Integration

Miko implements the Subsonic REST API, allowing you to use your favorite music clients.

- **Endpoint**: `http://your-ip:8082/rest`
- **Supported Clients**: DSub, Play:Sub, Symfonium, Amperfy, Substreamer, etc.
- **Features**: Browsing, Streaming, Starred, Playlists, Search, and Lyrics.

## API Endpoints (Internal)

In addition to the Subsonic API, Miko provides internal endpoints for management.

### Authentication  
- **POST** `/api/login` - User authentication
- **POST** `/api/cookiecloud/identity` - Update CookieCloud credentials
- **POST** `/api/cookiecloud/pull` - Force sync cookies from CookieCloud

### Music Management
- **GET** `/api/download` - Download music via URI (NetEase, etc.)
- **GET** `/api/platform/:platform/user` - Get platform-specific user info

## Development

### Running Tests

```bash
go test ./...
```

### Building

Miko includes a cross-platform build script that handles optimizations and multi-arch builds.

```bash
# Build for current platform (development)
./scripts/build.sh

# Build for all platforms (release mode with optimizations)
./scripts/build.sh --release

# Build for a specific platform
./scripts/build.sh --release linux/amd64
```

### Using Docker

Miko provides multi-arch Docker images. You can build your own or use the pre-built ones.

```bash
# Build image locally
docker build -f docker/Dockerfile -t miko .

# Run container
# Map a local directory to /data for persistent storage
docker run -p 8082:8082 \
  -v $(pwd)/data:/data \
  -e MIKO_SUBSONIC_DATADIR=/data \
  miko
```

## License

MIT License