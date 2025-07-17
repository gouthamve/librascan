# LibraScan

A personal library management system with barcode scanning capabilities built in Go. LibraScan helps you catalog and manage your book collection with features like barcode scanning, AI-powered book enrichment, and a web interface for browsing your library.

## Features

- **Barcode Scanner Integration**: Scan book ISBNs using a USB barcode scanner
- **Book Metadata Retrieval**: Automatically fetch book details from Google Books and Open Library APIs
- **AI Enrichment**: Enhance book metadata using Perplexity AI
- **Web Interface**: Browse and search your book collection through a responsive web UI
- **Terminal UI**: Alternative TUI interface for terminal enthusiasts
- **Borrowing System**: Track who has borrowed which books
- **Full-Text Search**: Search through your collection using Bleve
- **Database Backup**: Automatic SQLite backups to S3-compatible storage via Litestream
- **Observability**: Built-in OpenTelemetry instrumentation and Prometheus metrics

## Installation

### Prerequisites

- Go 1.19 or higher
- SQLite3
- Linux (for barcode scanner functionality)
- Perplexity API key (for AI enrichment features)

### Building from Source

```bash
# Clone the repository
git clone https://github.com/gouthamve/librascan.git
cd librascan

# Install dependencies and build
make build

# Or use the development mode with hot reload
make dev
```

### Docker

```bash
# Build Docker image
make docker-build

# Run with Docker
docker run -p 8080:8080 \
  -e PERPLEXITY_KEY=your-api-key \
  -v $(pwd)/.db:/app/.db \
  gouthamve/librascan
```

## Usage

### Running the Server

```bash
# Start the HTTP server
./librascan serve --perplexity-key YOUR_API_KEY

# The server will start on http://localhost:8080
```

### Using the Barcode Scanner

Connect your USB barcode scanner and find its device path (usually `/dev/input/eventX`):

```bash
# Start the barcode scanner CLI
./librascan read-isbn \
  --server-url http://localhost:8080 \
  --input-device-path /dev/input/event0
```

Scan a book's ISBN barcode and it will automatically be added to your library.

### Terminal UI

```bash
# Launch the TUI interface
./librascan tui --server-url http://localhost:8080
```

## API Endpoints

- `GET /` - Web interface showing all books
- `GET /books` - Get all books (JSON)
- `GET /books/:isbn` - Get a specific book
- `POST /books/:isbn` - Add a book by ISBN
- `DELETE /books/:isbn` - Delete a book
- `POST /books/borrow` - Borrow a book
- `GET /people` - Get all people (for borrowing system)
- `GET /shelf/:id` - Get shelf information
- `GET /metrics` - Prometheus metrics

### Adding a Book

```bash
# Add a book by ISBN with optional shelf location
curl -X POST "http://localhost:8080/books/9780134685991?shelf_id=1&row_number=3"
```

### Borrowing a Book

```bash
curl -X POST http://localhost:8080/books/borrow \
  -H "Content-Type: application/json" \
  -d '{"isbn": 9780134685991, "person_name": "John Doe"}'
```

## Configuration

### Environment Variables

- `PERPLEXITY_KEY` - Perplexity API key for AI enrichment
- `OTEL_EXPORTER_OTLP_ENDPOINT` - OpenTelemetry collector endpoint
- `OTEL_SERVICE_NAME` - Service name for telemetry (default: librascan)

### Database

LibraScan uses SQLite for data storage. The database is stored in `./.db/librascan.db` by default.

### Litestream Backup

Configure Litestream for automatic backups by editing `litestream.yml`:

```yaml
dbs:
  - path: ./.db/librascan.db
    replicas:
      - type: s3
        bucket: your-bucket
        path: librascan
        endpoint: your-s3-endpoint
```

## Development

### Project Structure

```
librascan/
├── cmd/librascan/      # Main application entry points
├── pkg/
│   ├── handlers/       # HTTP request handlers
│   ├── models/         # Data structures
│   ├── db/            # Database queries (sqlc generated)
│   ├── readIsbn/      # Barcode scanner integration
│   ├── tui/           # Terminal UI
│   └── cron/          # Background tasks
├── migrations/         # Database migrations
├── sql/               # SQL queries for sqlc
└── templates/         # HTML templates
```

### Running Tests

```bash
# Run all tests
make test

# Run specific package tests
go test ./pkg/handlers/...
```

### Database Migrations

Migrations are managed using Goose and are embedded in the binary:

```bash
# Migrations run automatically on server start
# Located in migrations/ directory
```

### Development Setup

```bash
# Install development dependencies
make setup-deps

# Download latest database from backup
make download-db

# Update dependencies
make update-deps
```

## Deployment

### Gokrazy (Raspberry Pi)

LibraScan includes Gokrazy configuration for deployment on embedded Linux devices:

```bash
# Deploy to Raspberry Pi
gokrazy_packer -update
```

### Systemd Service

Create a systemd service file at `/etc/systemd/system/librascan.service`:

```ini
[Unit]
Description=LibraScan Book Management System
After=network.target

[Service]
Type=simple
User=librascan
ExecStart=/usr/local/bin/librascan serve --perplexity-key YOUR_KEY
Restart=always
Environment="OTEL_SERVICE_NAME=librascan"

[Install]
WantedBy=multi-user.target
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Google Books API for book metadata
- Open Library API for additional book information
- Perplexity AI for intelligent book enrichment
- Bleve for full-text search capabilities
- Litestream for SQLite replication