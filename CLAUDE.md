# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Product Vision

**Skejy** is a WhatsApp-native scheduling ecosystem that operates through chat-based interaction. It empowers small businesses to manage appointments effortlessly by providing an intuitive conversational interface for discovery and booking.

**Core Objectives:**
- **Eliminate friction** - No app or website required, WhatsApp only
- **Automate scheduling** - Virtual AI secretary handles scheduling and approvals
- **Flexible sessions** - Supports both 1:1 and multi-participant meetings
- **Smart discovery** - Search by location, service type, and availability

## Project Status

**Current State:** Foundation phase - MongoDB schema and migration system implemented.

**Implemented:**
- Data models (Business Units, Schedules, Bookings)
- MongoDB migration system with JSON schema validation
- Local development environment (Kind + MongoDB)

**Planned:**
- Microservices architecture (see below)
- WhatsApp integration
- AI-powered conversational interface
- Kafka event streaming
- Redis caching layer

## Architecture Overview

### Planned Microservices

1. **Gateway Service** - API gateway and request routing
2. **Business Units Service** - Manages business registration and profiles
3. **Booking Service** - Handles booking creation, updates, and cancellations
4. **Schedule Service** - Manages availability windows and time slots
5. **Search Service** - Location and service-based discovery
6. **Meetings Status Service** - Tracks meeting lifecycle and state
7. **Meetings Notifier Service** - Sends reminders and approval requests

### Infrastructure Stack

- **Database:** MongoDB (schema-validated, with custom migration system)
- **Message Broker:** Kafka (planned - for event streaming between services)
- **Cache:** Redis (planned - for search results and session data)
- **Container Orchestration:** Kubernetes (local development via Kind)
- **Language:** Go 1.21.6

### Core Domain Models

- **Business Units** (`pkg/model/business_unit.go`) - Organizations that manage schedules (cities, labels, maintainers, priority)
- **Schedules** (`pkg/model/schedule.go`) - Define availability windows (working days, hours, locations, capacity)
- **Bookings** (`pkg/model/booking.go`) - Individual appointments (time slots, participants, status)

## Feature Flows

### Business Owner Flows
1. **Registration** - Create business unit record via WhatsApp conversation
2. **View Daily Schedule** - Request current day's bookings and availability
3. **Create Meeting** - Book time slot for customer, system sends approval notification to customer
4. **Approve/Decline Customer Requests** - Receive notifications when customers request meetings

### Customer Flows
1. **Search & Discovery** - Search by city (location) and service labels, optionally filter by date/time range
2. **Create Meeting** - Request booking, system sends approval notification to business owner
3. **Receive Reminders** - Get notified 10 minutes before scheduled meetings

### System Automation
- AI secretary interprets natural language booking requests
- Automatic approval workflow management
- Smart notifications (10-min pre-meeting reminders)
- Real-time availability updates

## Design Principles

When implementing new features, follow these principles:

### Microservices Communication
- **Event-driven:** Use Kafka for asynchronous communication between services
- **Service boundaries:** Each service owns its data and exposes well-defined APIs
- **Eventual consistency:** Accept eventual consistency where appropriate for scalability

### Data Management
- **Schema validation:** All MongoDB collections must have JSON schema validators
- **Idempotent migrations:** Database changes must be safely repeatable
- **Index strategy:** Follow existing patterns (see Indexes section below)

### WhatsApp Integration
- **Conversational first:** All interactions should feel natural in chat
- **Context retention:** Maintain conversation state for multi-turn dialogues
- **Graceful degradation:** Handle WhatsApp API failures with user-friendly messages

### AI Secretary Behavior
- **Intent recognition:** Parse natural language to extract booking parameters (date, time, service)
- **Confirmation flow:** Always confirm before creating bookings
- **Ambiguity handling:** Ask clarifying questions when intent is unclear

## Development Commands

### Local Environment Setup
```bash
# Spin up full local environment (Kind cluster + MongoDB + migrations)
make local-up

# Individual components
make kind-up     # Start Kind cluster only
make mongo-up    # Deploy MongoDB only
make migrate     # Run database migrations only

# Tear down environment
make kind-down   # Delete Kind cluster
```

### Building and Running
```bash
# Build migration binary
go build -o bin/migrate ./cmd/migrate

# Run migrations manually (requires MONGO_URI env var)
MONGO_URI="mongodb://localhost:27017" go run cmd/migrate/main.go
```

### Testing
```bash
# Run all tests
go test ./...

# Test specific package
go test ./pkg/model
go test ./internal/migrations/mongo
```

## Current Implementation Details

### Directory Structure
```
skeji/
├── cmd/                      # Application entrypoints
│   └── migrate/              # Migration tool executable
├── internal/                 # Private application code
│   └── migrations/
│       └── mongo/            # MongoDB schema management
│           ├── migrate.go    # Migration orchestration logic
│           └── validators/   # JSON schema validators
├── pkg/                      # Public libraries
│   ├── model/                # Domain models (Business Unit, Schedule, Booking)
│   ├── errors/               # Error definitions
│   ├── http/                 # HTTP utilities
│   └── logger/               # Logging utilities
├── deployment/local/         # Local development infrastructure
│   ├── kind/                 # Kubernetes in Docker setup
│   ├── mongo/                # MongoDB Kubernetes manifests
│   └── migrate/              # Migration job configuration
├── build/docker/             # Dockerfiles
└── api/proto/                # Protocol buffer definitions (planned)
```

### Expected Structure for Microservices

As services are implemented, the structure will evolve to:
```
skeji/
├── cmd/
│   ├── gateway/              # API Gateway service
│   ├── business-units/       # Business Units service
│   ├── booking/              # Booking service
│   ├── schedule/             # Schedule service
│   ├── search/               # Search service
│   ├── meetings-status/      # Meetings Status service
│   ├── meetings-notifier/    # Meetings Notifier service
│   └── migrate/              # Migration tool
├── internal/
│   ├── gateway/              # Gateway-specific logic
│   ├── businessunits/        # Business units service logic
│   ├── booking/              # Booking service logic
│   ├── schedule/             # Schedule service logic
│   ├── search/               # Search service logic
│   ├── whatsapp/             # WhatsApp integration layer
│   ├── ai/                   # AI secretary logic
│   └── migrations/           # Database migrations
├── pkg/                      # Shared libraries across services
│   ├── model/                # Domain models
│   ├── kafka/                # Kafka client utilities
│   ├── redis/                # Redis client utilities
│   └── ...
└── api/proto/                # gRPC/protobuf definitions
```

### MongoDB Migration System

**Location:** `internal/migrations/mongo/`

Skeji uses a custom idempotent migration system that:
- Creates collections with JSON schema validators
- Applies and maintains indexes
- Logs migration runs in `_migrations` collection
- Never drops or loses data

**Key Principles:**
1. **All migrations are idempotent** - Safe to run multiple times
2. **Never make new fields required immediately** - Add as optional first
3. **Three-phase schema evolution:**
   - Phase 1: Add field to validator as optional
   - Phase 2: Backfill existing documents with default value
   - Phase 3: Mark field as required in validator
4. **Schema validators are strict** - Validation level is set to "strict"

**Migration Architecture:**
- `migrate.go` - Defines indexes and orchestrates collection setup
- `validators/*.go` - JSON schema definitions for each collection
- Each collection has its indexes and validators defined in `RunMigration()`

**Database:**
- Database name: `skeji`
- Collections: `Business_units`, `Schedules`, `Bookings`, `_migrations` (metadata)

### Indexes

**Business_units:**
- `admin_phone` (single)
- `maintainers` (single)
- `cities` + `labels` + `priority` (compound, priority descending)

**Schedules:**
- `_id` (single)
- `business_id` + `city` (compound)

**Bookings:**
- `business_id` + `schedule_id` + `start_time` + `end_time` (compound)
- `participants` + `start_time` (compound)

### Local Development Flow

1. **Environment Setup** - `make local-up` creates a Kind cluster with MongoDB and runs initial migrations
2. **Kubernetes Context** - Kind cluster named `skeji-local` with 1 control-plane + 2 worker nodes
3. **MongoDB Access** - MongoDB exposed on `localhost:27017` via port forwarding
4. **Namespaces:**
   - `mongo` - MongoDB deployment
   - `migration` - Migration job execution

### Migration Job Execution

When running `make migrate`:
1. Builds Docker image from `build/docker/migrate.Dockerfile`
2. Loads image into Kind cluster
3. Runs Kubernetes Job in `migration` namespace
4. Job connects to MongoDB using `MONGO_URI` environment variable
5. Executes `RunMigration()` which:
   - Ensures collections exist
   - Applies/updates validators
   - Creates indexes
   - Logs completion to `_migrations`

## Working with Schema Changes

**Adding a New Field:**
```bash
# 1. Edit validator file (e.g., validators/booking.go)
#    Add field to properties, NOT to required array

# 2. Run migration
make migrate

# 3. Create backfill script in cmd/backfill/ if needed

# 4. After backfill, add to required array and re-run migration
make migrate
```

**Adding a New Collection:**
1. Create model in `pkg/model/`
2. Create validator in `internal/migrations/mongo/validators/`
3. Add to `collections` map in `migrate.go` with indexes
4. Run `make migrate`

**Reference:** See detailed guide in `internal/migrations/mongo/README.md`

## Environment Variables

- `MONGO_URI` - MongoDB connection string (required for migrations)
- Additional env vars can be loaded from `.env.local` (automatically sourced by `deployment/local/scripts/local_env_up.sh`)

## Kubernetes Operations

```bash
# View MongoDB pod logs
kubectl logs -n mongo -l app=mongo

# Access MongoDB shell
kubectl exec -it -n mongo <pod-name> -- mongosh

# Check migration job status
kubectl get jobs -n migration
kubectl logs -n migration job/skeji-migrate

# View collections
kubectl exec -it -n mongo <pod-name> -- mongosh skeji --eval 'db.getCollectionNames()'

# Check validator
kubectl exec -it -n mongo <pod-name> -- mongosh skeji --eval 'db.getCollectionInfos({name: "Bookings"})[0].options.validator'
```

## Quick Reference: Data Models

### Business_units Collection
```
_id                 ObjectId
name                string
cities              []string
labels              []string
admin_phone         string
maintainers         []string
priority            int
time_zone           string
created_at          timestamp
```

**Indexes:**
- `{admin_phone: 1}`
- `{maintainers: 1}`
- `{cities: 1, labels: 1, priority: -1}` - Compound index for search optimization

### Schedules Collection
```
_id                              ObjectId
business_id                      string
name                             string
city                             string
address                          string
start_of_day                     string (HH:MM format)
end_of_day                       string (HH:MM format)
working_days                     []string (e.g., ["Monday", "Tuesday"])
default_meeting_duration_min     int
default_break_duration_min       int
max_participants_per_slot        int
exceptions                       []string (dates to exclude)
created_at                       timestamp
```

**Indexes:**
- `{_id: 1}`
- `{business_id: 1, city: 1}` - Compound index for filtering schedules by business and location

### Bookings Collection
```
_id              ObjectId
business_id      string
schedule_id      string
service_label    string
start_time       timestamp
end_time         timestamp
capacity         int
participants     []string (phone numbers)
status           string (pending/approved/declined/cancelled)
managed_by       string (who created the booking)
created_at       timestamp
```

**Indexes:**
- `{business_id: 1, schedule_id: 1, start_time: 1, end_time: 1}` - Compound index for availability checks
- `{participants: 1, start_time: 1}` - Compound index for user's upcoming meetings
