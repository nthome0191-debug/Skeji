# Maestro REST API

The Maestro REST API provides HTTP endpoints to execute orchestration flows.

## Architecture

```
HTTP Request → Handler → Service → MaestroContext → Flow → Response
```

- **Handlers** (`handlers/`): Parse HTTP requests, validate input, return JSON responses
- **Service** (`service/`): Create MaestroContext and execute flows
- **Flows** (`flows/`): Business logic for each orchestration flow

## Endpoints

### Execute Flow
Execute a specific flow with input parameters.

**Endpoint:** `POST /api/v1/maestro/execute`

**Request Body:**
```json
{
  "flow": "create_business_unit",
  "input": {
    "name": "Salon Bella",
    "admin_phone": "+972501234567",
    "cities": ["Tel Aviv", "Haifa"],
    "labels": ["hair", "beauty"]
  }
}
```

**Response (Success):**
```json
{
  "success": true,
  "output": {
    "business_unit": { ... },
    "schedules": [ ... ]
  }
}
```

**Response (Error):**
```json
{
  "success": false,
  "error": "required param [name] is missing"
}
```

### List Available Flows
Get a list of all available flows.

**Endpoint:** `GET /api/v1/maestro/flows`

**Response:**
```json
{
  "flows": [
    "create_business_unit",
    "create_booking",
    "get_daily_schedule",
    "search_business"
  ]
}
```

### Health Check
Check if the Maestro service is healthy.

**Endpoint:** `GET /api/v1/maestro/health`

**Response:**
```json
{
  "status": "healthy"
}
```

## Available Flows

### create_business_unit
Creates a business unit with schedules for each city.

**Required Input:**
- `name` (string): Business unit name
- `admin_phone` (string): Admin phone number (E.164 format)
- `cities` ([]string): List of cities
- `labels` ([]string): List of service labels

**Optional Input:**
- `time_zone` (string): Business unit timezone
- `website_urls` ([]string): List of website URLs
- `maintainers` (map[string]string): Map of maintainer names to phones
- `priority` (int64): Priority level
- `start_of_day` (string): Schedule start time (HH:MM)
- `end_of_day` (string): Schedule end time (HH:MM)
- `working_days` ([]string): List of working days
- `schedule_time_zone` (string): Schedule timezone
- `default_meeting_duration_min` (int): Meeting duration
- `default_break_duration_min` (int): Break duration
- `max_participants_per_slot` (int): Max participants
- `exceptions` ([]string): Exception dates

**Output:**
- `business_unit`: Created business unit object
- `schedules`: Array of created schedules (one per city)

### create_booking
Creates a booking for a time slot.

**Required Input:**
- `requester_phone` (string): Requester's phone number
- `requester_name` (string): Requester's name
- `slot_id` (string): Opaque slot token
- `start_time` (string): Start time (RFC3339 format)

**Optional Input:**
- `end_time` (string): End time (only for maintainers)

**Output:**
- Booking created successfully

### get_daily_schedule
Gets the daily schedule for a maintainer.

**Required Input:**
- `maintainer_phone` (string): Maintainer's phone number

**Optional Input:**
- `start_time` (string): Start time (RFC3339 format, defaults to today 00:00)
- `end_time` (string): End time (RFC3339 format, defaults to today 23:59)

**Output:**
- `schedule`: Hierarchical daily schedule view

### search_business
Searches for available time slots.

**Required Input:**
- `cities` ([]string): List of cities to search in
- `labels` ([]string): List of service labels

**Optional Input:**
- `start_time` (string): Start time (RFC3339 format)
- `end_time` (string): End time (RFC3339 format)

**Output:**
- `businesses`: List of businesses with available slots

## Usage Example

### Setup Router
```go
package main

import (
    "net/http"
    "skeji/internal/maestro/api"
    "skeji/pkg/client"
    "skeji/pkg/logger"
)

func main() {
    log := logger.New()

    // Setup client
    client := client.NewClient()
    client.SetBusinessUnitClient("http://localhost:8080")
    client.SetScheduleClient("http://localhost:8080")
    client.SetBookingClient("http://localhost:8080")

    // Setup router
    router := api.SetupRouter(client, log)

    // Start server
    log.Info("Starting Maestro API server on :8090")
    http.ListenAndServe(":8090", router)
}
```

### Execute Flow via cURL
```bash
# Create a business unit
curl -X POST http://localhost:8090/api/v1/maestro/execute \
  -H "Content-Type: application/json" \
  -d '{
    "flow": "create_business_unit",
    "input": {
      "name": "Salon Bella",
      "admin_phone": "+972501234567",
      "cities": ["Tel Aviv", "Haifa"],
      "labels": ["hair", "beauty"],
      "start_of_day": "09:00",
      "end_of_day": "18:00",
      "working_days": ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday"],
      "default_meeting_duration_min": 60,
      "default_break_duration_min": 15,
      "max_participants_per_slot": 1
    }
  }'

# List available flows
curl http://localhost:8090/api/v1/maestro/flows

# Health check
curl http://localhost:8090/api/v1/maestro/health
```

## Adding New Flows

To add a new flow:

1. **Create the flow function** in `flows/`:
   ```go
   func MyNewFlow(ctx *maestro.MaestroContext) error {
       // Extract inputs
       param := ctx.ExtractString("param_name")

       // Execute logic
       // ...

       // Set output
       ctx.Output["result"] = result
       return nil
   }
   ```

2. **Register the flow** in `service/service.go`:
   ```go
   var flowRegistry = map[string]FlowHandler{
       "my_new_flow": flows.MyNewFlow,
       // ... existing flows
   }
   ```

3. **Document the flow** in this README

The flow will automatically be available via the API.
