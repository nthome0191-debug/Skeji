# Skeji Codebase - Critical Issues Report

**Review Date:** 2025-11-05
**Review Scope:** All production code in `cmd/`, `internal/`, and `pkg/` (excluding test files)
**Methodology:** Manual security and reliability audit

---

## Executive Summary

This report identifies 10 critical and high-severity issues discovered in the Skeji codebase. The issues range from potential goroutine leaks and race conditions to security vulnerabilities and data integrity risks. Immediate attention is recommended for Critical and High severity issues.

**Issue Breakdown:**
- Critical: 4 issues
- High: 4 issues
- Medium: 2 issues

---

## Critical Issues

### 1. Goroutine Leak in Request Timeout Middleware

**Severity:** Critical
**File:** `/Users/nataliaharoni/Projects/Skeji/pkg/middleware/timeout.go`
**Lines:** 69-72

**Description:**
The timeout middleware spawns a goroutine to handle the request but doesn't ensure proper cleanup. If the context times out, the goroutine serving the request continues executing even after the timeout response has been sent to the client. This creates a goroutine leak where requests continue processing in the background.

```go
go func() {
    next.ServeHTTP(tw, r)
    close(done)
}()
```

**Impact:**
- **Resource Exhaustion:** Under load, leaked goroutines accumulate, consuming memory and CPU
- **Database Connection Leaks:** Long-running database operations in leaked goroutines can hold connections
- **Unpredictable Behavior:** Background operations may complete and attempt to modify shared state
- **Production Outage Risk:** Service can become unresponsive under sustained load

**Recommended Fix:**
The goroutine needs to respect the context cancellation. However, this is challenging with the current design because `next.ServeHTTP` doesn't inherently respect context cancellation. Consider:

1. Ensure all downstream operations check `ctx.Done()` channel
2. Add monitoring/logging for timed-out requests that continue processing
3. Implement a request budget or circuit breaker to prevent cascading failures
4. Consider using `context.AfterFunc` (Go 1.21+) for cleanup actions

**Example Fix Pattern:**
```go
// In repository layer, always check context
func (r *mongoBusinessUnitRepository) FindByID(ctx context.Context, id string) (*model.BusinessUnit, error) {
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    // ... rest of implementation
}
```

---

### 2. MongoDB Client Not Properly Closed on Graceful Shutdown

**Severity:** Critical
**File:** `/Users/nataliaharoni/Projects/Skeji/pkg/config/config.go`
**Lines:** 96-101

**Description:**
The `GracefulShutdown` method disconnects the MongoDB client using `context.Background()` without a timeout. If the disconnect operation hangs (network issues, server problems), the graceful shutdown will block indefinitely.

```go
func (cfg *Config) GracefulShutdown() {
    err := cfg.Client.Mongo.Disconnect(context.Background())
    if err != nil {
        cfg.Log.Warn("error occured during attempt to disconnect mongo client: %s", err)
    }
}
```

**Impact:**
- **Hung Shutdown:** Application cannot terminate cleanly during deployment or restart
- **Container/Pod Termination:** In Kubernetes, this leads to SIGKILL after grace period
- **Data Loss Risk:** In-flight operations may not complete properly
- **Orchestration Issues:** Health checks fail, rolling updates get stuck

**Recommended Fix:**
```go
func (cfg *Config) GracefulShutdown() {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    err := cfg.Client.Mongo.Disconnect(ctx)
    if err != nil {
        cfg.Log.Error("Failed to disconnect MongoDB client", "error", err)
    } else {
        cfg.Log.Info("MongoDB client disconnected successfully")
    }
}
```

---

### 3. Missing Context Propagation in Health Check

**Severity:** Critical
**File:** `/Users/nataliaharoni/Projects/Skeji/internal/businessunits/handler/health.go`
**Lines:** 40-44

**Description:**
The ready check creates a new context with timeout instead of deriving from the request context. This means if the server is shutting down, health checks won't be cancelled and will continue attempting to ping MongoDB.

```go
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    if err := h.mongoClient.Ping(ctx, nil); err != nil {
```

**Impact:**
- **Delayed Shutdown:** Health checks continue during graceful shutdown
- **Resource Contention:** Ongoing health checks compete with shutdown procedures
- **False Health Status:** May report healthy when service is actually terminating
- **Load Balancer Confusion:** Traffic may be routed to terminating instances

**Recommended Fix:**
```go
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
    // Derive from request context so it's cancelled during shutdown
    ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
    defer cancel()

    if err := h.mongoClient.Ping(ctx, nil); err != nil {
```

---

### 4. Concurrent Map Access in GetAll Service Method

**Severity:** Critical
**File:** `/Users/nataliaharoni/Projects/Skeji/internal/businessunits/service/business_unit.go`
**Lines:** 121-146

**Description:**
The `GetAll` method uses a `sync.WaitGroup` to parallelize Count and FindAll operations, but the error variables `errCount` and `errFind` are accessed from multiple goroutines without synchronization.

```go
var count int64
var units []*model.BusinessUnit
var errCount, errFind error
var wg sync.WaitGroup
wg.Add(2)
go func() {
    defer wg.Done()
    var err error
    count, err = s.repo.Count(ctx)
    if err != nil {
        s.logger.Error("Failed to count business units", "error", err)
        errCount = apperrors.Internal("Failed to count business units", err)  // Race!
    }
}()

go func() {
    defer wg.Done()
    var err error
    units, err = s.repo.FindAll(ctx, limit, offset)
    if err != nil {
        s.logger.Error("Failed to get all business units", ...)
        errFind = apperrors.Internal("Failed to retrieve business units", err)  // Race!
    }
}()
wg.Wait()

if errCount != nil {  // Potential race reading
    return nil, 0, errCount
}
```

**Impact:**
- **Race Condition:** Go race detector will flag this as a data race
- **Undefined Behavior:** Reads and writes without synchronization lead to unpredictable results
- **Possible Panic:** In extreme cases, concurrent map access can panic
- **Silent Failures:** Errors might not be properly returned to caller

**Recommended Fix:**
Use channels or a mutex to synchronize error handling:

```go
type result struct {
    units []*model.BusinessUnit
    count int64
    err   error
}

func (s *businessUnitService) GetAll(ctx context.Context, limit int, offset int) ([]*model.BusinessUnit, int64, error) {
    if limit <= 0 {
        limit = 10
    }
    if limit > 100 {
        limit = 100
    }
    if offset < 0 {
        offset = 0
    }

    countCh := make(chan result, 1)
    findCh := make(chan result, 1)

    go func() {
        count, err := s.repo.Count(ctx)
        if err != nil {
            s.logger.Error("Failed to count business units", "error", err)
            countCh <- result{err: apperrors.Internal("Failed to count business units", err)}
            return
        }
        countCh <- result{count: count}
    }()

    go func() {
        units, err := s.repo.FindAll(ctx, limit, offset)
        if err != nil {
            s.logger.Error("Failed to get all business units", "limit", limit, "offset", offset, "error", err)
            findCh <- result{err: apperrors.Internal("Failed to retrieve business units", err)}
            return
        }
        findCh <- result{units: units}
    }()

    countRes := <-countCh
    findRes := <-findCh

    if countRes.err != nil {
        return nil, 0, countRes.err
    }
    if findRes.err != nil {
        return nil, 0, findRes.err
    }

    return findRes.units, countRes.count, nil
}
```

---

## High Severity Issues

### 5. Missing MongoDB Connection Cleanup in Migration Tool

**Severity:** High
**File:** `/Users/nataliaharoni/Projects/Skeji/cmd/migrate/main.go`
**Lines:** 35

**Description:**
The migration tool uses `defer client.Disconnect(ctx)` with the same context that has a 30-second timeout. If the migration takes 29 seconds, the disconnect will only have 1 second to complete, which may not be enough.

```go
func migrateMongo(ctx context.Context) {
    mongoURI := os.Getenv("MONGO_URI")
    if mongoURI == "" {
        log.Fatal("MONGO_URI environment variable is required")
    }

    client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
    if err != nil {
        log.Fatalf("Failed to connect to MongoDB: %v", err)
    }
    defer client.Disconnect(ctx)  // This ctx will expire!
```

**Impact:**
- **Connection Leaks:** MongoDB connections may not close properly
- **Migration Failures:** Longer migrations fail prematurely
- **Resource Exhaustion:** Leaked connections accumulate in connection pool
- **Deployment Issues:** Migration jobs fail to complete in CI/CD pipelines

**Recommended Fix:**
```go
func migrateMongo(ctx context.Context) {
    mongoURI := os.Getenv("MONGO_URI")
    if mongoURI == "" {
        log.Fatal("MONGO_URI environment variable is required")
    }

    client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
    if err != nil {
        log.Fatalf("Failed to connect to MongoDB: %v", err)
    }
    defer func() {
        // Use a fresh context for disconnect with reasonable timeout
        disconnectCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        if err := client.Disconnect(disconnectCtx); err != nil {
            log.Printf("Warning: Failed to disconnect MongoDB client: %v", err)
        }
    }()

    fmt.Printf("Connected to MongoDB: %s\n", mongoURI)

    if err := mongoMigration.RunMigration(ctx, client); err != nil {
        log.Fatalf("Migration failed: %v", err)
    }
}
```

---

### 6. Potential Double Write in Timeout Middleware

**Severity:** High
**File:** `/Users/nataliaharoni/Projects/Skeji/pkg/middleware/timeout.go`
**Lines:** 74-87

**Description:**
When a timeout occurs, the middleware writes a timeout response (lines 81-84), but the original goroutine may also attempt to write a response. While there's a `written` flag check, the sequence of operations isn't atomic and could lead to multiple writes.

```go
select {
case <-done:
    return
case <-ctx.Done():
    tw.timeout()
    tw.mu.Lock()
    if !tw.written {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusServiceUnavailable)
        _, _ = w.Write([]byte(`{"error":"Request timeout"}`))
        tw.written = true
    }
    tw.mu.Unlock()
}
```

**Impact:**
- **HTTP Protocol Violation:** Multiple WriteHeader calls cause errors
- **Corrupted Responses:** Client receives malformed HTTP responses
- **Panic Risk:** ResponseWriter panics on duplicate WriteHeader in some implementations
- **Client Confusion:** Partial or duplicate response bodies

**Recommended Fix:**
The current locking is good, but ensure the `timeoutWriter` properly prevents the inner goroutine from writing after timeout. The issue is that the inner goroutine writes to `tw.ResponseWriter` directly in some cases.

```go
func (tw *timeoutWriter) Write(b []byte) (int, error) {
    tw.mu.Lock()
    defer tw.mu.Unlock()

    if tw.timedOut {
        // Silently discard writes after timeout
        return len(b), nil  // Pretend success to prevent errors
    }

    if !tw.written {
        tw.statusCode = http.StatusOK
        tw.written = true
    }

    return tw.ResponseWriter.Write(b)
}
```

---

### 7. No Validation of MongoDB Collection Validator Updates

**Severity:** High
**File:** `/Users/nataliaharoni/Projects/Skeji/internal/migrations/mongo/migrate.go`
**Lines:** 106-120

**Description:**
When updating collection validators, the code only logs a warning if the update fails but continues execution. If a validator update fails, the collection will have an outdated schema, leading to potential data integrity issues.

```go
if err := db.RunCommand(collCtx, command).Err(); err != nil {
    fmt.Printf("⚠️  Warning: failed updating validator for %s: %v\n", name, err)
}
```

**Impact:**
- **Schema Drift:** Collections accept invalid data that shouldn't pass validation
- **Data Corruption:** Invalid documents get inserted without detection
- **Silent Failures:** Migrations appear successful but schema is wrong
- **Production Incidents:** Invalid data causes application errors downstream

**Recommended Fix:**
```go
if err := db.RunCommand(collCtx, command).Err(); err != nil {
    // Validator update failures should be fatal in most cases
    return fmt.Errorf("CRITICAL: failed to update validator for %s: %w", name, err)
}
fmt.Printf("✅ Successfully updated validator for %s\n", name)
```

**Alternative Approach (for non-breaking changes):**
```go
if err := db.RunCommand(collCtx, command).Err(); err != nil {
    // Only warn if it's a backwards-compatible change
    if isCriticalValidatorChange(name, validator) {
        return fmt.Errorf("CRITICAL: failed to update validator for %s: %w", name, err)
    }
    fmt.Printf("⚠️  Warning: failed updating non-critical validator for %s: %v\n", name, err)
}
```

---

### 8. WhatsApp Secret Can Be Empty in Development

**Severity:** High
**File:** `/Users/nataliaharoni/Projects/Skeji/pkg/app/application.go`
**Lines:** 64-69

**Description:**
The code calls `log.Fatal` if `WHATSAPP_APP_SECRET` is empty, but this happens AFTER the middleware chain is partially built. This means there's a brief window where the app could process requests without signature verification.

```go
if cfg.WhatsAppAppSecret != "" {
    appHttpHandler = middleware.WhatsAppSignatureVerification(cfg.WhatsAppAppSecret, cfg.Log)(appHttpHandler)
    cfg.Log.Info("WhatsApp signature verification enabled")
} else {
    cfg.Log.Fatal("WHATSAPP_APP_SECRET environment variable is required")
}
```

**Impact:**
- **Security Bypass:** Brief window where requests aren't validated
- **Logic Error:** Fatal after partial initialization is confusing
- **Testing Issues:** Makes it harder to run the service in test environments
- **Defense in Depth Violation:** Security check happens too late in initialization

**Recommended Fix:**
Move the validation to the config loading phase:

```go
// In pkg/config/config.go
func Load(serviceName string) *Config {
    whatsappSecret := getEnvStr(EnvWhatsAppAppSecret, "")
    if whatsappSecret == "" {
        log.Fatal("WHATSAPP_APP_SECRET environment variable is required")
    }

    cfg := &Config{
        // ... other config
        WhatsAppAppSecret: whatsappSecret,
        // ...
    }
    // ... rest of initialization
}

// In pkg/app/application.go - now safe to assume it's set
func (a *Application) setAppHandler(cfg *config.Config, appHandler contracts.Handler) {
    // ...
    appHttpHandler = middleware.WhatsAppSignatureVerification(cfg.WhatsAppAppSecret, cfg.Log)(appHttpHandler)
    // ...
}
```

---

### 9. MongoDB Index Creation Without Error Handling

**Severity:** High
**File:** `/Users/nataliaharoni/Projects/Skeji/internal/migrations/mongo/migrate.go`
**Lines:** 130-137

**Description:**
The `ensureIndexes` function creates indexes but doesn't handle partial failures. If some indexes are created successfully and others fail, the migration continues with an incomplete index set.

```go
func ensureIndexes(ctx context.Context, db *mongo.Database, name string, models []mongo.IndexModel) error {
    coll := db.Collection(name)
    if len(models) == 0 {
        return nil
    }

    _, err := coll.Indexes().CreateMany(ctx, models)
    if err != nil {
        return fmt.Errorf("failed creating indexes for %s: %w", name, err)
    }

    fmt.Printf("📚 Ensured indexes for %s\n", name)
    return nil
}
```

**Impact:**
- **Performance Degradation:** Missing indexes cause slow queries
- **Incorrect Query Results:** Some queries may fail or return wrong results
- **Inconsistent State:** Different environments may have different indexes
- **Difficult Debugging:** Index issues are hard to detect and diagnose

**Recommended Fix:**
Add better error context and validation:

```go
func ensureIndexes(ctx context.Context, db *mongo.Database, name string, models []mongo.IndexModel) error {
    coll := db.Collection(name)
    if len(models) == 0 {
        fmt.Printf("ℹ️  No indexes to create for %s\n", name)
        return nil
    }

    fmt.Printf("📚 Creating %d index(es) for %s...\n", len(models), name)

    createdIndexes, err := coll.Indexes().CreateMany(ctx, models)
    if err != nil {
        return fmt.Errorf("failed creating indexes for %s: %w (created %d of %d)", name, err, len(createdIndexes), len(models))
    }

    if len(createdIndexes) != len(models) {
        return fmt.Errorf("index creation mismatch for %s: expected %d, created %d", name, len(models), len(createdIndexes))
    }

    fmt.Printf("✅ Successfully created %d index(es) for %s\n", len(createdIndexes), name)
    return nil
}
```

Also, consider listing existing indexes before creation to provide better diagnostics:

```go
func ensureIndexes(ctx context.Context, db *mongo.Database, name string, models []mongo.IndexModel) error {
    coll := db.Collection(name)
    if len(models) == 0 {
        fmt.Printf("ℹ️  No indexes to create for %s\n", name)
        return nil
    }

    // List existing indexes for diagnostics
    cursor, err := coll.Indexes().List(ctx)
    if err != nil {
        fmt.Printf("⚠️  Warning: Could not list existing indexes for %s: %v\n", name, err)
    } else {
        var existingIndexes []bson.M
        if err := cursor.All(ctx, &existingIndexes); err == nil {
            fmt.Printf("ℹ️  Collection %s currently has %d index(es)\n", name, len(existingIndexes))
        }
    }

    fmt.Printf("📚 Creating/updating %d index(es) for %s...\n", len(models), name)

    createdIndexes, err := coll.Indexes().CreateMany(ctx, models)
    if err != nil {
        return fmt.Errorf("failed creating indexes for %s: %w", name, err)
    }

    fmt.Printf("✅ Successfully ensured %d index(es) for %s\n", len(createdIndexes), name)
    return nil
}
```

---

## Medium Severity Issues

### 10. Unbounded Slice Allocation in Rate Limiter

**Severity:** Medium
**File:** `/Users/nataliaharoni/Projects/Skeji/pkg/middleware/rate_limit.go`
**Lines:** 73-78

**Description:**
The `Allow` method creates a new slice for valid timestamps on every request. If an attacker makes many requests just under the rate limit threshold, this creates significant garbage collection pressure.

```go
validTimestamps := make([]time.Time, 0)
for _, ts := range timestamps {
    if now.Sub(ts) < rl.window {
        validTimestamps = append(validTimestamps, ts)
    }
}
```

**Impact:**
- **Memory Pressure:** High allocation rate under load
- **GC Overhead:** Frequent garbage collection pauses
- **Performance Degradation:** Slower response times during attacks
- **Resource Exhaustion:** Combined with other issues, can cause service degradation

**Recommended Fix:**
Reuse the existing slice and modify in-place:

```go
func (rl *PhoneRateLimiter) Allow(phone string) bool {
    if phone == "" {
        return true
    }

    now := time.Now()

    rl.mu.Lock()
    defer rl.mu.Unlock()

    timestamps := rl.requests[phone]

    // Filter in-place to reduce allocations
    validCount := 0
    for i := 0; i < len(timestamps); i++ {
        if now.Sub(timestamps[i]) < rl.window {
            timestamps[validCount] = timestamps[i]
            validCount++
        }
    }
    timestamps = timestamps[:validCount]

    if len(timestamps) >= rl.limit {
        // Update map even on rejection to keep state clean
        rl.requests[phone] = timestamps
        return false
    }

    timestamps = append(timestamps, now)
    rl.requests[phone] = timestamps

    return true
}
```

Note: This fix also corrects a subtle bug where the original code holds both RLock and Lock, which could cause deadlocks.

---

### 11. Potential Logger Format String Vulnerability

**Severity:** Medium
**File:** `/Users/nataliaharoni/Projects/Skeji/pkg/config/config.go`
**Lines:** 99

**Description:**
The log message uses `%s` format specifier in a non-formatting context, which could be confusing and potentially lead to issues if the error message contains format specifiers.

```go
cfg.Log.Warn("error occured during attempt to disconnect mongo client: %s", err)
```

**Impact:**
- **Incorrect Logging:** Error message not properly formatted
- **Information Disclosure:** In rare cases, could expose internals
- **Debugging Confusion:** Makes log analysis harder
- **Typo:** "occured" should be "occurred"

**Recommended Fix:**
```go
cfg.Log.Warn("Error occurred during MongoDB client disconnect", "error", err)
```

This follows structured logging best practices and is consistent with the rest of the codebase.

---

## Summary of Recommendations

### Immediate Actions (Critical/High)
1. Fix the goroutine leak in timeout middleware
2. Add proper context timeout to graceful shutdown
3. Fix race condition in GetAll service method
4. Ensure MongoDB disconnect uses proper context
5. Move WhatsApp secret validation to config loading
6. Make validator update failures fatal in migrations

### Short-term Actions (Medium)
7. Optimize rate limiter to reduce allocations
8. Fix logging inconsistencies
9. Add comprehensive monitoring for:
   - Goroutine counts
   - MongoDB connection pool stats
   - Request timeout rates
   - Rate limit rejections

### Long-term Improvements
10. Implement circuit breakers for external dependencies
11. Add distributed tracing (OpenTelemetry)
12. Implement comprehensive integration tests
13. Add chaos engineering tests for timeout scenarios
14. Consider implementing transaction support in repositories
15. Add metrics for migration execution time and success rates

---

## Testing Recommendations

To validate fixes and prevent regressions:

1. **Race Detection:** Run all tests with `-race` flag
   ```bash
   go test -race ./...
   ```

2. **Load Testing:** Test timeout behavior under load
   ```bash
   # Use tools like k6 or vegeta to simulate high load
   ```

3. **Chaos Testing:** Simulate network failures, slow dependencies
   ```bash
   # Test behavior when MongoDB is slow or unavailable
   ```

4. **Migration Testing:** Test migration rollback and re-run scenarios
   ```bash
   # Ensure migrations are truly idempotent
   ```

---

## Additional Observations

### Code Quality Strengths
- Good separation of concerns (handler/service/repository layers)
- Comprehensive middleware stack
- Proper use of structured logging
- Context propagation in most places
- Input sanitization functions

### Areas for Future Enhancement
- Add OpenAPI/Swagger documentation
- Implement distributed tracing
- Add more comprehensive error types
- Consider implementing the repository transaction functionality
- Add health check for all dependencies (not just MongoDB)
- Implement feature flags for gradual rollouts

---

**Report Generated By:** Claude Code
**Review Methodology:** Static analysis, security audit, and architectural review
**Next Review:** Recommended within 1 month after fixes are implemented
