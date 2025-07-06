# Activity Log Microservice

A microservice built with Go, ArangoDB, NATS, and gRPC following Domain-Driven Design (DDD) and Clean Architecture patterns.

## Features

- **Clean Architecture**: Follows DDD principles with clear separation of concerns
- **Event-Driven**: Uses NATS for asynchronous event processing
- **ArangoDB Storage**: Stores data in ArangoDB with full ACID compliance
- **gRPC API**: High-performance API with protocol buffers
- **Monitoring**: Integrated with Prometheus and Jaeger for observability
- **100% Test Coverage**: Comprehensive unit tests for all components

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   User Service  │───▶│      NATS       │───▶│ Activity Log    │
│                 │    │                 │    │ Service         │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                       │
                                              ┌────────┼────────┐
                                              ▼        ▼        
                                         ┌─────────┐ ┌──────┐ 
                                         │ArangoDB │ │NATS  │ 
                                         │         │ │Event │ 
                                         └─────────┘ └──────┘ 
```

## Project Structure

```
.
├── cmd/server/                 # Application entry point
├── internal/
│   ├── domain/                 # Domain layer (entities, value objects)
│   ├── application/            # Application layer (use cases)
│   ├── infrastructure/         # Infrastructure layer (databases, messaging)
│   └── delivery/              # Delivery layer (gRPC, NATS handlers)
├── pkg/proto/                 # Protocol buffer definitions
├── configs/                   # Configuration files
├── docker-compose.yml         # Docker composition
└── Dockerfile                 # Container definition
```

## Quick Start

### Prerequisites

- Go 1.21+
- Docker and Docker Compose
- Protocol Buffer Compiler (`protoc`)

### Running with Docker Compose

1. Clone the repository and navigate to the project directory

2. Start all services:
```bash
make docker-run
```

3. Check logs:
```bash
make docker-logs
```

### Running Locally

1. Install dependencies:
```bash
make deps
```

2. Generate protobuf files:
```bash
make proto
```

3. Start infrastructure services:
```bash
docker-compose up -d arangodb nats jaeger prometheus grafana
```

4. Run the service:
```bash
make run
```

## Testing

Run all tests with coverage:
```bash
make test-coverage
```

View coverage report:
```bash
open coverage.html
```

## API Usage

### gRPC Service

The service exposes a gRPC API with the following methods:

- `CreateActivityLog`: Create a new activity log entry
- `GetActivityLog`: Retrieve an activity log by ID
- `ListActivityLogs`: List activity logs for a company with pagination

### Example gRPC Client

```go
conn, err := grpc.Dial("localhost:9000", grpc.WithInsecure())
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

client := pb.NewActivityLogServiceClient(conn)

req := &pb.CreateActivityLogRequest{
    ActivityName:     "user_created",
    CompanyId:        "company1",
    ObjectName:       "user",
    ObjectId:         "user123",
    Changes:          `{"name": "John Doe", "email": "john@example.com"}`,
    FormattedMessage: "User John Doe was created",
    ActorId:          "actor1",
    ActorName:        "Admin",
    ActorEmail:       "admin@example.com",
}

resp, err := client.CreateActivityLog(context.Background(), req)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Created activity log: %s\n", resp.ActivityLog.Id)
```

## Configuration

Configuration is managed through YAML files and environment variables. See `configs/config.yaml` for all available options.

Key environment variables:
- `CONFIG_PATH`: Path to configuration file
- `ARANGO_URL`: ArangoDB connection URL
- `ARANGO_PASSWORD`: ArangoDB password
- `NATS_URL`: NATS server URL
- `JAEGER_ENDPOINT`: Jaeger tracing endpoint

## Monitoring

### Prometheus Metrics

Available at `http://localhost:2112/metrics`:
- Activity log creation counters
- Processing duration histograms
- NATS message processing metrics
- Database operation metrics

### Jaeger Tracing

View distributed traces at `http://localhost:16686`

### Grafana Dashboard

Access dashboards at `http://localhost:3000` (admin/admin)

## Development

### Available Make Commands

```bash
make build          # Build the binary
make run            # Run the service locally
make test           # Run all tests
make test-coverage  # Run tests with coverage report
make docker-build   # Build Docker image
make docker-run     # Start services with Docker Compose
make docker-stop    # Stop Docker Compose services
make proto          # Generate protobuf files
make deps           # Download and tidy dependencies
make lint           # Run linter
make clean          # Clean build artifacts
```
