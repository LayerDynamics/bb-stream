# BB-Stream

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A Backblaze B2 cloud storage streaming application with CLI, HTTP API, and desktop GUI.

## Features

- **Streaming uploads/downloads** with real-time progress tracking
- **Bidirectional sync** between local directories and B2 buckets
- **Watch mode** for automatic file uploads on change
- **Live Read support** for reading files while they upload
- **Desktop app** with drag-and-drop interface
- **WebSocket events** for real-time notifications
- **Production hardening** with security headers, path validation, and graceful shutdown

## Installation

### CLI (Go)

```bash
# From source
go install github.com/LayerDynamics/bb-stream/cmd/bb-stream@latest

# Or build locally
git clone https://github.com/LayerDynamics/bb-stream
cd bb-stream
go build -o bb-stream ./cmd/bb-stream/
```

### Desktop App

Download the latest release for your platform from the [Releases](https://github.com/LayerDynamics/bb-stream/releases) page.

Or build from source:

```bash
cd desktop
npm install
npm run tauri build
```

## Quick Start

### 1. Configure credentials

```bash
bb-stream config init
# Enter your B2 Key ID and Application Key
```

Or use environment variables:

```bash
export BB_KEY_ID=your-key-id
export BB_APP_KEY=your-application-key
export BB_DEFAULT_BUCKET=your-bucket
```

### 2. Basic operations

```bash
# List buckets
bb-stream ls

# List files in a bucket
bb-stream ls mybucket

# Upload a file
bb-stream upload ./file.txt mybucket/path/file.txt

# Download a file
bb-stream download mybucket/path/file.txt ./downloaded.txt

# Delete a file
bb-stream rm mybucket/path/file.txt
```

### 3. Streaming

```bash
# Stream from stdin
cat large-file.bin | bb-stream stream-up mybucket/large-file.bin

# Stream to stdout
bb-stream stream-down mybucket/large-file.bin > output.bin
```

### 4. Sync

```bash
# Sync local folder to B2
bb-stream sync ./local-folder mybucket/backup --to-remote

# Sync B2 to local folder
bb-stream sync ./local-folder mybucket/backup --to-local

# Dry run (preview changes)
bb-stream sync ./local-folder mybucket/backup --to-remote --dry-run
```

### 5. Watch mode

```bash
# Auto-upload files on change
bb-stream watch ./watched-folder mybucket/uploads
```

### 6. API server

```bash
# Start the HTTP API server
bb-stream serve --port 8765

# With version flag
bb-stream --version
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `config init` | Initialize configuration interactively |
| `config show` | Show current configuration |
| `ls [bucket] [path]` | List buckets or files |
| `upload <file> <bucket/path>` | Upload a file |
| `download <bucket/path> <file>` | Download a file |
| `rm <bucket/path>` | Delete a file |
| `stream-up <bucket/path>` | Stream stdin to B2 |
| `stream-down <bucket/path>` | Stream B2 file to stdout |
| `sync <source> <dest>` | Sync directory with bucket |
| `watch <local> <bucket/path>` | Watch directory for changes |
| `serve [--port]` | Start HTTP API server |

## API Endpoints

When running in server mode (`bb-stream serve`):

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/api/version` | Get version info |
| GET | `/api/status` | Server status and stats |
| POST | `/api/auth` | Validate credentials |
| GET | `/api/buckets` | List buckets |
| GET | `/api/buckets/{name}/files` | List files |
| POST | `/api/upload` | Upload file (multipart) |
| POST | `/api/upload/stream` | Stream upload |
| GET | `/api/download/{bucket}/{path}` | Download file |
| GET | `/api/stream/{bucket}/{path}` | Stream download |
| DELETE | `/api/delete/{bucket}/{path}` | Delete file |
| POST | `/api/sync/start` | Start sync job |
| GET | `/api/sync/status/{id}` | Get sync status |
| POST | `/api/watch/start` | Start watch job |
| POST | `/api/watch/stop` | Stop watch job |
| GET | `/api/jobs` | List active jobs |
| GET | `/api/config` | Get/set configuration |
| GET | `/api/ws` | WebSocket events |

## Desktop App

The desktop application provides a full GUI experience:

- **Bucket browser** with drag-and-drop upload
- **Real-time progress** for uploads and downloads
- **Sync management** with visual status
- **Watch job control** panel
- **Backend health monitoring** with auto-recovery
- **Dynamic port allocation** for conflict-free operation

### Building the Desktop App

```bash
cd desktop
npm install
npm run tauri dev    # Development mode
npm run tauri build  # Production build
```

## Configuration

Config file location: `~/.config/bb-stream/config.yaml`

```yaml
key_id: your-b2-key-id
application_key: your-b2-application-key
default_bucket: your-default-bucket
api_port: 8765
api_key: optional-api-key-for-auth
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `BB_KEY_ID` | B2 Key ID |
| `BB_APP_KEY` | B2 Application Key |
| `BB_DEFAULT_BUCKET` | Default bucket name |
| `BB_API_KEY` | API authentication key |

## Security

BB-Stream includes production security features:

- **Path traversal protection** - Validates all file paths to prevent directory escape
- **Security headers** - X-Frame-Options, CSP, X-Content-Type-Options, etc.
- **Input validation** - Bucket names and object paths are sanitized
- **Error sanitization** - Internal errors are not exposed to clients

## Architecture

```
bb-stream/
├── cmd/bb-stream/          # CLI entrypoint
├── internal/
│   ├── api/                # HTTP API server
│   │   ├── handlers.go     # Request handlers
│   │   ├── server.go       # Server setup
│   │   ├── middleware.go   # Auth & security middleware
│   │   └── websocket.go    # WebSocket hub
│   ├── b2/                 # B2 client wrapper
│   ├── config/             # Configuration management
│   ├── sync/               # Bidirectional sync logic
│   └── watch/              # File system watcher
├── pkg/
│   ├── progress/           # Progress callback utilities
│   ├── logging/            # Structured logging (slog)
│   ├── errors/             # Error handling & sanitization
│   └── retry/              # Exponential backoff retry
└── desktop/                # Tauri + Svelte app
    ├── src/                # Svelte 5 frontend
    │   ├── lib/
    │   │   ├── api.ts      # API client
    │   │   ├── websocket.ts # WebSocket client
    │   │   └── components/ # UI components
    │   └── App.svelte      # Main app
    └── src-tauri/          # Rust backend
        └── src/lib.rs      # Sidecar management
```

## Development

### Prerequisites

- Go 1.21+
- Node.js 18+
- Rust 1.70+ (for Tauri desktop app)

### Building

```bash
# Build CLI
go build -o bb-stream ./cmd/bb-stream/

# Run tests
go test ./...

# Build desktop app
cd desktop
npm install
npm run tauri dev
```

### Testing

```bash
# Run all Go tests
go test ./...

# Run with coverage
go test -cover ./...

# Check Svelte types
cd desktop && npm run check
```

## WebSocket Events

Subscribe to real-time events via WebSocket at `/api/ws`:

| Event Type | Description |
|------------|-------------|
| `upload_progress` | Upload progress updates |
| `download_progress` | Download progress updates |
| `sync_progress` | Sync job progress |
| `sync_complete` | Sync job completed |
| `watch_event` | File change detected |
| `error` | Error notification |

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

MIT

## Credits

- [Blazer](https://github.com/Backblaze/blazer) - Official B2 Go SDK
- [Tauri](https://tauri.app/) - Desktop framework
- [Svelte](https://svelte.dev/) - UI framework
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Chi](https://github.com/go-chi/chi) - HTTP router
