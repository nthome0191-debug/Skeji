# Kafka Consumer Groups - Quick Reference

## Consumer Group Naming Convention

Pattern: `<service-name>-<purpose>-consumer-group`

---

## All Consumer Groups

| # | Consumer Group | Consumes From | Service | Replicas | Key |
|---|----------------|---------------|---------|----------|-----|
| 1 | `gateway-consumer-group` | `ingress-requests-buffer` | Gateway | 3-6 | `user_phone_number` |
| 2 | `scrub-layer-consumer-group` | `gw-to-scrub-layer` | Scrub Layer | 3-6 | `user_phone_number` |
| 3 | `llm-layer-consumer-group` | `scrub-layer-to-llm-layer` | LLM Layer | 3-6 | `conversation_id` |
| 4 | `pipeline-executor-consumer-group` | `llm-to-pipeline-executor` | Pipeline Executor | 3-6 | `conversation_id` |
| 5 | `pipeline-executor-responses-consumer-group` | `business-units-to-pipeline-executor`<br>`schedules-to-pipeline-executor`<br>`bookings-to-pipeline-executor` | Pipeline Executor | 3-6 | `correlation_id` |
| 6 | `business-units-service-consumer-group` | `pipeline-executor-to-business-units` | Business Units | 2-3 | `correlation_id` |
| 7 | `schedules-service-consumer-group` | `pipeline-executor-to-schedules` | Schedules | 3-4 | `correlation_id` |
| 8 | `bookings-service-consumer-group` | `pipeline-executor-to-bookings` | Bookings | 4-6 | `correlation_id` |
| 9 | `notifier-service-scrub-consumer-group` | `scrub-layer-to-notifier` | Notifier | 2-3 | `user_phone_number` |
| 10 | `notifier-service-pipeline-consumer-group` | `pipeline-executor-to-notifier` | Notifier | 3-4 | `user_phone_number` |
| 11 | `notifier-service-events-consumer-group` | `domain-events-to-notifier` | Notifier | 3-4 | `event_id` |
| 12 | `manual-intervention-consumer-group` | `dlq-scrub-layer`<br>`dlq-llm-layer`<br>`dlq-pipeline-executor`<br>`dlq-domain-services` | Manual/Ops | 1 | `original_key` |

---

## Topic to Consumer Group Mapping

### Infrastructure Topics

```
ingress-requests-buffer
  └─ gateway-consumer-group (Gateway Service)

gw-to-scrub-layer
  └─ scrub-layer-consumer-group (Scrub Layer Service)

scrub-layer-to-llm-layer
  └─ llm-layer-consumer-group (LLM Layer Service)

scrub-layer-to-notifier
  └─ notifier-service-scrub-consumer-group (Notifier Service)

llm-to-pipeline-executor
  └─ pipeline-executor-consumer-group (Pipeline Executor Service)
```

### Domain Command Topics

```
pipeline-executor-to-business-units
  └─ business-units-service-consumer-group (Business Units Service)

pipeline-executor-to-schedules
  └─ schedules-service-consumer-group (Schedules Service)

pipeline-executor-to-bookings
  └─ bookings-service-consumer-group (Bookings Service)
```

### Domain Response Topics

```
business-units-to-pipeline-executor  ┐
schedules-to-pipeline-executor       ├─ pipeline-executor-responses-consumer-group
bookings-to-pipeline-executor        ┘   (Pipeline Executor Service)
```

### Notification Topics

```
pipeline-executor-to-notifier
  └─ notifier-service-pipeline-consumer-group (Notifier Service)

domain-events-to-notifier
  └─ notifier-service-events-consumer-group (Notifier Service)
```

### Dead Letter Queues

```
dlq-scrub-layer          ┐
dlq-llm-layer            ├─ manual-intervention-consumer-group
dlq-pipeline-executor    │   (Manual/Ops Team)
dlq-domain-services      ┘
```

---

## Kubernetes Environment Variables per Service

### Gateway Service
```yaml
KAFKA_BROKERS: skeji-kafka-kafka-brokers.kafka.svc:9092
KAFKA_CONSUMER_GROUP: gateway-consumer-group
KAFKA_CONSUME_FROM: ingress-requests-buffer
KAFKA_PUBLISH_TO: gw-to-scrub-layer
```

### Scrub Layer Service
```yaml
KAFKA_BROKERS: skeji-kafka-kafka-brokers.kafka.svc:9092
KAFKA_CONSUMER_GROUP: scrub-layer-consumer-group
KAFKA_CONSUME_FROM: gw-to-scrub-layer
KAFKA_PUBLISH_TO: scrub-layer-to-llm-layer,scrub-layer-to-notifier,dlq-scrub-layer
```

### LLM Layer Service
```yaml
KAFKA_BROKERS: skeji-kafka-kafka-brokers.kafka.svc:9092
KAFKA_CONSUMER_GROUP: llm-layer-consumer-group
KAFKA_CONSUME_FROM: scrub-layer-to-llm-layer
KAFKA_PUBLISH_TO: llm-to-pipeline-executor,dlq-llm-layer
```

### Pipeline Executor Service
```yaml
KAFKA_BROKERS: skeji-kafka-kafka-brokers.kafka.svc:9092

# Request consumer
KAFKA_REQUEST_CONSUMER_GROUP: pipeline-executor-consumer-group
KAFKA_REQUEST_TOPIC: llm-to-pipeline-executor

# Response consumer
KAFKA_RESPONSE_CONSUMER_GROUP: pipeline-executor-responses-consumer-group
KAFKA_RESPONSE_TOPICS: business-units-to-pipeline-executor,schedules-to-pipeline-executor,bookings-to-pipeline-executor

# Publishers
KAFKA_PUBLISH_COMMANDS_TO: pipeline-executor-to-business-units,pipeline-executor-to-schedules,pipeline-executor-to-bookings
KAFKA_PUBLISH_RESULTS_TO: pipeline-executor-to-notifier,dlq-pipeline-executor
```

### Business Units Service
```yaml
KAFKA_BROKERS: skeji-kafka-kafka-brokers.kafka.svc:9092
KAFKA_CONSUMER_GROUP: business-units-service-consumer-group
KAFKA_CONSUME_FROM: pipeline-executor-to-business-units
KAFKA_PUBLISH_TO: business-units-to-pipeline-executor,domain-events-to-notifier,dlq-domain-services
```

### Schedules Service
```yaml
KAFKA_BROKERS: skeji-kafka-kafka-brokers.kafka.svc:9092
KAFKA_CONSUMER_GROUP: schedules-service-consumer-group
KAFKA_CONSUME_FROM: pipeline-executor-to-schedules
KAFKA_PUBLISH_TO: schedules-to-pipeline-executor,domain-events-to-notifier,dlq-domain-services
```

### Bookings Service
```yaml
KAFKA_BROKERS: skeji-kafka-kafka-brokers.kafka.svc:9092
KAFKA_CONSUMER_GROUP: bookings-service-consumer-group
KAFKA_CONSUME_FROM: pipeline-executor-to-bookings
KAFKA_PUBLISH_TO: bookings-to-pipeline-executor,domain-events-to-notifier,dlq-domain-services
```

### Notifier Service
```yaml
KAFKA_BROKERS: skeji-kafka-kafka-brokers.kafka.svc:9092

# Multiple consumer groups for different sources
KAFKA_SCRUB_CONSUMER_GROUP: notifier-service-scrub-consumer-group
KAFKA_SCRUB_TOPIC: scrub-layer-to-notifier

KAFKA_PIPELINE_CONSUMER_GROUP: notifier-service-pipeline-consumer-group
KAFKA_PIPELINE_TOPIC: pipeline-executor-to-notifier

KAFKA_EVENTS_CONSUMER_GROUP: notifier-service-events-consumer-group
KAFKA_EVENTS_TOPIC: domain-events-to-notifier
```

---

## Monitoring Commands

### List all consumer groups
```bash
kubectl exec -it -n kafka skeji-kafka-dual-0 -- \
  bin/kafka-consumer-groups.sh \
  --bootstrap-server localhost:9092 \
  --list
```

### Check consumer group lag
```bash
kubectl exec -it -n kafka skeji-kafka-dual-0 -- \
  bin/kafka-consumer-groups.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --group bookings-service-consumer-group
```

### List all topics
```bash
kubectl exec -it -n kafka skeji-kafka-dual-0 -- \
  bin/kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --list
```

### View topic details
```bash
kubectl exec -it -n kafka skeji-kafka-dual-0 -- \
  bin/kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --describe \
  --topic pipeline-executor-to-bookings
```

### Reset consumer group offset (USE WITH CAUTION)
```bash
kubectl exec -it -n kafka skeji-kafka-dual-0 -- \
  bin/kafka-consumer-groups.sh \
  --bootstrap-server localhost:9092 \
  --group bookings-service-consumer-group \
  --reset-offsets \
  --to-earliest \
  --topic pipeline-executor-to-bookings \
  --execute
```
