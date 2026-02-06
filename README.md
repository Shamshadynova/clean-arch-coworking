# Clean Arch Coworking

Booking system for a coworking space built with **Domain-Driven Design** and **Clean (Hexagonal) Architecture** in Go.

## Prerequisites

- Go 1.24+
- Docker & Docker Compose (for PostgreSQL)
- [golangci-lint](https://golangci-lint.run/usage/install/) (optional, for linting)

## Quick Start

```bash
# Start PostgreSQL (prepared for future migration)
make db-up

# Run the service
make run

# Run tests
make test

# Run linter
make lint
```

## Project Structure

```
├── api/                          # OpenAPI specification
│   └── openapi.yaml
├── cmd/
│   └── booking-service/
│       └── main.go               # Entry point, dependency wiring
├── internal/
│   ├── config/
│   │   └── config.go             # Environment-based configuration
│   └── booking/
│       ├── adapters/
│       │   └── http/
│       │       ├── handler.go    # HTTP handlers & router
│       │       └── middleware.go  # RequestID, Logger, Recovery
│       ├── application/
│       │   ├── dto.go            # Input/output DTOs
│       │   ├── interfaces.go     # Port definitions
│       │   └── service.go        # Use-case orchestration
│       ├── domain/
│       │   ├── booking.go        # Booking aggregate
│       │   ├── booking_service.go# Domain service
│       │   ├── date_range.go     # DateRange value object
│       │   ├── errors.go         # Domain errors
│       │   ├── events.go         # Domain events
│       │   ├── money.go          # Money value object
│       │   └── room.go           # Room entity (stub)
│       └── infrastructure/
│           ├── bus/dummy/         # No-op event bus
│           ├── memory/            # In-memory booking repository
│           ├── outbox/            # Outbox event store
│           ├── policy/dummy/      # Dummy availability & pricing
│           └── transaction/       # Unit of Work
├── docker-compose.yml
├── Makefile
├── .env.example
└── .golangci.yml
```

## API Endpoints

| Method | Path              | Description          |
|--------|-------------------|----------------------|
| POST   | `/bookings`       | Create a new booking |
| GET    | `/bookings/{id}`  | Get booking details  |

See [`api/openapi.yaml`](api/openapi.yaml) for the full specification.

## Architecture

The codebase follows hexagonal architecture:

- **Domain** — aggregates, value objects, domain events, domain services. Zero external dependencies.
- **Application** — use-case services that orchestrate domain logic via port interfaces.
- **Infrastructure** — concrete adapters: repositories, event bus, outbox, policies.
- **Adapters** — HTTP handlers that translate between the outside world and application DTOs.

## Configuration

Configuration is read from environment variables (see `.env.example`):

| Variable          | Default     | Description            |
|-------------------|-------------|------------------------|
| `APP_PORT`        | `8080`      | HTTP server port       |
| `LOG_LEVEL`       | `info`      | Log level (debug/info/warn/error) |
| `POSTGRES_HOST`   | `localhost` | PostgreSQL host        |
| `POSTGRES_PORT`   | `5432`      | PostgreSQL port        |
| `POSTGRES_DB`     | `coworking` | Database name          |
| `POSTGRES_USER`   | `coworking` | Database user          |
| `POSTGRES_PASSWORD`| `secret`   | Database password      |

## Running Tests

```bash
go test -v -race ./...
```
