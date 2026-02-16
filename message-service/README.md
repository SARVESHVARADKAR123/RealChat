# Message Service

Clean architecture microservice for handling real-time messaging in RealChat.

## Architecture

This service follows **Hexagonal/Clean Architecture** principles with clear separation of concerns:

### Directory Structure

```
message-service/
├── cmd/server/           # Application entrypoint (wiring only)
├── internal/
│   ├── app/             # Application bootstrap & lifecycle
│   ├── domain/          # Pure business logic (NO infrastructure)
│   ├── service/         # Use-cases & orchestration
│   ├── repository/      # Repository interfaces
│   ├── infrastructure/  # Repository implementations (Postgres, Redis, Kafka)
│   ├── transport/       # Delivery layer (gRPC, HTTP)
│   ├── config/          # Configuration management
│   └── middleware/      # Cross-cutting concerns
├── pkg/                 # Reusable libraries
├── api/                 # Generated protobuf code
├── migrations/          # Database schema migrations
└── test/               # Integration & E2E tests
```

### Layer Responsibilities

- **Domain**: Pure business entities and rules, no external dependencies
- **Service**: Orchestrates use-cases, coordinates between domain and repositories
- **Repository**: Interfaces defining data access contracts
- **Infrastructure**: Concrete implementations (Postgres, Redis, Kafka)
- **Transport**: HTTP/gRPC handlers, request/response mapping
- **App**: Dependency injection and application wiring

## Technology Stack

- **Language**: Go 1.21+
- **Database**: PostgreSQL
- **Cache**: Redis
- **Message Queue**: Kafka
- **Transport**: gRPC (primary), HTTP (optional)

## Getting Started

TODO: Add build and run instructions

## Development

TODO: Add development workflow
