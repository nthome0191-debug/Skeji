# Kafka Package - Usage Guide

This package provides a clean, production-ready abstraction over Kafka for Skeji microservices. It handles producers, consumers, error handling, retries, DLQ, and observability.

## Features

- ✅ **Simple API** - Minimal code to publish/consume messages
- ✅ **Configuration via Environment Variables** - Zero hardcoding
- ✅ **Automatic Retries** - Transient errors are retried automatically
- ✅ **Dead Letter Queue (DLQ)** - Failed messages go to DLQ
- ✅ **Message Ordering** - Key-based partitioning ensures ordering
- ✅ **Middleware Support** - Logging, metrics, tracing
- ✅ **Graceful Shutdown** - Proper resource cleanup
- ✅ **Observability** - Built-in metrics and logging

---

## Installation

```bash
go get github.com/segmentio/kafka-go
go get github.com/google/uuid
```

---

## Quick Start

### Producer Example

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/nataliaharoni/skeji/pkg/kafka"
    "github.com/nataliaharoni/skeji/pkg/kafka/middleware"
)

func main() {
    // Set environment variables
    os.Setenv("KAFKA_BROKERS", "skeji-kafka-kafka-brokers.kafka.svc:9092")

    // Create producer config from environment
    config, err := kafka.NewProducerConfigFromEnv("pipeline-executor-to-bookings")
    if err != nil {
        log.Fatal(err)
    }

    // Set DLQ topic
    config.DLQTopic = "dlq-domain-services"

    // Create producer
    producer, err := kafka.NewProducer(config)
    if err != nil {
        log.Fatal(err)
    }
    defer producer.Close()

    // Add middleware (optional)
    producer.Use(middleware.LoggingProducerMiddleware())
    producer.Use(middleware.MetricsProducerMiddleware())

    // Build and publish a message
    msg := kafka.NewMessage().
        WithKey("booking-12345").                  // Partition key
        WithValue(map[string]interface{}{          // Your payload
            "action": "create_booking",
            "params": map[string]interface{}{
                "user_phone": "+972501234567",
                "start_time": "2025-11-18T15:00:00Z",
            },
        }).
        WithCorrelationID("corr-abc-123").         // For request-response tracking
        WithEventType("booking.create.requested"). // Event type
        WithSource("pipeline-executor").           // Source service
        Build()

    // Publish
    ctx := context.Background()
    if err := producer.Publish(ctx, msg); err != nil {
        log.Printf("Failed to publish: %v", err)
    } else {
        log.Println("Message published successfully!")
    }
}
```

### Consumer Example

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/nataliaharoni/skeji/pkg/kafka"
    "github.com/nataliaharoni/skeji/pkg/kafka/middleware"
)

// Your domain event
type BookingCommand struct {
    Action string                 `json:"action"`
    Params map[string]interface{} `json:"params"`
}

func main() {
    // Set environment variables
    os.Setenv("KAFKA_BROKERS", "skeji-kafka-kafka-brokers.kafka.svc:9092")

    // Create consumer config from environment
    config, err := kafka.NewConsumerConfigFromEnv(
        "pipeline-executor-to-bookings",     // Topic
        "bookings-service-consumer-group",   // Consumer group
    )
    if err != nil {
        log.Fatal(err)
    }

    // Set DLQ and retry config
    config.DLQTopic = "dlq-domain-services"
    config.MaxRetries = 3

    // Define message handler
    handler := func(ctx context.Context, msg kafka.Message) error {
        var cmd BookingCommand
        if err := msg.DecodeValue(&cmd); err != nil {
            return kafka.NewPermanentError("invalid message format", err)
        }

        log.Printf("Processing command: %s", cmd.Action)

        // Your business logic here
        if cmd.Action == "create_booking" {
            return handleCreateBooking(ctx, cmd.Params)
        }

        return nil
    }

    // Create consumer
    consumer, err := kafka.NewConsumer(config, handler)
    if err != nil {
        log.Fatal(err)
    }
    defer consumer.Close()

    // Add middleware (optional)
    consumer.Use(middleware.LoggingConsumerMiddleware())
    consumer.Use(middleware.MetricsConsumerMiddleware())

    // Start consuming
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Handle graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

    go func() {
        <-sigChan
        log.Println("Shutting down consumer...")
        cancel()
    }()

    log.Println("Consumer started, waiting for messages...")
    if err := consumer.Start(ctx); err != nil && err != context.Canceled {
        log.Printf("Consumer error: %v", err)
    }
}

func handleCreateBooking(ctx context.Context, params map[string]interface{}) error {
    // Your business logic
    log.Printf("Creating booking with params: %+v", params)
    return nil
}
```

---

## Environment Variables

### Required

| Variable | Description | Example |
|----------|-------------|---------|
| `KAFKA_BROKERS` | Comma-separated list of Kafka brokers | `broker1:9092,broker2:9092` |

### Producer (Optional)

| Variable | Default | Description |
|----------|---------|-------------|
| `KAFKA_PRODUCER_MAX_ATTEMPTS` | `3` | Max retries for failed writes |
| `KAFKA_PRODUCER_BATCH_TIMEOUT` | `10ms` | Time to wait before sending batch |
| `KAFKA_PRODUCER_REQUIRE_ACKS` | `-1` | Required acks (-1=all, 0=none, 1=leader) |
| `KAFKA_PRODUCER_COMPRESSION` | `snappy` | Compression type (none, gzip, snappy, lz4, zstd) |
| `KAFKA_PRODUCER_ASYNC` | `false` | Async writes (higher throughput, less durability) |
| `KAFKA_PRODUCER_DLQ_TOPIC` | `""` | Dead letter queue topic |

### Consumer (Optional)

| Variable | Default | Description |
|----------|---------|-------------|
| `KAFKA_CONSUMER_START_OFFSET` | `-1` | Start offset (-1=newest, -2=oldest) |
| `KAFKA_CONSUMER_MIN_BYTES` | `1` | Min bytes per fetch |
| `KAFKA_CONSUMER_MAX_BYTES` | `10MB` | Max bytes per fetch |
| `KAFKA_CONSUMER_MAX_WAIT` | `500ms` | Max wait for min bytes |
| `KAFKA_CONSUMER_COMMIT_INTERVAL` | `1s` | Offset commit interval |
| `KAFKA_CONSUMER_HEARTBEAT_INTERVAL` | `3s` | Consumer heartbeat interval |
| `KAFKA_CONSUMER_SESSION_TIMEOUT` | `10s` | Session timeout |
| `KAFKA_CONSUMER_REBALANCE_TIMEOUT` | `60s` | Rebalance timeout |
| `KAFKA_CONSUMER_DLQ_TOPIC` | `""` | Dead letter queue topic |
| `KAFKA_CONSUMER_MAX_RETRIES` | `3` | Max retries before DLQ |

---

## Message Headers

Standard headers automatically added/available:

| Header | Description |
|--------|-------------|
| `event-id` | Unique event ID (auto-generated UUID) |
| `event-type` | Event type (e.g., "booking.created") |
| `correlation-id` | For request-response tracking |
| `conversation-id` | For multi-turn conversations |
| `schema-version` | Schema version (e.g., "v1") |
| `source` | Source service name |
| `timestamp` | Event timestamp (RFC3339) |
| `retry-count` | Number of retries |

---

## Error Handling

### Error Types

```go
// Transient errors are retried automatically
kafka.NewTransientError("database connection failed", err)

// Permanent errors go directly to DLQ
kafka.NewPermanentError("invalid message schema", err)

// Business errors (not retried, not sent to DLQ)
kafka.NewBusinessError("booking conflict", err)
```

### Automatic Retry Logic

1. **Transient errors** → Retry up to `MaxRetries`
2. **Permanent errors** → Send to DLQ immediately
3. **Max retries exceeded** → Send to DLQ

The consumer automatically classifies errors based on error messages:

**Transient:**
- connection refused, timeout, deadline exceeded
- network is unreachable, broken pipe
- i/o timeout, temporary failure

**Permanent:**
- invalid message, schema mismatch
- deserialization failed, unknown topic

---

## Middleware

Middleware allows you to intercept and process messages before/after the handler.

### Built-in Middleware

```go
// Logging middleware
producer.Use(middleware.LoggingProducerMiddleware())
consumer.Use(middleware.LoggingConsumerMiddleware())

// Metrics middleware
producer.Use(middleware.MetricsProducerMiddleware())
consumer.Use(middleware.MetricsConsumerMiddleware())
```

### Custom Middleware

```go
// Producer middleware
func CustomProducerMiddleware() kafka.ProducerMiddleware {
    return func(ctx context.Context, msg kafka.Message, next func(context.Context, kafka.Message) error) error {
        // Before publish
        log.Println("Before publish")

        err := next(ctx, msg)

        // After publish
        log.Println("After publish")

        return err
    }
}

// Consumer middleware
func CustomConsumerMiddleware() kafka.ConsumerMiddleware {
    return func(ctx context.Context, msg kafka.Message, next kafka.MessageHandler) error {
        // Before processing
        log.Println("Before processing")

        err := next(ctx, msg)

        // After processing
        log.Println("After processing")

        return err
    }
}
```

---

## Message Ordering

Messages with the **same key** are guaranteed to be ordered (within a partition).

### Example: Ensure all messages for a booking are ordered

```go
bookingID := "booking-12345"

msg := kafka.NewMessage().
    WithKey(bookingID).  // All messages with this key go to same partition
    WithValue(event).
    Build()
```

### Recommended Key Patterns

| Service | Key | Ordering Guarantee |
|---------|-----|-------------------|
| Gateway | `user_phone_number` | All messages from user are ordered |
| LLM Layer | `conversation_id` | All turns in conversation are ordered |
| Pipeline Executor | `correlation_id` | All steps in pipeline are ordered |
| Domain Services | `entity_id` (e.g., `booking_id`) | All events for entity are ordered |

---

## Best Practices

### 1. Always Use Correlation IDs

```go
msg := kafka.NewMessage().
    WithCorrelationID("corr-abc-123").
    WithKey("booking-12345").
    WithValue(event).
    Build()
```

This allows end-to-end request tracing.

### 2. Use Fluent Message Builder

```go
msg := kafka.NewMessage().
    WithKey("booking-123").
    WithValue(event).
    WithEventType("booking.created").
    WithCorrelationID(correlationID).
    WithSource("bookings-service").
    WithSchemaVersion("v1").
    Build()
```

### 3. Handle Errors Appropriately

```go
func handler(ctx context.Context, msg kafka.Message) error {
    // Permanent error (schema mismatch)
    if err := msg.DecodeValue(&event); err != nil {
        return kafka.NewPermanentError("invalid schema", err)
    }

    // Transient error (database connection)
    if err := db.Save(event); err != nil {
        return kafka.NewTransientError("database error", err)
    }

    // Business error (conflict)
    if exists {
        return kafka.NewBusinessError("booking already exists", nil)
    }

    return nil
}
```

### 4. Graceful Shutdown

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Capture shutdown signals
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

go func() {
    <-sigChan
    cancel() // This will stop the consumer gracefully
}()

consumer.Start(ctx)
```

### 5. Monitor Metrics

```go
// Get metrics
metrics := middleware.GetMetrics()

// Print metrics periodically
ticker := time.NewTicker(30 * time.Second)
go func() {
    for range ticker.C {
        metrics.PrintMetrics()
    }
}()
```

---

## Testing

### Unit Testing Producer

```go
func TestPublishMessage(t *testing.T) {
    // Use environment variables or mock config
    config := &kafka.ProducerConfig{
        Config: kafka.Config{
            Brokers: []string{"localhost:9092"},
        },
        Topic:       "test-topic",
        MaxAttempts: 3,
    }

    producer, err := kafka.NewProducer(config)
    if err != nil {
        t.Fatal(err)
    }
    defer producer.Close()

    msg := kafka.NewMessage().
        WithKey("test-key").
        WithValue(map[string]string{"foo": "bar"}).
        Build()

    err = producer.Publish(context.Background(), msg)
    if err != nil {
        t.Errorf("Failed to publish: %v", err)
    }
}
```

---

## Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: bookings-service
spec:
  replicas: 4
  template:
    spec:
      containers:
      - name: bookings-service
        image: skeji/bookings-service:latest
        env:
        - name: KAFKA_BROKERS
          value: "skeji-kafka-kafka-brokers.kafka.svc:9092"
        - name: KAFKA_CONSUMER_DLQ_TOPIC
          value: "dlq-domain-services"
        - name: KAFKA_CONSUMER_MAX_RETRIES
          value: "3"
```

---

## Troubleshooting

### Issue: Consumer lag is growing

**Solution:** Scale up consumer replicas. Kafka will automatically rebalance partitions.

```bash
kubectl scale deployment bookings-service --replicas=6
```

### Issue: Messages going to DLQ

**Solution:** Check DLQ for error messages:

```bash
kubectl exec -it -n kafka skeji-kafka-dual-0 -- \
  bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic dlq-domain-services \
  --from-beginning
```

### Issue: Producer timeout

**Solution:** Check Kafka broker connectivity:

```bash
kubectl get svc -n kafka
kubectl logs -n kafka skeji-kafka-dual-0
```

---

## Advanced: Request-Response Pattern

For synchronous request-response over Kafka:

```go
// Producer (Pipeline Executor)
correlationID := uuid.New().String()
msg := kafka.NewMessage().
    WithCorrelationID(correlationID).
    WithKey(correlationID).
    WithValue(command).
    Build()

producer.Publish(ctx, msg)

// Wait for response with matching correlation ID
// (requires subscribing to response topic)
```

---

## Migration from Direct kafka-go Usage

**Before:**
```go
writer := kafka.NewWriter(kafka.WriterConfig{
    Brokers: []string{"localhost:9092"},
    Topic:   "my-topic",
})
writer.WriteMessages(ctx, kafka.Message{
    Key:   []byte("key"),
    Value: []byte("value"),
})
```

**After:**
```go
config, _ := kafka.NewProducerConfigFromEnv("my-topic")
producer, _ := kafka.NewProducer(config)
msg := kafka.NewMessage().WithKey("key").WithValue("value").Build()
producer.Publish(ctx, msg)
```

---

## Summary

This Kafka package provides:

1. **Simple API** - Publish/consume in 3 lines of code
2. **Configuration-driven** - All config via environment variables
3. **Production-ready** - Retries, DLQ, metrics, logging
4. **Zero boilerplate** - Microservices focus on business logic

For questions or issues, see `/deployment/local/kafka/KAFKA_INTEGRATION_GUIDE.md`.
