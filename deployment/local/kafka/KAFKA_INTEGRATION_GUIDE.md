# Kafka Integration Guide - Skeji Microservices

## Overview

This guide explains how to integrate Kafka into each Skeji microservice, following industry best practices for event-driven architecture.

## Message Flow Architecture

```
WhatsApp Messages
      ↓
┌─────────────────┐
│  Gateway (GW)   │ → ingress-requests-buffer
└─────────────────┘
      ↓ gw-to-scrub-layer
┌─────────────────┐
│  Scrub Layer    │ ─┬→ scrub-layer-to-llm-layer
└─────────────────┘  └→ scrub-layer-to-notifier (declined msgs)
      ↓
┌─────────────────┐
│   LLM Layer     │ → llm-to-pipeline-executor
└─────────────────┘
      ↓
┌─────────────────────────┐
│  Pipeline Executor      │ ─┬→ pipeline-executor-to-business-units
└─────────────────────────┘  ├→ pipeline-executor-to-schedules
      ↑                      └→ pipeline-executor-to-bookings
      │
      └─ Receives responses from:
         • business-units-to-pipeline-executor
         • schedules-to-pipeline-executor
         • bookings-to-pipeline-executor
      ↓
┌─────────────────┐
│  Notifier       │ ← pipeline-executor-to-notifier
└─────────────────┘ ← domain-events-to-notifier
      ↓
   WhatsApp
```

---

## Consumer Groups by Service

### 1. Gateway Service

**Consumer Groups:**
```yaml
- gateway-consumer-group
  Consumes from: ingress-requests-buffer
  Publishes to: gw-to-scrub-layer
  Instances: 3-6 (scale with traffic)
  Partition strategy: Round-robin
```

**Responsibilities:**
- Consume incoming WhatsApp messages from buffer
- Extract user phone number and message content
- Forward enriched message to Scrub Layer
- Handle backpressure during traffic bursts

**Message Key:** `user_phone_number` (ensures ordering per user)

---

### 2. Scrub Layer Service

**Consumer Groups:**
```yaml
- scrub-layer-consumer-group
  Consumes from: gw-to-scrub-layer
  Publishes to:
    - scrub-layer-to-llm-layer (valid messages)
    - scrub-layer-to-notifier (invalid messages)
    - dlq-scrub-layer (processing failures)
  Instances: 3-6
  Partition strategy: Hash by user_phone_number
```

**Responsibilities:**
- Validate incoming messages (spam detection, format validation)
- Enrich messages with user context from MongoDB
- Route valid messages to LLM Layer
- Send rejection notifications for invalid messages
- Handle errors → DLQ

**Message Key:** `user_phone_number`

**Example Decision Flow:**
```go
if msg.IsSpam() {
    publishTo("scrub-layer-to-notifier", RejectionNotification{
        Reason: "spam_detected",
    })
} else if msg.IsValid() {
    enrichedMsg := enrichMessage(msg)
    publishTo("scrub-layer-to-llm-layer", enrichedMsg)
} else {
    publishTo("dlq-scrub-layer", msg)
}
```

---

### 3. LLM Layer Service

**Consumer Groups:**
```yaml
- llm-layer-consumer-group
  Consumes from: scrub-layer-to-llm-layer
  Publishes to:
    - llm-to-pipeline-executor (successful conversions)
    - dlq-llm-layer (LLM failures)
  Instances: 3-6
  Partition strategy: Hash by user_phone_number
```

**Responsibilities:**
- Convert natural language to REST API call sequences
- Extract intent and parameters (e.g., "book haircut tomorrow 3pm" → API calls)
- Maintain conversation context
- Handle LLM errors → DLQ

**Message Key:** `conversation_id` or `user_phone_number`

**Output Example:**
```json
{
  "conversation_id": "conv_12345",
  "user_phone": "+972501234567",
  "pipeline": [
    {
      "service": "schedules",
      "action": "search_availability",
      "params": {
        "city": "Tel Aviv",
        "service_label": "haircut",
        "date": "2025-11-18",
        "time": "15:00"
      }
    },
    {
      "service": "bookings",
      "action": "create_booking",
      "params": {
        "schedule_id": "${previous_step.schedule_id}",
        "start_time": "2025-11-18T15:00:00Z"
      }
    }
  ]
}
```

---

### 4. Pipeline Executor Service

**Consumer Groups:**
```yaml
- pipeline-executor-consumer-group
  Consumes from: llm-to-pipeline-executor
  Publishes to:
    - pipeline-executor-to-business-units
    - pipeline-executor-to-schedules
    - pipeline-executor-to-bookings
  Instances: 3-6
  Partition strategy: Hash by conversation_id

- pipeline-executor-responses-consumer-group
  Consumes from:
    - business-units-to-pipeline-executor
    - schedules-to-pipeline-executor
    - bookings-to-pipeline-executor
  Publishes to:
    - pipeline-executor-to-notifier (final result)
    - dlq-pipeline-executor (orchestration failures)
  Instances: 3-6
  Partition strategy: Hash by correlation_id
```

**Responsibilities:**
- Orchestrate multi-step REST API call sequences
- Send commands to domain services via Kafka
- Collect responses from domain services
- Aggregate results and send final response to Notifier
- Handle timeouts and partial failures → DLQ

**Message Key (outgoing):** `correlation_id` (for tracking request-response pairs)

**Example Orchestration:**
```go
type PipelineExecution struct {
    CorrelationID   string
    ConversationID  string
    Steps           []Step
    Responses       map[string]Response
    CurrentStep     int
}

func (pe *PipelineExecutor) Execute(ctx context.Context, pipeline Pipeline) {
    correlationID := generateCorrelationID()

    for _, step := range pipeline.Steps {
        command := Command{
            CorrelationID: correlationID,
            ConversationID: pipeline.ConversationID,
            Action: step.Action,
            Params: step.Params,
        }

        // Publish to domain service topic
        topicName := fmt.Sprintf("pipeline-executor-to-%s", step.Service)
        publishTo(topicName, correlationID, command)

        // Wait for response (handled by response consumer)
    }
}
```

---

### 5. Business Units Service

**Consumer Groups:**
```yaml
- business-units-service-consumer-group
  Consumes from: pipeline-executor-to-business-units
  Publishes to:
    - business-units-to-pipeline-executor (responses)
    - domain-events-to-notifier (events like new business registered)
    - dlq-domain-services (processing failures)
  Instances: 2-3
  Partition strategy: Hash by business_id
```

**Responsibilities:**
- Handle business unit registration
- Process queries for business information
- Publish domain events (business created, updated)
- Send responses back to Pipeline Executor

**Message Key (response):** `correlation_id`

---

### 6. Schedules Service

**Consumer Groups:**
```yaml
- schedules-service-consumer-group
  Consumes from: pipeline-executor-to-schedules
  Publishes to:
    - schedules-to-pipeline-executor (responses)
    - domain-events-to-notifier (availability changes)
    - dlq-domain-services (processing failures)
  Instances: 3-4
  Partition strategy: Hash by schedule_id
```

**Responsibilities:**
- Search for available time slots
- Create/update schedules
- Query schedule details
- Publish availability change events
- Send responses back to Pipeline Executor

**Message Key (response):** `correlation_id`

---

### 7. Bookings Service

**Consumer Groups:**
```yaml
- bookings-service-consumer-group
  Consumes from: pipeline-executor-to-bookings
  Publishes to:
    - bookings-to-pipeline-executor (responses)
    - domain-events-to-notifier (booking created, approved, etc.)
    - dlq-domain-services (processing failures)
  Instances: 4-6 (highest volume)
  Partition strategy: Hash by booking_id
```

**Responsibilities:**
- Create bookings
- Update booking status (confirm, cancel, complete)
- Query booking details
- Publish booking lifecycle events
- Send responses back to Pipeline Executor

**Message Key (response):** `correlation_id`

**Domain Events Published:**
- `booking.created` → notifier sends approval request
- `booking.approved` → notifier schedules reminder
- `booking.cancelled` → notifier sends cancellation notice

---

### 8. Notifier Service

**Consumer Groups:**
```yaml
- notifier-service-scrub-consumer-group
  Consumes from: scrub-layer-to-notifier
  Publishes to: WhatsApp API
  Instances: 2-3

- notifier-service-pipeline-consumer-group
  Consumes from: pipeline-executor-to-notifier
  Publishes to: WhatsApp API
  Instances: 3-4

- notifier-service-events-consumer-group
  Consumes from: domain-events-to-notifier
  Publishes to: WhatsApp API
  Instances: 3-4
```

**Responsibilities:**
- Send WhatsApp messages to users
- Handle approval requests
- Schedule and send reminders (10 min before booking)
- Send rejection/error notifications
- Handle WhatsApp API failures

**Note:** Multiple consumer groups allow independent scaling and offset management for different notification types.

---

## Kafka Client Configuration (Go)

### Environment Variables

Each service should expose these environment variables:

```bash
KAFKA_BROKERS=skeji-kafka-kafka-brokers.kafka.svc:9092
KAFKA_CONSUMER_GROUP=<service-specific-group-name>
KAFKA_CONSUME_FROM_TOPIC=<topic-name>
KAFKA_PUBLISH_TO_TOPICS=<comma-separated-topics>
```

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
        - name: KAFKA_BROKERS
          value: "skeji-kafka-kafka-brokers.kafka.svc:9092"
        - name: KAFKA_CONSUMER_GROUP
          value: "bookings-service-consumer-group"
        - name: KAFKA_CONSUME_FROM_TOPIC
          value: "pipeline-executor-to-bookings"
        - name: MONGO_URI
          valueFrom:
            secretKeyRef:
              name: mongodb-credentials
              key: connection-string
```

---

## Monitoring & Observability

### Consumer Lag Monitoring

Check consumer lag for all groups:

```bash
kubectl exec -it -n kafka skeji-kafka-dual-0 -- bin/kafka-consumer-groups.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --all-groups
```

Check specific consumer group:

```bash
kubectl exec -it -n kafka skeji-kafka-dual-0 -- bin/kafka-consumer-groups.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --group bookings-service-consumer-group
```

### Key Metrics to Track

1. **Consumer Lag** - Messages waiting to be processed
2. **Processing Rate** - Messages/second per consumer
3. **Error Rate** - Failed message processing
4. **DLQ Size** - Messages in dead letter queues
5. **End-to-End Latency** - Time from ingress to final response

### Alerting Thresholds

- Consumer lag > 10,000 messages → Scale up consumers
- DLQ message rate > 1% → Investigate errors
- Processing time > 5 seconds (p95) → Optimize handler logic

---

## Message Ordering Guarantees

### Ordering Rules

1. **Messages with same key** → guaranteed ordering within partition
2. **Messages across partitions** → no ordering guarantee
3. **Multiple consumer instances** → each instance processes different partitions

### Key Selection Strategy

| Service | Message Key | Ordering Guarantee |
|---------|-------------|-------------------|
| Gateway | `user_phone_number` | All messages from same user are ordered |
| Scrub Layer | `user_phone_number` | Same |
| LLM Layer | `conversation_id` | All messages in same conversation are ordered |
| Pipeline Executor | `correlation_id` | All steps of same pipeline are ordered |
| Domain Services (response) | `correlation_id` | Responses match request order |
| Domain Services (events) | `entity_id` (e.g., `booking_id`) | All events for same entity are ordered |

---

## Error Handling Strategy

### Error Classification

1. **Transient Errors** (network issues, timeouts)
   - Retry up to 3 times with exponential backoff
   - Don't commit offset until successful

2. **Permanent Errors** (schema mismatch, invalid data)
   - Log error with full context
   - Publish to DLQ immediately
   - Commit offset (don't block the queue)

3. **Business Logic Errors** (booking conflict, invalid time slot)
   - Publish error response to notifier
   - Commit offset

### DLQ Processing

Dead letter queues should be monitored and processed manually or with automated replay:

```bash
# List messages in DLQ
kubectl exec -it -n kafka skeji-kafka-dual-0 -- bin/kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic dlq-pipeline-executor \
  --from-beginning
```

---

## Deployment Checklist

- [ ] Kafka cluster is running (`kubectl get kafka -n kafka`)
- [ ] All topics are created (`kubectl get kafkatopics -n kafka`)
- [ ] Service has KAFKA_BROKERS environment variable
- [ ] Service has unique KAFKA_CONSUMER_GROUP name
- [ ] Producer uses appropriate message keys for ordering
- [ ] Consumer commits offsets only after successful processing
- [ ] Error handling includes DLQ publishing
- [ ] Metrics are exposed (consumer lag, error rate)
- [ ] Logging includes correlation_id for request tracing

---

## Next Steps

1. **Implement Shared Kafka Package** (`pkg/kafka/`)
   - Producer wrapper
   - Consumer wrapper
   - Event schemas
   - Middleware (logging, metrics)

2. **Add Kafka to Each Service**
   - Import `pkg/kafka`
   - Configure consumer groups
   - Implement message handlers
   - Add error handling and DLQ logic

3. **Set Up Monitoring**
   - Consumer lag dashboards
   - Alert on high lag or error rates
   - Trace end-to-end message flow

4. **Load Testing**
   - Test backpressure handling
   - Verify partition rebalancing
   - Validate message ordering
