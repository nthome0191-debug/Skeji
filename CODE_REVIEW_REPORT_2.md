


### 4. 🟠 HIGH: Missing Nil Check on Slice Pointer

**Location**: `internal/businessunits/service/business_unit.go:379`

**Issue**:
```go
if updates.Maintainers != nil {
    normalized := sanitizer.NormalizeMaintainers(*updates.Maintainers)
    // What if normalized is empty? No check!
    updates.Maintainers = &normalized
}
```

**Problem**:
- Dereferences `updates.Maintainers` without checking if the slice is nil after dereferencing
- If `NormalizeMaintainers` returns empty slice, still assigns it
- But more importantly: in schedules service (line 395), same pattern exists with `Exceptions` field

**Actually checking schedules/service**:
At `internal/schedules/service/schedule.go:391-395`:
```go
if updates.Exceptions != nil {
    // BUG: What if *updates.Exceptions itself contains invalid dates?
    // No validation before merging!
    merged["exceptions"] = *updates.Exceptions
}
```

**Impact**:
- Potential nil pointer panic
- Invalid data merged without validation

**Priority**: 🟠 **HIGH** - Potential nil pointer dereference

---

### 5. 🟡 MEDIUM: ReDoS Vulnerability - Unescaped Regex Input

**Location**: `internal/schedules/repository/schedule.go:197`

**Issue**:
```go
if city != "" {
    filter["city"] = bson.M{"$regex": city, "$options": "i"}  // BUG: city not escaped
}
```

**Problem**:
- User-supplied `city` parameter used directly in regex without escaping
- Attacker can supply regex patterns like `"(a+)+b"` or `"a.*b"`
- Regular Expression Denial of Service (ReDoS) vulnerability
- Complex patterns can cause exponential backtracking

**Example Attack**:
```
GET /api/v1/schedules/search?business_id=xxx&city=(a+)+b
```

**Impact**:
- CPU exhaustion
- Denial of service
- Performance degradation

**Priority**: 🟡 **MEDIUM** - Security vulnerability (ReDoS)

---

### 6. 🟡 MEDIUM: Silently Ignored Input Validation Errors

**Locations**:
- `internal/businessunits/handler/business_unit.go:70-71`
- `internal/schedules/handler/schedule.go:78-79`

**Issue**:
```go
limit, _ := strconv.Atoi(query.Get("limit"))    // Error ignored
offset, _ := strconv.Atoi(query.Get("offset"))  // Error ignored
```

**Problem**:
- Errors from `strconv.Atoi()` are intentionally ignored with `_`
- Invalid input like `?limit=abc` silently treated as `0`
- No HTTP 400 Bad Request returned to caller
- Inconsistent API behavior - invalid input should be rejected

**Example**:
```
GET /api/v1/business-units?limit=abc&offset=xyz
```
Should return HTTP 400, but instead treats as `limit=0&offset=0`

**Impact**:
- Poor API design
- Confusing behavior for clients
- Invalid input accepted silently

**Priority**: 🟡 **MEDIUM** - API quality issue

---

### 7. 🟡 MEDIUM: Context Coordination Issues in Concurrent Goroutines

**Locations**:
- `internal/businessunits/service/business_unit.go:148, 160`
- `internal/schedules/service/schedule.go:140, 152`

**Issue**:
```go
go func() {
    defer wg.Done()
    ctx, cancel := context.WithTimeout(ctx, s.cfg.ReadTimeout)  // Independent timeout
    defer cancel()
    count, err = s.repo.Count(ctx)
    // ...
}()

go func() {
    defer wg.Done()
    ctx, cancel := context.WithTimeout(ctx, s.cfg.ReadTimeout)  // Another independent timeout
    defer cancel()
    units, err = s.repo.FindAll(ctx, limit, offset)
    // ...
}()
```

**Problem**:
- Each goroutine creates its own independent timeout context
- If one times out, the other continues running
- Parent function waits for BOTH to complete, even if one timed out
- No coordination between goroutines' timeouts

**Impact**:
- Function can take up to 2x ReadTimeout to complete
- One slow query doesn't cancel the other
- Inefficient timeout handling

**Priority**: 🟡 **MEDIUM** - Performance issue

---


**Priority**: 🟡 **MEDIUM** - Configuration validation gap

---

### 9. 🟢 LOW: Incomplete Error Context in Service Layer

**Multiple locations**: Throughout service layer

**Issue**:
Service methods log errors but don't include important context like user ID, request ID, etc.

**Example**:
```go
s.cfg.Log.Error("Failed to get all business units",
    "limit", limit,
    "offset", offset,
    "error", err,
)
// Missing: request_id, user_id, trace_id for debugging
```

**Problem**:
- Hard to trace errors in production logs
- Missing correlation IDs for distributed tracing
- Incomplete observability

**Impact**:
- Debugging difficulty
- Poor observability

**Priority**: 🟢 **LOW** - Observability issue

---
