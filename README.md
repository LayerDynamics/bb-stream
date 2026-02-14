# BB-Stream

A comprehensive Backblaze B2 cloud storage streaming application with CLI, HTTP API, and desktop GUI.

## Features

- **Streaming uploads/downloads** with real-time progress tracking
- **Bidirectional sync** between local directories and B2 buckets
- **Watch mode** for automatic file uploads on change
- **Live Read support** for reading files while they upload
- **Desktop app** with drag-and-drop interface
- **WebSocket events** for real-time notifications

## Installation

### CLI (Go)

```bash
# From source
go install github.com/ryanoboyle/bb-stream/cmd/bb-stream@latest

# Or build locally
git clone https://github.com/ryanoboyle/bb-stream
cd bb-stream
go build -o bb-stream ./cmd/bb-stream/
```

### Desktop App

Download the latest release for your platform from the [Releases](https://github.com/ryanoboyle/bb-stream/releases) page.

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
| GET | `/api/ws` | WebSocket events |

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

## Development

### Prerequisites

- Go 1.21+
- Node.js 18+
- Rust (for Tauri desktop app)

### Building

```bash
# Build CLI
go build -o bb-stream ./cmd/bb-stream/

# Build and run desktop app
cd desktop
npm install
npm run tauri dev
```

### Testing

```bash
# Run Go tests
go test ./...

# Check Svelte types
cd desktop && npm run check
```

### Project Structure

```
bb-stream/
├── cmd/bb-stream/          # CLI entrypoint
├── internal/
│   ├── api/                # HTTP API server
│   ├── b2/                 # B2 client wrapper
│   ├── config/             # Configuration management
│   ├── sync/               # Sync logic
│   └── watch/              # File watcher
├── pkg/progress/           # Progress utilities
└── desktop/                # Tauri + Svelte app
    ├── src/                # Svelte frontend
    └── src-tauri/          # Rust backend
```

## License

MIT

## Credits

- [Blazer](https://github.com/Backblaze/blazer) - Official B2 Go SDK
- [Tauri](https://tauri.app/) - Desktop framework
- [Svelte](https://svelte.dev/) - UI framework
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Chi](https://github.com/go-chi/chi) - HTTP router
