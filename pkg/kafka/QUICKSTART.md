# Kafka Package - Quick Start Guide

## What Was Created

A complete, production-ready Kafka abstraction layer with:

```
pkg/kafka/
├── config.go              # Configuration (from env vars)
├── message.go             # Message types and builder
├── producer.go            # Producer wrapper
├── consumer.go            # Consumer wrapper
├── errors.go              # Error handling & classification
├── middleware/
│   ├── logging.go         # Logging middleware
│   └── metrics.go         # Metrics middleware
├── README.md              # Full documentation
└── QUICKSTART.md          # This file
```

## Benefits for Microservices

✅ **Zero Boilerplate** - 3 lines of code to publish/consume
✅ **Configuration-Driven** - All config via environment variables
✅ **Automatic Retries** - Transient errors retried automatically
✅ **Dead Letter Queue** - Failed messages preserved for investigation
✅ **Message Ordering** - Key-based partitioning guarantees ordering
✅ **Observability** - Built-in logging and metrics
✅ **Type Safety** - Compile-time checks for message structure

---

## Microservice Integration Examples

### Example 1: Bookings Service (Consumer)

```go
package main

import (
    "context"
    "log"
    "os"
    "skeji/pkg/kafka"
    "skeji/pkg/kafka/middleware"
)

type BookingCommand struct {
    Action string                 `json:"action"`
    Params map[string]interface{} `json:"params"`
}

func main() {
    // 1. Create consumer config from environment
    config, _ := kafka.NewConsumerConfigFromEnv(
        "pipeline-executor-to-bookings",
        "bookings-service-consumer-group",
    )
    config.DLQTopic = "dlq-domain-services"
    config.MaxRetries = 3

    // 2. Define message handler
    handler := func(ctx context.Context, msg kafka.Message) error {
        var cmd BookingCommand
        if err := msg.DecodeValue(&cmd); err != nil {
            return kafka.NewPermanentError("invalid format", err)
        }

        log.Printf("Processing: %s (correlation_id=%s)",
            cmd.Action, msg.GetCorrelationID())

        // Your business logic here
        return handleBookingCommand(ctx, cmd)
    }

    // 3. Create consumer with middleware
    consumer, _ := kafka.NewConsumer(config, handler)
    consumer.Use(middleware.LoggingConsumerMiddleware())
    consumer.Use(middleware.MetricsConsumerMiddleware())
    defer consumer.Close()

    // 4. Start consuming
    log.Println("Bookings service started...")
    consumer.Start(context.Background())
}

func handleBookingCommand(ctx context.Context, cmd BookingCommand) error {
    // Your domain logic
    return nil
}
```

### Example 2: Pipeline Executor (Producer + Consumer)

```go
package main

import (
    "context"
    "skeji/pkg/kafka"
    "skeji/pkg/kafka/middleware"
)

func main() {
    // Producer for sending commands to domain services
    producerConfig, _ := kafka.NewProducerConfigFromEnv("pipeline-executor-to-bookings")
    producerConfig.DLQTopic = "dlq-pipeline-executor"
    producer, _ := kafka.NewProducer(producerConfig)
    producer.Use(middleware.LoggingProducerMiddleware())
    defer producer.Close()

    // Consumer for receiving responses from domain services
    consumerConfig, _ := kafka.NewConsumerConfigFromEnv(
        "bookings-to-pipeline-executor",
        "pipeline-executor-responses-consumer-group",
    )
    handler := func(ctx context.Context, msg kafka.Message) error {
        // Handle response from bookings service
        correlationID := msg.GetCorrelationID()
        log.Printf("Received response for correlation_id=%s", correlationID)
        return nil
    }
    consumer, _ := kafka.NewConsumer(consumerConfig, handler)
    defer consumer.Close()

    // Send command
    msg := kafka.NewMessage().
        WithKey("booking-12345").
        WithCorrelationID("corr-abc-123").
        WithValue(map[string]interface{}{
            "action": "create_booking",
            "params": map[string]interface{}{
                "user_phone": "+972501234567",
            },
        }).
        Build()

    producer.Publish(context.Background(), msg)

    // Start consuming responses
    consumer.Start(context.Background())
}
```

### Example 3: Gateway Service (Simple Producer)

```go
package main

import (
    "context"
    "skeji/pkg/kafka"
)

func forwardMessageToScrubLayer(userPhone, messageText string) error {
    config, _ := kafka.NewProducerConfigFromEnv("gw-to-scrub-layer")
    producer, _ := kafka.NewProducer(config)
    defer producer.Close()

    msg := kafka.NewMessage().
        WithKey(userPhone).              // Partition by phone for ordering
        WithValue(map[string]string{
            "user_phone": userPhone,
            "message":    messageText,
        }).
        WithSource("gateway").
        Build()

    return producer.Publish(context.Background(), msg)
}
```

---

## Environment Variables Setup

### Kubernetes Deployment Example

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
        # Required
        - name: KAFKA_BROKERS
          value: "skeji-kafka-kafka-brokers.kafka.svc:9092"

        # Optional (with defaults)
        - name: KAFKA_CONSUMER_DLQ_TOPIC
          value: "dlq-domain-services"
        - name: KAFKA_CONSUMER_MAX_RETRIES
          value: "3"
        - name: KAFKA_ENABLE_MIDDLEWARE
          value: "true"
```

### Local Development (.env)

```bash
KAFKA_BROKERS=localhost:9092
KAFKA_CONSUMER_DLQ_TOPIC=dlq-domain-services
KAFKA_CONSUMER_MAX_RETRIES=3
```

---

## Message Patterns

### Pattern 1: Simple Event Publishing

```go
msg := kafka.NewMessage().
    WithKey("entity-id").
    WithValue(myStruct).
    WithEventType("booking.created").
    Build()

producer.Publish(ctx, msg)
```

### Pattern 2: Request-Response (with Correlation ID)

```go
// Service A: Send request
correlationID := uuid.New().String()
msg := kafka.NewMessage().
    WithKey(correlationID).
    WithCorrelationID(correlationID).
    WithValue(request).
    Build()

producer.Publish(ctx, msg)

// Service B: Send response with same correlation ID
response := kafka.NewMessage().
    WithKey(correlationID).
    WithCorrelationID(correlationID).  // Same ID!
    WithValue(result).
    Build()

producer.Publish(ctx, response)
```

### Pattern 3: Error Handling

```go
handler := func(ctx context.Context, msg kafka.Message) error {
    // Permanent error → DLQ immediately
    if invalidSchema {
        return kafka.NewPermanentError("invalid schema", err)
    }

    // Transient error → Retry then DLQ
    if dbError {
        return kafka.NewTransientError("db connection failed", err)
    }

    // Business error → No retry, no DLQ
    if conflict {
        return kafka.NewBusinessError("booking conflict", nil)
    }

    return nil
}
```

---

## Key Design Principles

### 1. Message Keys = Ordering Guarantee

Messages with the **same key** are guaranteed to be processed in order.

```go
// All messages for booking-123 will be processed in order
msg1 := kafka.NewMessage().WithKey("booking-123").WithValue(created).Build()
msg2 := kafka.NewMessage().WithKey("booking-123").WithValue(updated).Build()
msg3 := kafka.NewMessage().WithKey("booking-123").WithValue(cancelled).Build()
```

### 2. Correlation IDs = Request Tracing

Use correlation IDs to track requests across services.

```go
msg := kafka.NewMessage().
    WithCorrelationID("corr-abc-123").
    WithConversationID("conv-456").    // For multi-turn conversations
    Build()
```

### 3. Event Types = Clear Intent

Always set event types for clarity.

```go
msg := kafka.NewMessage().
    WithEventType("booking.created").
    WithEventType("booking.approved").
    WithEventType("booking.cancelled").
    Build()
```

---

## Testing Your Integration

### 1. Build Test

```bash
go build ./pkg/kafka/...
```

### 2. Run Consumer Locally

```bash
export KAFKA_BROKERS=localhost:9092
go run cmd/bookings-service/main.go
```

### 3. Check Consumer Lag

```bash
kubectl exec -it -n kafka skeji-kafka-dual-0 -- \
  bin/kafka-consumer-groups.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --group bookings-service-consumer-group
```

### 4. View DLQ Messages

```bash
kubectl exec -it -n kafka skeji-kafka-dual-0 -- \
  bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic dlq-domain-services \
  --from-beginning
```

---

## Migration Checklist

For each microservice:

- [ ] Import `skeji/pkg/kafka`
- [ ] Create config from environment (`NewProducerConfigFromEnv` or `NewConsumerConfigFromEnv`)
- [ ] Create producer/consumer with config
- [ ] Add middleware (logging, metrics)
- [ ] Define message handler (consumers only)
- [ ] Use `NewMessage()` builder for publishing
- [ ] Set `KAFKA_BROKERS` in Kubernetes deployment
- [ ] Set DLQ topic in config
- [ ] Test locally
- [ ] Deploy and monitor consumer lag

---

## Common Gotchas

### ❌ Don't forget to set message keys
```go
// BAD: No key = random partitioning = no ordering
msg := kafka.NewMessage().WithValue(data).Build()

// GOOD: Key ensures ordering
msg := kafka.NewMessage().WithKey("booking-123").WithValue(data).Build()
```

### ❌ Don't ignore correlation IDs
```go
// BAD: Can't trace request-response
msg := kafka.NewMessage().WithValue(data).Build()

// GOOD: Traceable across services
msg := kafka.NewMessage().WithCorrelationID(correlationID).WithValue(data).Build()
```

### ❌ Don't forget to close producers/consumers
```go
// GOOD: Always defer close
producer, _ := kafka.NewProducer(config)
defer producer.Close()
```

---

## Next Steps

1. **Read Full Documentation:** `pkg/kafka/README.md`
2. **Review Integration Guide:** `/deployment/local/kafka/KAFKA_INTEGRATION_GUIDE.md`
3. **Check Topic List:** `/deployment/local/kafka/kafka-topics.yaml`
4. **Apply Topics:** `kubectl apply -f deployment/local/kafka/kafka-topics.yaml`

---

## Support

For issues or questions:
1. Check `pkg/kafka/README.md` for detailed docs
2. Review examples in this file
3. Check Kafka cluster status: `kubectl get kafka -n kafka`
