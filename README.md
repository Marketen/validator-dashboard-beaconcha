# Validator Dashboard API

A read-only public REST API that consumes the Beaconcha v2 API and aggregates validator-related data.

## Features

- **Single Endpoint**: `GET /validator` to fetch aggregated data for up to 100 validators
- **Multi-chain Support**: Supports `mainnet` and `hoodi` chains
- **Flexible Time Ranges**: Query rewards/performance for `24h`, `7d`, `30d`, `90d`, or `all_time`
- **Aggregated Data**: Returns per-validator overviews with combined rewards/performance metrics
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
GET /validator?ids=1,2,3&chain=mainnet&range=all_time
```

**Query Parameters:**
| Parameter | Required | Description |
|-----------|----------|-------------|
| `ids` | Yes | Comma-separated list of validator indices (1-100, unique, non-negative) |
| `chain` | Yes | Target chain: `mainnet` or `hoodi` |
| `range` | No | Evaluation window for aggregates: `24h`, `7d`, `30d`, `90d`, `all_time` (default: `all_time`) |

**Example Request:**
```bash
curl "http://localhost:8080/validator?ids=1,2,3&chain=mainnet&range=24h"
```

**Response Structure:**

The response contains:
- `validators`: Per-validator overview data (status, balances, epochs, etc.)
- `rewards`: **Aggregated** rewards for ALL requested validators combined
- `performance`: **Aggregated** performance metrics for ALL requested validators combined

```json
{
  "validators": {
    "1": {
      "slashed": false,
      "status": "active_online",
      "withdrawalCredentials": {
        "type": "execution",
        "prefix": "0x01",
        "credential": "...",
        "address": "0x..."
      },
      "activationEpoch": 0,
      "exitEpoch": 0,
      "currentBalance": "32004175273000000000",
      "effectiveBalance": "32000000000000000000",
      "online": true
    },
    "2": {
      "slashed": false,
      "status": "active_online",
      "...": "..."
    },
    "3": {
      "...": "..."
    }
  },
  "rewards": {
    "total": "6099749000000000",
    "totalReward": "6104156000000000",
    "totalPenalty": "4407000000000",
    "totalMissed": "36022000000000",
    "proposals": {
      "total": "0",
      "executionLayerReward": "0",
      "attestationInclusionReward": "0",
      "syncInclusionReward": "0",
      "slashingInclusionReward": "0",
      "missedClReward": "0",
      "missedElReward": "0"
    },
    "attestations": {
      "total": "6099749000000000",
      "head": "1544410000000000",
      "source": "1598352000000000",
      "target": "2956987000000000",
      "inactivityLeakPenalty": "0"
    },
    "syncCommittees": {
      "total": "0",
      "reward": "0",
      "penalty": "0",
      "missedReward": "0"
    }
  },
  "performance": {
    "beaconscore": 0.9934041,
    "attestations": {
      "assigned": 450,
      "included": 450,
      "missed": 0,
      "correctHead": 442,
      "correctSource": 450,
      "correctTarget": 449,
      "avgInclusionDelay": 0.0044444446,
      "beaconscore": 0.9934041
    },
    "syncCommittees": {
      "assigned": 0,
      "successful": 0,
      "missed": 0,
      "beaconscore": null
    },
    "proposals": {
      "assigned": 0,
      "successful": 0,
      "missed": 0,
      "includedSlashings": 0,
      "beaconscore": null
    }
  }
}
```

**Note:** The `rewards` and `performance` sections are aggregated across ALL validators in the request—they are NOT per-validator. If you request validators 1, 2, and 3, the rewards/performance represent the combined totals for all three.
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
