# Validator Dashboard API

A read-only public REST API that consumes the Beaconcha v2 API and aggregates validator-related data.

## Features

- **Single Endpoint**: `POST /validator` to fetch aggregated data for up to 100 validators
- **Rate Limiting**: 
  - Global rate limiting for Beaconcha API (1 req/sec)
  - Per-IP rate limiting for abuse prevention (configurable)
- **Caching**: In-memory cache with configurable TTL (15-30 minutes)
- **Abuse Prevention**: Request validation, body size limits, and rate limiting

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
POST /validator
Content-Type: application/json

{
  "validatorIds": [12345, 67890]
}
```

**Request Constraints:**
- `validatorIds`: Array of integers
  - Minimum: 1 validator
  - Maximum: 100 validators
  - Values must be unique
  - Values must be non-negative

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
      "exitEpoch": 9223372036854776000,
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
| `CACHE_TTL` | Cache time-to-live (must be 15-30 minutes) | `20m` |
| `IP_RATE_LIMIT_REQUESTS` | Max requests per IP per window | `60` |
| `IP_RATE_LIMIT_WINDOW` | Rate limit window duration | `1m` |
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
│   ├── cache/
│   │   ├── cache.go         # In-memory cache implementation
│   │   └── cache_test.go    # Cache tests
│   ├── config/
│   │   └── config.go        # Configuration management
│   ├── models/
│   │   ├── api.go           # Public API models
│   │   └── beaconcha.go     # Beaconcha API models
│   ├── ratelimiter/
│   │   ├── ratelimiter.go   # Rate limiter implementation
│   │   └── ratelimiter_test.go
│   └── service/
│       └── validator.go         # Business logic layer
├── docker-compose.yaml
├── Dockerfile
├── go.mod
└── README.md
```

### Design Decisions

1. **Rate Limiting Strategy**
   - Global rate limiter using `golang.org/x/time/rate` for Beaconcha API calls
   - Per-IP rate limiting for incoming requests to prevent abuse
   - Both are configurable via environment variables

2. **Caching**
   - Simple in-memory cache with TTL
   - Cache key is derived from sorted validator IDs for consistency
   - Automatic cleanup of expired entries
   - `GetOrSet` pattern to prevent thundering herd

3. **Beaconcha Client**
   - Encapsulated behind a dedicated client layer
   - All requests wait for the global rate limiter before executing
   - Strongly-typed request/response models

4. **Middleware Stack**
   - Max body size (1MB) - prevents large payload attacks
   - Rate limiting - per-IP abuse prevention
   - CORS - allows cross-origin requests
   - Logging - structured JSON logs
   - Recovery - graceful panic handling

5. **Testability**
   - Interfaces and dependency injection for testable components
   - Unit tests for core functionality
   - Components can be easily mocked for integration tests

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
