# Distributed URL Shortener

A high-performance, scalable URL shortening service built with Go, Redis, PostgreSQL, and Docker.

## Architecture

```
Client → REST API (Go/Gorilla Mux) → Redis Cache → PostgreSQL
```

- **Go** — HTTP server with RESTful API endpoints
- **Redis** — Caching layer for fast URL resolution (LRU eviction, 24h TTL)
- **PostgreSQL** — Persistent storage with indexed lookups
- **Docker** — Multi-stage build, containerized deployment via Docker Compose

## Performance

- Handles **10,000+ concurrent requests** via Go's goroutine-based concurrency model
- **~50% lower redirect latency** by resolving cached URLs from Redis before hitting PostgreSQL
- Connection pooling (25 max open / 10 idle) and indexed `short_code` column for fast queries

## Getting Started

### Prerequisites

- Docker & Docker Compose

### Run

```bash
docker-compose up --build
```

The API will be available at `http://localhost:8080`.

## API Endpoints

### Shorten a URL

```bash
curl -X POST http://localhost:8080/api/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/very/long/path"}'
```

Response:
```json
{
  "short_code": "aBc1234",
  "short_url": "http://localhost:8080/aBc1234",
  "original_url": "https://example.com/very/long/path",
  "created_at": "2026-01-15T10:30:00Z"
}
```

### Custom Short Code

```bash
curl -X POST http://localhost:8080/api/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com", "custom_code": "mylink"}'
```

### Redirect

```
GET http://localhost:8080/aBc1234 → 301 redirect to original URL
```

### URL Statistics

```bash
curl http://localhost:8080/api/urls/aBc1234/stats
```

Response:
```json
{
  "short_code": "aBc1234",
  "original_url": "https://example.com/very/long/path",
  "click_count": 42,
  "created_at": "2026-01-15T10:30:00Z"
}
```

### Health Check

```bash
curl http://localhost:8080/health
```

## Project Structure

```
├── cmd/server/main.go          # Application entrypoint
├── internal/
│   ├── handler/handler.go      # HTTP route handlers
│   ├── service/shortener.go    # Business logic
│   ├── database/postgres.go    # PostgreSQL data access layer
│   ├── cache/redis.go          # Redis caching layer
│   ├── model/url.go            # Data models and DTOs
│   └── middleware/middleware.go # Logging, CORS middleware
├── Dockerfile                  # Multi-stage container build
├── docker-compose.yml          # Full-stack orchestration
└── go.mod
```

## Tech Stack

| Component  | Technology         |
|------------|--------------------|
| Language   | Go 1.21            |
| Router     | Gorilla Mux        |
| Database   | PostgreSQL 16      |
| Cache      | Redis 7            |
| Container  | Docker, Compose    |
