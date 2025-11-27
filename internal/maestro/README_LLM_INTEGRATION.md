# Flow API Reference
## LLM Output Format
Your task is to analyze user input and produce ONE of these three outputs:

**1. Ready to Execute** - All required parameters available:
```json
{
  "flow": "flow_name",
  "parameters": {"key": "value", ...}
}
```
**2. Missing Parameters** - Need more information:
```json
{
  "missing_parameters": ["param1", "param2"]
}
```
**3. Unsupported Flow**:
```json
{
  "error": "flow is currently not supported"
}
```
---
##  API
**Endpoint**: `POST http://.../api/v1/.../execute`
**Request**: `{"flow": "flow_name", "input": {...}}`
**Response**: `{"success": true, "output": {...}}` or `{"success": false, "error": "..."}`
---
## 1. search_business
**Purpose**: Find businesses by location and service type with optional availability filtering.
**Required**: `cities` (array), `labels` (array)
**Optional**: `start` (RFC3339, default: now), `end` (RFC3339, default: start+36h)
**Example**:
```json
{"flow": "search_business", "input": {"cities": ["netanya"], "labels": ["hair_salon"], "start": "2025-11-27T15:30:00Z", "end": "2025-11-27T19:30:00Z"}}
```
**Returns**: Array of businesses with `name`, `phones`, `branches` (each with `city`, `address`, `open_slots` containing `id`, `start`, `end`)
---
## 2. create_booking
**Purpose**: Book a time slot. Customers get pending status, admins/maintainers get confirmed.
**Required**: `requester_phone` (E.164), `requester_name` (string, optional for admins), `slot_id` (from search), `start_time` (RFC3339)
**Optional**: `end_time` (RFC3339, admin only, default: calculated from schedule)
**Example**:
```json
{"flow": "create_booking", "input": {"requester_phone": "+972501234567", "requester_name": "Sarah", "slot_id": "abc123...", "start_time": "2025-11-27T15:30:00Z"}}
```
**Returns**: Booking object with `id`, `status` (pending/confirmed), `start_time`, `end_time`, `participants`
---
## 3. get_daily_schedule
**Purpose**: View bookings for admin/maintainer's businesses.
**Required**: `phone` (E.164)
**Optional**: `cities` (array), `labels` (array), `start` (RFC3339, default: now), `end` (RFC3339, default: start+10h)
**Example**:
```json
{"flow": "get_daily_schedule", "input": {"phone": "+972501234567", "start": "2025-11-27T00:00:00Z", "end": "2025-11-27T23:59:59Z"}}
```
**Returns**: Object with `units` array, each containing `name`, `labels`, `schedules` with `bookings` (each has `start`, `end`, `label`, `participants`)
---
## 4. create_business_unit
**Purpose**: Register new business with schedules (one per city by default).
**Required**: `name` (string, 2-100 chars), `admin_phone` (E.164), `cities` (array, 1-50), `labels` (array, 1-10)
**Optional**: `time_zone` (IANA), `website_urls` (array), `maintainers` (object), `priority` (int), `start_of_day` (HH:MM), `end_of_day` (HH:MM), `working_days` (array), `schedule_time_zone` (IANA), `default_meeting_duration_min` (int, 5-480), `default_break_duration_min` (int, 0-480), `max_participants_per_slot` (int, 1-200), `exceptions` (array), `schedule_per_city` (bool, default: false)
**Example**:
```json
{"flow": "create_business_unit", "input": {"name": "Style Salon", "admin_phone": "+972501234567", "cities": ["netanya"], "labels": ["hair_salon"], "start_of_day": "09:00", "end_of_day": "19:00", "working_days": ["sunday","monday","tuesday","wednesday","thursday"], "default_meeting_duration_min": 45}}
```
**Returns**: `business_unit_id`, `schedule_ids` (array)
---
## Intent Classification
Match user intent to flow, then check if all required parameters are available:
**Search** (find, search, show, available, looking for) → `search_business`
- Required: `cities`, `labels`
- If missing: Output `{"missing_parameters": [...]}`
**Book** (book, reserve, schedule, appointment) → `create_booking`
- Required: `slot_id`, `start_time`, `requester_phone`, `requester_name`
- If missing: Output `{"missing_parameters": [...]}`
- Note: `slot_id` must come from previous search
**View** (my schedule, my appointments, what do I have) → `get_daily_schedule`
- Required: `phone`
- If missing: Output `{"missing_parameters": ["phone"]}`
**Register** (register, create business, set up) → `create_business_unit`
- Required: `name`, `admin_phone`, `cities`, `labels`
- If missing: Output `{"missing_parameters": [...]}`
**No match**: Output `{"error": "flow is currently not supported"}`
---
## Data Formats
**Phone**: E.164 format (`+972501234567`)
**Time**: RFC3339 (`2025-11-27T15:30:00Z`)
**Cities**: Lowercase with underscores (`tel_aviv`, `mevaseret_zion`)
**Labels**: Common mappings - "hair salon"→`hair_salon`, "dentist"→`dentist`, "software developer"→`software_engineering`
---
## Examples
**Example 1 - Ready to Execute**:
```
User: "Find hair salon in netanya available today 15:30-19:30"
LLM Output:
{
  "flow": "search_business",
  "parameters": {
    "cities": ["netanya"],
    "labels": ["hair_salon"],
    "start": "2025-11-27T15:30:00Z",
    "end": "2025-11-27T19:30:00Z"
  }
}
```
**Example 2 - Missing Parameters**:
```
User: "Book an appointment for me"
LLM Output:
{
  "missing_parameters": ["slot_id", "start_time", "requester_phone", "requester_name"]
}
```
**Example 3 - Unsupported Flow**:
```
User: "Cancel my appointment"
LLM Output:
{
  "error": "flow is currently not supported"
}
```
**Example 4 - Multi-Turn Conversation**:
```
Turn 1:
User: "Find hair salon in netanya available today 15:30-19:30"
LLM: {"flow": "search_business", "parameters": {...}}
[Execute and store slot_ids from results]
Turn 2:
User: "Book first one at 4pm, name Sarah +972501234567"
LLM: {"flow": "create_booking", "parameters": {"slot_id": "abc123", "start_time": "2025-11-27T16:00:00Z", "requester_name": "Sarah", "requester_phone": "+972501234567"}}
```