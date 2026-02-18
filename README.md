# RSSHub Go - RSS feed generation framework

A high-performance RSS feed generation framework written in Go, inspired by RSSHub.

## Features

- 🚀 **High Performance**: Built on Go's concurrency model
- 🎯 **Easy to Use**: Simple API for creating RSS routes
- 📦 **Extensible**: Plugin-based architecture
- 🔧 **Flexible**: Support for HTML scraping, JSON APIs, XML feeds
- ⚡ **Production Ready**: Built-in caching, rate limiting, retries
- 🛡 **Type Safe**: Full Go types and interfaces

## Quick Start

```go
package main

import (
    "context"
    "log"
    "github.com/rsshub/go"
    "github.com/rsshub/go/models"
)

func main() {
    engine := rsshub.NewEngine()

    // Register a simple route
    engine.Register(&rsshub.Route{
        Path:    "/example",
        Handler: exampleHandler,
    })

    // Start server
    log.Fatal(engine.Start(":8080"))
}

func exampleHandler(ctx context.Context, req *rsshub.Request) (*models.Feed, error) {
    // Your logic here
    return &models.Feed{
        Title: "Example Feed",
        Items: []models.Item{
            {Title: "Item 1", Link: "https://example.com/1"},
        },
    }, nil
}
```

## Project Structure

```
rsshub-go/
├── internal/
│   ├── client/          # HTTP client with retry, proxy support
│   ├── parser/          # HTML, JSON, XML parsers utilities
│   └── middleware/      # Middleware implementations
├── pkg/
│   ├── models/          # Data structures (Feed, Item, Route)
│   ├── rss/            # RSS/Atom XML generation
│   └── cache/          # Cache abstraction (memory, redis)
├── routes/             # Route implementations
│   ├── github/
│   ├── twitter/
│   └── ...
├── examples/           # Example implementations
└── cmd/               # CLI tool
```

## Components

### Core Interfaces

- `Route`: Route definition and handler
- `Engine`: Main application with middleware stack
- `Client`: HTTP client with retry logic
- `Feed`: RSS feed structure
- `Cache`: Cache storage interface

### Utilities

- HTML parsing (goquery)
- JSON/XML parsing
- Date parsing and formatting
- User-Agent rotation
- Proxy support
- Rate limiting

## Development

```bash
# Run tests
go test ./...

# Build
go build -o rsshub ./cmd/server

# Run
./rsshub
```

## Documentation

See `/pkg` for detailed API documentation.

## License

AGPL-3.0
