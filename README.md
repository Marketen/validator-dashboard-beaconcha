# Validator Dashboard API

A read-only public REST API that consumes the Beaconcha v2 API and aggregates validator-related data.

## Features

- **Single Endpoint**: `GET /validator` to fetch aggregated data for up to 100 validators
- **Multi-chain Support**: Supports `mainnet` and `hoodi` chains
- **Beaconcha Rate Limiting**: Adaptive rate limiting using Beaconcha response headers
- **Abuse Prevention**: Request validation and query parameter limits
- **Cursor-based Pagination**: Automatically fetches all pages from Beaconcha v2 API
- **Nginx Ready**: Designed to be deployed behind nginx for caching and per-IP rate limiting

## Quick Start

### Prerequisites

- Docker and Docker Compose
- A Beaconcha API key (optional, for higher rate limits)

### Running with Docker Compose

```bash
# Clone the repository
git clone https://github.com/Marketen/validator-dashboard-beaconcha.git
cd validator-dashboard-beaconcha

# Set your API key
cp .env.example .env
# Edit .env and add your BEACONCHAIN_API_KEY

# Start the service
docker compose up -d

# Check logs
docker compose logs -f
```

### Running Locally (Development)

```bash
# Download dependencies
go mod tidy

# Set API key and run
BEACONCHAIN_API_KEY=your_key go run cmd/server/main.go
```

## API Endpoints

### Health Check

```
GET /health
```

Response:
```json
{
  "status": "healthy",
  "time": "2026-01-02T12:00:00Z"
}
```

### Get Validator Data

```
GET /validator?ids=12345,67890&chain=mainnet
```

**Query Parameters:**
- `ids`: Comma-separated list of validator indices
  - Minimum: 1 validator
  - Maximum: 100 validators
  - Values must be unique
  - Values must be non-negative integers
- `chain`: Required string, either `mainnet` or `hoodi`

**Example:**
```bash
curl "http://localhost:8080/validator?ids=12345,67890&chain=mainnet"
```

**Response:**

```json
{
  "12345": {
    "overview": {
      "slashed": false,
      "status": "active_online",
      "withdrawalCredentials": {
        "type": "execution",
        "address": "0x..."
      },
      "activationEpoch": 290297,
      "exitEpoch": 0,
      "currentBalance": 32014494648,
      "effectiveBalance": 32000000000,
      "online": true
    },
    "rewards": {
      "total": "1234567890000000000",
      "totalReward": "1234567890000000000",
      "totalPenalty": "0",
      "totalMissed": "0",
      "proposals": {
        "total": "...",
        "executionLayerReward": "...",
        "attestationInclusionReward": "...",
        "syncInclusionReward": "...",
        "slashingInclusionReward": "...",
        "missedClReward": "...",
        "missedElReward": "..."
      },
      "attestations": {
        "total": "...",
        "head": "...",
        "source": "...",
        "target": "...",
        "inactivityLeakPenalty": "..."
      },
      "syncCommittees": {
        "total": "...",
        "reward": "...",
        "penalty": "...",
        "missedReward": "..."
      }
    },
    "performance": {
      "beaconscore": 0.99,
      "attestations": {
        "assigned": 10000,
        "included": 9990,
        "missed": 10,
        "correctHead": 9980,
        "correctSource": 9990,
        "correctTarget": 9985,
        "avgInclusionDelay": 1.05,
        "beaconscore": 0.99
      },
      "syncCommittees": {
        "assigned": 512,
        "successful": 510,
        "missed": 2,
        "beaconscore": 0.996
      },
      "proposals": {
        "assigned": 5,
        "successful": 5,
        "missed": 0,
        "includedSlashings": 0,
        "beaconscore": 1.0
      }
    }
  }
}
```

## Configuration

Configuration is done via environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `BEACONCHAIN_BASE_URL` | Beaconcha API base URL | `https://beaconcha.in` |
| `BEACONCHAIN_API_KEY` | Beaconcha API key | (empty) |
| `BEACONCHAIN_RATE_LIMIT` | Rate limit for Beaconcha API calls | `1s` |
| `BEACONCHAIN_TIMEOUT` | Timeout for Beaconcha API calls | `30s` |

| `MAX_VALIDATOR_IDS` | Max validators per request | `100` |

## Architecture

### Project Structure

```
.
├── cmd/
│   └── server/
│       └── main.go          # Application entry point
├── internal/
│   ├── api/
│   │   ├── handler.go       # HTTP handlers and middleware
│   │   └── handler_test.go  # Handler tests
│   ├── beaconcha/
│   │   └── client.go        # Beaconcha API client
│   ├── config/
│   │   └── config.go        # Configuration management
│   ├── models/
│   │   ├── api.go           # Public API models
│   │   └── beaconcha.go     # Beaconcha API models
│   ├── ratelimiter/
│   │   ├── ratelimiter.go   # Beaconcha rate limiter
│   │   └── ratelimiter_test.go
│   └── service/
│       └── validator.go     # Business logic layer
├── docker-compose.yaml
├── Dockerfile
├── go.mod
└── README.md
```

### Design Decisions

1. **Beaconcha Rate Limiting**
   - Adaptive rate limiting using Beaconcha response headers (`ratelimit-remaining`, `ratelimit-reset`)
   - Falls back to token bucket rate limiter using `golang.org/x/time/rate`
   - Configurable via environment variables

2. **Pagination**
   - Cursor-based pagination for the validators endpoint (page size: 10)
   - Automatically fetches all pages until no `next_cursor` is returned
   - Each page request respects rate limiting

3. **Beaconcha Client**
   - Encapsulated behind a dedicated client layer
   - All requests wait for the adaptive rate limiter before executing
   - Parses rate limit headers from responses to optimize request timing
   - Strongly-typed request/response models

4. **Middleware Stack**
   - Max body size (1MB) - prevents large payload attacks
   - CORS - allows cross-origin requests
   - Logging - structured JSON logs
   - Recovery - graceful panic handling

5. **Nginx Integration**
   - Per-IP rate limiting and response caching should be handled by nginx
   - See [Nginx Configuration](#nginx-configuration) for setup examples

6. **Testability**
   - Interfaces and dependency injection for testable components
   - Unit tests for core functionality
   - Components can be easily mocked for integration tests

## Nginx Configuration

This API is designed to be deployed behind nginx for **per-IP rate limiting** and **response caching**.

### Rate Limiting

Nginx can implement per-IP rate limiting with burst support:

```nginx
# Define rate limit zone (10 req/min = 1 req per 6 seconds, with burst of 5)
limit_req_zone $binary_remote_addr zone=validator_api:10m rate=10r/m;

server {
    listen 80;
    server_name your-domain.com;

    location /validator {
        # Apply rate limit: allow burst of 5 requests, then enforce rate
        limit_req zone=validator_api burst=5 nodelay;
        limit_req_status 429;

        proxy_pass http://validator-dashboard:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    location /health {
        # No rate limiting on health checks
        proxy_pass http://validator-dashboard:8080;
    }
}
```

### Response Caching

With GET requests, nginx can cache responses using the URL as the cache key (no Lua required):

```nginx
# Cache zone configuration
proxy_cache_path /var/cache/nginx/validator levels=1:2 keys_zone=validator_cache:10m max_size=100m inactive=30m;

server {
    listen 80;
    server_name your-domain.com;

    location /validator {
        # Rate limiting
        limit_req zone=validator_api burst=5 nodelay;
        limit_req_status 429;

        # Enable caching
        proxy_cache validator_cache;
        proxy_cache_valid 200 20m;  # Cache successful responses for 20 minutes
        
        # Add cache status header for debugging
        add_header X-Cache-Status $upstream_cache_status;

        proxy_pass http://validator-dashboard:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    location /health {
        proxy_pass http://validator-dashboard:8080;
    }
}
```

### Complete Example with Docker Compose

```yaml
# docker-compose.yaml
services:
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - nginx-cache:/var/cache/nginx
    depends_on:
      - validator-dashboard

  validator-dashboard:
    build: .
    environment:
      - BEACONCHAIN_API_KEY=${BEACONCHAIN_API_KEY}
    expose:
      - "8080"

volumes:
  nginx-cache:
```

### How Caching Works

With GET requests, the URL serves as a natural cache key:

- **Request A**: `/validator?ids=3,5,6&chain=mainnet` → cached
- **Request B**: `/validator?ids=7,5,70&chain=hoodi` → cached separately (different URL)
- **Request C**: `/validator?ids=3,5,6&chain=mainnet` → cache HIT (same URL as Request A)

Different validator IDs or chains result in different URLs, so they are cached separately.

## Development

### Running Tests

```bash
go test ./... -v
```

### Building

```bash
go build -o validator-dashboard ./cmd/server
```

## License

MIT License
