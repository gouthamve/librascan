# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

librascan is a Go-based personal library management system with barcode scanning capabilities. It consists of:
- HTTP API server for book management
- CLI barcode scanner integration
- Terminal UI (TUI) for browsing books
- Integration with Google Books API for metadata
- AI enrichment via Perplexity API
- SQLite database with Litestream backup to S3

## Common Development Commands

### Build and Run
```bash
# Build the application
make build

# Run the server (development mode with hot reload)
make dev

# Run the server (production)
./librascan serve --perplexity-key YOUR_KEY

# Run barcode scanner CLI
./librascan read-isbn --server-url http://localhost:8080 --input-device-path /dev/input/eventX

# Run TUI interface
./librascan tui --server-url http://localhost:8080
```

### Development Setup
```bash
# Install development dependencies (reflex for hot reload, gokrazy tools)
make setup-deps

# Download latest database from backup
make download-db

# Update all Go dependencies
make update-deps
```

### Testing
```bash
# Run all tests
make test

# Run specific package tests
go test ./pkg/handlers/...
```

### Docker Operations
```bash
# Build Docker image
make docker-build

# Push to registry
make docker-push
```

## Architecture Overview

### Core Components

1. **cmd/librascan/** - Main application entry point
   - `main.go`: CLI commands (serve, read-isbn, tui)
   - `run_server.go`: Server initialization with middleware
   - `routes.go`: API endpoint definitions
   - `otel.go`: OpenTelemetry instrumentation setup

2. **pkg/** - Core business logic
   - `handlers/`: HTTP request handlers for CRUD operations
   - `models/`: Data structures and Google Books API integration
   - `readIsbn/`: Barcode scanner integration using evdev
   - `tui/`: Terminal UI implementation using tview
   - `cron/`: Background tasks for AI enrichment

3. **Database**
   - SQLite with modernc.org/sqlite driver
   - Migrations in `/migrations/` using Goose
   - Automatic backup via Litestream to S3-compatible storage
   - Full-text search using Bleve v2

### API Endpoints

The server exposes these main endpoints (defined in `routes.go`):
- Book CRUD operations
- Borrowing system
- Search functionality
- Static file serving for web UI

### Key Dependencies

- **Web Framework**: Echo v4
- **Database**: SQLite with JSONB support
- **Search**: Bleve v2 full-text search
- **Observability**: OpenTelemetry with Prometheus metrics
- **UI**: tview for terminal interface
- **External APIs**: Google Books, Perplexity AI

### Database Schema

The database uses 4 migrations:
1. Initial tables for books and metadata
2. Schema refinements
3. Borrowing system implementation
4. AI enrichment features

### Deployment

- Supports Docker deployment
- Gokrazy configuration for embedded Linux (Raspberry Pi)
- Litestream configuration for database replication
- OpenTelemetry instrumentation for observability

## Development Tips

1. The server requires a Perplexity API key for AI enrichment features
2. Database is stored in `./.db/librascan.db` (gitignored)
3. Use `make dev` for hot reload during development
4. Barcode scanner requires Linux evdev access (typically `/dev/input/eventX`)
5. The TUI and scanner CLI communicate with the server via HTTP API