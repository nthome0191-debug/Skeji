# Code Review - Top 10 Critical Issues

**Review Date:** 2025-11-05
**Scope:** Production code (cmd/, internal/, pkg/) - Tests excluded
**Total Issues Found:** 21 (Showing top 10 most critical)

---

## 🔴 CRITICAL - DEPLOY BLOCKERS

## 🟠 HIGH PRIORITY

### 4. Phone Number Sanitization Security Flaw

**File:** `pkg/sanitizer/phone.go:8-29`

**Problem:**
```go
func NormalizePhone(phone string) string {
    // ... extracts all digits ...
    if normalized != "" {
        return "+" + normalized  // ⚠️ Blindly prepends +
    }
    return ""
}
```

**Impact:**
- Input: "+1-800-FLOWERS" becomes "+18003569377" (incorrect)
- Input: "972501234567" becomes "+972501234567" (may be invalid)
- **Authentication/authorization bypass**: Users could register with malformed numbers
- Database inconsistencies

**Fix:**
Use proper phone validation library (e.g., `github.com/nyaruka/phonenumbers`) to parse and normalize according to E.164 standard.

---

### 5. Race Condition in Rate Limiter

**File:** `pkg/middleware/rate_limit.go:62-88`

**Problem:**
The `Allow` method holds a write lock throughout timestamp validation, while cleanup goroutine also accesses the map.

**Impact:**
- High lock contention during traffic spikes
- Potential deadlocks during high load
- Performance degradation for all requests
- Race detector warnings

**Fix:**
Use `sync.RWMutex` more effectively or implement concurrent-safe map with read locks for checking, write locks only for modifications.

---

### 6. Timeout Middleware Race Condition

**File:** `pkg/middleware/timeout.go:74-81`

**Problem:**
```go
case <-ctx.Done():
    tw.timeout()
    w.Header().Set("Content-Type", "application/json")  // ⚠️ Race condition
    w.WriteHeader(http.StatusServiceUnavailable)
    _, _ = w.Write([]byte(`{"error":"Request timeout"}`))
```

Both the request handler and timeout handler can write to the response simultaneously.

**Impact:**
- HTTP panic: "http: multiple response.WriteHeader calls"
- Inconsistent responses sent to clients
- Application crashes

**Fix:**
```go
case <-ctx.Done():
    tw.timeout()
    tw.mu.Lock()
    if !tw.written {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusServiceUnavailable)
        _, _ = w.Write([]byte(`{"error":"Request timeout"}`))
    }
    tw.mu.Unlock()
```

---

### 7. Priority Field Type Mismatch

**File:** `internal/migrations/mongo/validators/business_unit.go:17` and `pkg/model/business_unit.go:12`

**Problem:**
```go
// MongoDB validator expects int64
"priority": bson.M{"bsonType": "long"},

// Go model uses int (32-bit on 32-bit systems)
Priority int `bson:"priority" validate:"omitempty,min=0"`
```

**Impact:**
- Data corruption on 32-bit systems
- Priority values could be truncated or rejected
- Validation failures

**Fix:**
```go
// In model
Priority int64 `bson:"priority" validate:"omitempty,min=0"`
```

---

### 8. Search Method Missing Limit - **DoS VULNERABILITY**

**File:** `internal/businessunits/repository/business_unit.go:165-191`

**Problem:**
```go
cursor, err := r.collection.Find(ctx, filter, opts)  // ⚠️ No limit
```

**Impact:**
- DoS vulnerability: attacker can request all business units
- Memory exhaustion if database has millions of records
- Network bandwidth abuse
- Slow API responses

**Fix:**
```go
// Add limit parameter or enforce max result size
opts.SetLimit(1000)  // Or make it configurable
```

---

### 9. Idempotency Store Memory Leak

**File:** `pkg/middleware/idempotency.go:50-52`

**Problem:**
```go
func (s *InMemoryIdempotencyStore) Get(key string) (*CachedResponse, bool) {
    // ...
    if time.Since(response.CreatedAt) > s.ttl {
        return nil, false  // ⚠️ Returns false but doesn't delete
    }
    return response, true
}
```

**Impact:**
- Memory leak: expired entries accumulate between cleanup cycles (1 hour)
- Map grows unbounded in high-traffic scenarios
- OOM risk

**Fix:**
```go
if time.Since(response.CreatedAt) > s.ttl {
    s.mu.RUnlock()
    s.mu.Lock()
    delete(s.store, key)
    s.mu.Unlock()
    return nil, false
}
```

---

### 10. Priority Always Overwritten in Service Layer

**File:** `internal/businessunits/service/business_unit.go:330`

**Problem:**
```go
func (s *businessUnitService) applyDefaultsForNewCreatedBusiness(bu *model.BusinessUnit) {
    if bu.TimeZone == "" {
        bu.TimeZone = locale.InferTimezoneFromPhone(bu.AdminPhone)
    }
    bu.Priority = DefaultPriority  // ⚠️ Unconditionally overwrites user input
}
```

**Impact:**
- Users cannot set custom priority during creation
- Business logic broken for priority-based search
- Always defaults to 10, ignoring any provided value

**Fix:**
```go
if bu.Priority == 0 {
    bu.Priority = DefaultPriority
}
```

---

## Summary

| Severity | Count | Action Required |
|----------|-------|-----------------|
| Critical | 3 | **Cannot deploy without fixing** |
| High | 7 | Fix before production |
| **Total** | **10** | **Immediate attention required** |

## Immediate Action Plan

1. **Fix Issue #1** - Remove MongoDB disconnect in SetMongo
2. **Fix Issue #2** - Add cfg assignment in repository constructor
3. **Fix Issue #3** - Add WhatsApp secret validation on startup
4. **Fix Issue #4** - Implement proper phone number validation
5. **Fix Issue #5** - Resolve rate limiter race condition
6. **Fix Issue #6** - Fix timeout middleware race condition
7. **Fix Issue #7** - Change Priority field to int64
8. **Fix Issue #8** - Add search result limit
9. **Fix Issue #9** - Fix idempotency store cleanup
10. **Fix Issue #10** - Only set default priority if not provided

---

**Note:** 11 additional medium/low priority issues were identified but not included in this top 10 list. See full review report for complete details.
