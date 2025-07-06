# Sequence Diagrams

## Activity Log Service Sequence Diagrams

### 1. Create Activity Log via gRPC

```mermaid
sequenceDiagram
    participant Client
    participant gRPCServer as gRPC Server
    participant UseCase as Activity Log UseCase
    participant ArangoDB
    participant JSONFile as JSON File Storage
    participant NATS

    Client->>gRPCServer: CreateActivityLog(request)
    
    gRPCServer->>gRPCServer: Validate request parameters
    alt validation fails
        gRPCServer-->>Client: Error: Invalid parameters
    else validation passes
        gRPCServer->>UseCase: CreateActivityLog(request)
        
        UseCase->>UseCase: Create ActivityLog entity
        UseCase->>UseCase: Validate business rules
        
        alt validation fails
            UseCase-->>gRPCServer: Error: Business rule violation
            gRPCServer-->>Client: Error: Invalid activity log
        else validation passes
            UseCase->>ArangoDB: Create(activityLog)
            ArangoDB-->>UseCase: Success
            
            UseCase->>UseCase: Create ActivityLogCreated event
            UseCase->>NATS: PublishActivityLogCreated(event)
            NATS-->>UseCase: Success
            
            UseCase-->>gRPCServer: ActivityLog created
            gRPCServer-->>Client: CreateActivityLogResponse
        end
    end
```

### 2. Event-Driven Activity Log Processing

```mermaid
sequenceDiagram
    participant UserService
    participant NATS
    participant Consumer as NATS Consumer
    participant WorkerPool
    participant ArangoDB

    UserService->>NATS: Publish activity log event
    NATS->>Consumer: Deliver message to subscriber
    
    Consumer->>WorkerPool: Submit job to worker pool
    
    par Worker Processing
        WorkerPool->>WorkerPool: Process message in worker
        WorkerPool->>WorkerPool: Deserialize event
        
        WorkerPool->>ArangoDB: Save activity log
        ArangoDB-->>WorkerPool: Success
        
        WorkerPool->>Consumer: Job completed successfully
    and Error Handling
        alt processing fails
            WorkerPool->>Consumer: Job failed with error
            Consumer->>NATS: NAK (Negative Acknowledgment)
            Note over NATS: Message will be redelivered
        end
    end
    
    Consumer->>NATS: ACK (Acknowledge successful processing)
```

### 3. Get Activity Log by ID

```mermaid
sequenceDiagram
    participant Client
    participant gRPCServer as gRPC Server
    participant UseCase as Activity Log UseCase
    participant ArangoDB

    Client->>gRPCServer: GetActivityLog(id)
    
    gRPCServer->>gRPCServer: Validate ID parameter
    alt invalid ID
        gRPCServer-->>Client: Error: Invalid ID
    else valid ID
        gRPCServer->>UseCase: GetActivityLog(id)
        
        UseCase->>UseCase: Validate ActivityLogID value object
        UseCase->>ArangoDB: GetByID(id)
        
        alt not found
            ArangoDB-->>UseCase: Error: Not found
            UseCase-->>gRPCServer: Error: Activity log not found
            gRPCServer-->>Client: Error: Not found (404)
        else found
            ArangoDB-->>UseCase: ActivityLog entity
            UseCase-->>gRPCServer: ActivityLog
            gRPCServer-->>Client: GetActivityLogResponse
        end
    end
```

### 4. List Activity Logs with Pagination

```mermaid
sequenceDiagram
    participant Client
    participant gRPCServer as gRPC Server
    participant UseCase as Activity Log UseCase
    participant ArangoDB

    Client->>gRPCServer: ListActivityLogs(company_id, page, limit)
    
    gRPCServer->>gRPCServer: Validate parameters
    gRPCServer->>gRPCServer: Apply default pagination values
    
    alt invalid company_id
        gRPCServer-->>Client: Error: Company ID required
    else valid parameters
        gRPCServer->>UseCase: ListActivityLogs(company_id, page, limit)
        
        UseCase->>UseCase: Validate pagination parameters
        UseCase->>ArangoDB: GetByCompanyID(company_id, page, limit)
        
        ArangoDB->>ArangoDB: Execute paginated query
        ArangoDB->>ArangoDB: Count total records
        
        ArangoDB-->>UseCase: (activity_logs[], total_count)
        UseCase-->>gRPCServer: (activity_logs[], total, page, limit)
        gRPCServer-->>Client: ListActivityLogsResponse
    end
```


### 5. Error Handling and Retry Logic

```mermaid
sequenceDiagram
    participant Consumer as NATS Consumer
    participant WorkerPool
    participant ArangoDB
    participant DeadLetter as Dead Letter Queue

    Consumer->>WorkerPool: Process message (attempt 1)
    
    WorkerPool->>ArangoDB: Save activity log
    ArangoDB-->>WorkerPool: Error: Connection timeout
    
    WorkerPool->>Consumer: Job failed
    Consumer->>Consumer: Increment retry count
    
    alt retry_count < max_retries
        Consumer->>NATS: NAK (for redelivery)
        Note over Consumer: Wait for redelivery
        
        NATS->>Consumer: Redeliver message (attempt 2)
        Consumer->>WorkerPool: Process message (attempt 2)
        
        WorkerPool->>ArangoDB: Save activity log
        ArangoDB-->>WorkerPool: Success
        
        WorkerPool->>Consumer: Job completed
        Consumer->>NATS: ACK
    else retry_count >= max_retries
        Consumer->>DeadLetter: Send to dead letter queue
        Consumer->>NATS: ACK (to prevent infinite retries)
        
        Note over DeadLetter: Manual intervention required
    end
```

### 6. Monitoring and Tracing Flow

```mermaid
sequenceDiagram
    participant Client
    participant gRPCServer as gRPC Server
    participant Jaeger
    participant Prometheus

    Client->>gRPCServer: CreateActivityLog(request)
    
    Note over gRPCServer: Start trace span
    gRPCServer->>Jaeger: Create trace span "grpc.CreateActivityLog"
    
    gRPCServer->>Prometheus: Increment grpc_requests_total metric
    gRPCServer->>Prometheus: Start grpc_request_duration timer
    
    gRPCServer->>gRPCServer: Process request
    
    Note over gRPCServer: Business logic processing
    
    gRPCServer->>Prometheus: Record processing metrics
    gRPCServer->>Prometheus: Stop grpc_request_duration timer
    
    gRPCServer->>Jaeger: Finish trace span with status
    
    gRPCServer-->>Client: Response
```

### 7. Graceful Shutdown Process

```mermaid
sequenceDiagram
    participant OS
    participant Main
    participant gRPCServer as gRPC Server
    participant Consumer as NATS Consumer
    participant WorkerPool
    participant NATS

    OS->>Main: SIGTERM/SIGINT signal
    
    Main->>Main: Cancel context
    Main->>Consumer: Stop()
    
    Consumer->>WorkerPool: Stop worker pool
    
    par Graceful Worker Shutdown
        WorkerPool->>WorkerPool: Finish current jobs
        WorkerPool->>WorkerPool: Stop accepting new jobs
        WorkerPool-->>Consumer: All workers stopped
    and NATS Cleanup
        Consumer->>NATS: Unsubscribe from subjects
        Consumer->>NATS: Close connection
        NATS-->>Consumer: Connection closed
    end
    
    Consumer-->>Main: Consumer stopped
    
    Main->>gRPCServer: GracefulStop()
    gRPCServer->>gRPCServer: Finish current requests
    gRPCServer->>gRPCServer: Stop accepting new requests
    gRPCServer-->>Main: Server stopped
    
    Main->>Main: Cleanup resources
    Main->>OS: Exit process
```

## Flow Descriptions

### 1. **Synchronous gRPC Flow**
- Client makes direct gRPC calls to the service
- Immediate validation and response
- Data is stored in both ArangoDB and JSON files
- Events are published to NATS for further processing

### 2. **Asynchronous Event Processing**
- External services publish events to NATS
- Consumer picks up messages and processes them via worker pool
- Provides better throughput and resilience
- Failed messages are retried with exponential backoff

### 3. **Data Consistency**
- Primary storage in ArangoDB for queries  
- Event sourcing pattern for audit trail
- Strong consistency with single storage system

### 4. **Error Handling Strategy**
- Validation errors return immediately to client
- Infrastructure errors trigger retry logic
- Dead letter queue for failed messages
- Comprehensive logging and monitoring

### 5. **Observability**
- Distributed tracing with Jaeger
- Metrics collection with Prometheus
- Structured logging throughout the flow
- Health checks and monitoring endpoints