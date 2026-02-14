# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run Commands

```bash
# Go CLI
go build ./cmd/bb-stream/        # Build CLI
go test ./...                    # Run all tests
go test -v ./internal/sync/...   # Run specific package tests
go test -race ./...              # Run with race detector

# API Server
./bb-stream serve --port 8765    # Default port
pkill -f "bb-stream"; lsof -ti:8765 | xargs kill -9  # Kill server/port

# Desktop (Tauri + Svelte)
cd desktop
npm install
npm run tauri dev                # Development mode
npm run tauri build              # Production build
npm run check                    # Type-check Svelte
```

## Architecture

```
bb_stream/
├── cmd/bb-stream/      # CLI entrypoint (Cobra commands, all commands in main.go)
├── internal/
│   ├── api/            # HTTP API (Chi router, WebSocket hub)
│   │   ├── server.go   # Route setup and server lifecycle
│   │   ├── handlers.go # All HTTP handlers
│   │   └── websocket.go# WebSocket hub for real-time events
│   ├── b2/             # Blazer B2 client wrapper
│   │   ├── client.go   # Core client, bucket/object operations
│   │   ├── upload.go   # Upload with progress
│   │   ├── download.go # Download with progress
│   │   └── liveread.go # Read files while uploading
│   ├── config/         # Viper config (~/.config/bb-stream/config.yaml)
│   ├── sync/           # Bidirectional sync
│   │   ├── sync.go     # Syncer and ConcurrentSyncer
│   │   └── diff.go     # File comparison logic
│   └── watch/          # fsnotify watcher with debouncing
├── pkg/progress/       # progress.Callback type for transfers
└── desktop/
    ├── src/            # Svelte 5 frontend
    │   ├── lib/api.ts  # API client for backend
    │   └── lib/stores/ # Svelte stores (files, jobs, toasts)
    └── src-tauri/      # Rust shell
```

## Key Implementation Details

**API Handlers**: Use `context.Background()` for background goroutines, not request context (which gets cancelled).

**B2 Delete**: Requires listing file versions first - see `DeleteObject` in `internal/b2/client.go:161`.

**Sync**: Uses 1-second tolerance for file time comparisons (B2 timestamp precision). Default ignores: `.git`, `.DS_Store`, `node_modules`, `__pycache__`.

**Progress Callbacks**: All transfer operations accept `progress.Callback func(transferred, total int64)`.

## Environment Variables

- `BB_KEY_ID` - B2 Key ID
- `BB_APP_KEY` - B2 Application Key
- `BB_DEFAULT_BUCKET` - Default bucket
- `BB_API_KEY` - API authentication key

## Frontend (Svelte 5)

Use Svelte 5 runes exclusively:
- `$state()` for reactive variables (NOT `let`)
- `$props()` for component props
- `$derived()` for computed values (NOT `$:`)

DELETE requests in Tauri webview may need XMLHttpRequest instead of fetch.

## Debugging Principles

- Verify fixes work before claiming resolved - test specific reported behavior
- Bug reports are debugging tasks - investigate first, don't ask design questions
- If something is called but missing, implement it - don't remove the call
