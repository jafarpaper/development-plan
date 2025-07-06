# Initialization System Documentation

## Overview

The initialization system provides centralized dependency management for all microservices in the Activity Log Service. It eliminates code duplication and ensures consistent setup across all services.

## Architecture

### Core Components

1. **Dependencies Struct** - Holds all initialized dependencies
2. **InitializationOptions** - Configuration for service-specific requirements
3. **Service-Specific Initializers** - Tailored dependency setup for each service
4. **Cleanup Management** - Proper resource cleanup and connection closing

## Usage

### Basic Initialization

```go
import "activity-log-service/internal/initialization"

// Initialize with default options
deps, err := initialization.Initialize(nil)
if err != nil {
    log.Fatal("Failed to initialize dependencies:", err)
}
defer deps.Cleanup()

// Use dependencies
server := server.NewHTTPServer(deps.UseCase, deps.Config, deps.Logger, deps.Tracer)
```

### Service-Specific Initialization

Each service has a dedicated initializer function:

```go
// HTTP Server
deps, err := initialization.GetHTTPDependencies("configs/config.yaml")

// gRPC Server  
deps, err := initialization.GetGRPCDependencies("configs/config.yaml")

// NATS Consumer
deps, err := initialization.GetConsumerDependencies("configs/config.yaml")

// Cron Server
deps, err := initialization.GetCronDependencies("configs/config.yaml")
```

## Dependencies Structure

```go
type Dependencies struct {
    Config          *config.Config
    Logger          *logrus.Logger
    Tracer          opentracing.Tracer
    TracerCloser    func() error
    Repository      repository.ActivityLogRepository
    Cache           *cache.RedisCache
    Publisher       *messaging.NATSPublisher
    Mailer          *email.Mailer
    UseCase         *usecase.ActivityLogUseCase
}
```

## Service Requirements

### HTTP Server
- ✅ ArangoDB Repository
- ✅ Redis Cache (optional)
- ✅ NATS Publisher (required)
- ✅ Email Service (optional)
- ✅ Distributed Tracing
- ✅ Use Case Layer

### gRPC Server
- ✅ ArangoDB Repository
- ✅ Redis Cache (optional)
- ❌ NATS Publisher (not needed)
- ❌ Email Service (not needed)
- ✅ Distributed Tracing
- ✅ Use Case Layer

### NATS Consumer
- ✅ ArangoDB Repository
- ✅ Redis Cache (optional)
- ❌ NATS Publisher (not needed)
- ❌ Email Service (not needed)
- ✅ Distributed Tracing
- ❌ Use Case Layer (direct repository access)

### Cron Server
- ✅ ArangoDB Repository
- ✅ Redis Cache (required)
- ❌ NATS Publisher (not needed)
- ✅ Email Service (optional)
- ✅ Distributed Tracing
- ❌ Use Case Layer (direct repository access)

## Configuration Options

```go
type InitializationOptions struct {
    ConfigPath        string  // Path to config file
    RequireCache      bool    // Fail if Redis unavailable
    RequireEmail      bool    // Fail if Email not configured
    RequireNATS       bool    // Fail if NATS unavailable
    MetricsPortOffset int     // Offset for metrics port
}
```

## Error Handling

The initialization system provides comprehensive error handling:

- **Configuration Errors**: Invalid or missing config files
- **Connection Errors**: Database, Redis, NATS connection failures
- **Dependency Errors**: Missing required services based on options
- **Validation Errors**: Invalid configuration parameters

## Resource Cleanup

Always call `deps.Cleanup()` to properly close connections:

```go
deps, err := initialization.GetHTTPDependencies(configPath)
if err != nil {
    log.Fatal(err)
}

// Ensure cleanup happens even if main() panics
defer func() {
    if err := deps.Cleanup(); err != nil {
        log.Error("Cleanup failed:", err)
    }
}()
```

## Service Manager Script

Use the provided service manager for easy service management:

```bash
# Build all services
./scripts/service-manager.sh build all

# Start specific service
./scripts/service-manager.sh start http

# Check status
./scripts/service-manager.sh status all

# View logs
./scripts/service-manager.sh logs consumer

# Stop all services
./scripts/service-manager.sh stop all
```

## Benefits

### 🎯 **Consistency**
- All services use identical initialization patterns
- Standardized error handling and logging
- Consistent dependency injection

### 🔧 **Maintainability**
- Centralized configuration management
- Single source of truth for dependencies
- Easy to add new services or modify existing ones

### 🚀 **Developer Experience**
- Reduced boilerplate code in main.go files
- Clear separation of concerns
- Easy testing with mock dependencies

### 🛡️ **Reliability**
- Proper resource cleanup
- Graceful error handling
- Service-specific dependency validation

## Testing

The initialization system supports dependency injection for testing:

```go
func TestWithMockDeps(t *testing.T) {
    // Create mock dependencies
    mockRepo := &mocks.ActivityLogRepository{}
    mockLogger := logrus.New()
    
    // Create test dependencies
    deps := &initialization.Dependencies{
        Repository: mockRepo,
        Logger:     mockLogger,
        // ... other mocks
    }
    
    // Test your service
    server := server.NewHTTPServer(deps.UseCase, deps.Config, deps.Logger, deps.Tracer)
    // ... test code
}
```

## Migration Guide

### Before (Old Pattern)
```go
// Lots of repetitive initialization code in each main.go
cfg, err := config.LoadConfig(configPath)
logger := logrus.New()
tracer, closer, err := tracing.InitJaeger(&cfg.Jaeger)
arangoRepo, err := database.NewArangoActivityLogRepository(...)
// ... many more lines
```

### After (New Pattern)
```go
// Clean, centralized initialization
deps, err := initialization.GetHTTPDependencies(configPath)
defer deps.Cleanup()

server := server.NewHTTPServer(deps.UseCase, deps.Config, deps.Logger, deps.Tracer)
```

## Future Enhancements

- [ ] Health check endpoints in initialization
- [ ] Metrics collection during initialization
- [ ] Configuration hot-reloading
- [ ] Dependency graph visualization
- [ ] Performance monitoring integration