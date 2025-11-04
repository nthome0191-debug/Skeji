# Integration Tests

This directory contains comprehensive integration tests for all Skeji microservices.

NOTE: INTEGRATION TESTS NOT READY< NOT VALIDATED AND IT SEEMS LIKE CURRENTLY A PRIORITY CAN BE SET BY USER WHICH COULD NOT AND SHOULD NOT BE ALLOWED, TODO: FIX

## Directory Structure

```
test/integration/
├── common/                  # Shared test utilities (reusable across services)
│   ├── client.go           # HTTP client helpers
│   ├── mongo.go            # MongoDB test utilities
│   └── env.go              # Environment configuration
├── businessunits/          # Business Units service tests
│   ├── fixtures.go         # Test data builders
│   ├── create_test.go
│   ├── get_test.go
│   ├── search_test.go
│   └── update_delete_test.go
├── booking/                # Booking service tests (future)
├── schedule/               # Schedule service tests (future)
└── README.md               # This file
```

## Test Organization

Each microservice has its own test package:
- **common/** - Generic utilities shared by all services (MongoDB, HTTP client, env setup)
- **businessunits/** - Tests specific to the Business Units service
- **booking/** (future) - Tests for the Booking service
- **schedule/** (future) - Tests for the Schedule service

This structure ensures:
- Clear separation of concerns per service
- Reusable infrastructure code in `common/`
- Easy addition of new service test suites

## Business Units Service Tests

### Test Coverage

1. **POST /api/v1/business-units** - Create business unit
   - Valid input
   - Minimal required fields
   - Empty/missing fields
   - Invalid phone formats
   - Unsupported countries
   - Multiple businesses per admin

2. **GET /api/v1/business-units/id/:id** - Get by ID
   - Existing business unit
   - Non-existent ID
   - Invalid ID format

3. **GET /api/v1/business-units** - List all
   - Empty database
   - Multiple items with sorting
   - Pagination (limit, offset)
   - Default/max limits

4. **GET /api/v1/business-units/search** - Search
   - Valid cities and labels
   - Multiple cities/labels
   - Missing parameters
   - Empty parameters
   - Priority sorting
   - Case insensitive
   - Comma-delimited values

5. **PATCH /api/v1/business-units/id/:id** - Update
   - Full update
   - Partial update
   - Empty update
   - Non-existent ID
   - Invalid phone format

6. **DELETE /api/v1/business-units/id/:id** - Delete
   - Existing business unit
   - Non-existent ID
   - Invalid ID
   - Double delete

**Total: 36+ test cases**

## Running Tests

### Prerequisites

1. **MongoDB must be running** at `mongodb://localhost:27017` (or set `TEST_MONGO_URI`)
2. **Go 1.24+** installed
3. **Kind cluster** (optional, if using Kubernetes MongoDB)

### Quick Start

```bash
# Run all integration tests (all services)
./scripts/run-integration-tests.sh

# Or use Make
make test-integration
```

This script will:
1. Check MongoDB connectivity
2. Build the application
3. Run migrations on test database
4. Start the application in background
5. Run all integration tests
6. Clean up (stop app, etc.)

### Running Specific Service Tests

```bash
# Run only Business Units tests
go test -v ./test/integration/businessunits/... -count=1

# Run only create tests
go test -v ./test/integration/businessunits -run TestCreate -count=1

# Run a specific test case
go test -v ./test/integration/businessunits -run TestCreate_ValidBusinessUnit -count=1
```

### Manual Test Execution

If you prefer to manage the application lifecycle manually:

```bash
# 1. Start MongoDB (if not already running)
make mongo-up
kubectl port-forward -n mongo svc/mongo 27017:27017

# 2. In another terminal, build and start the application
go build -o bin/business-units ./cmd/business-units
PORT=8080 MONGO_URI="mongodb://localhost:27017" MONGO_DATABASE_NAME="skeji_test" ./bin/business-units

# 3. In another terminal, run the tests
TEST_SERVER_URL="http://localhost:8080" \
TEST_MONGO_URI="mongodb://localhost:27017" \
TEST_DB_NAME="skeji_test" \
go test -v ./test/integration/businessunits/... -count=1
```

## Environment Variables

Configure tests using environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `TEST_SERVER_URL` | Base URL of the test server | `http://localhost:8080` |
| `TEST_MONGO_URI` | MongoDB connection string | `mongodb://localhost:27017` |
| `TEST_DB_NAME` | Test database name | `skeji_test` |
| `TEST_SERVER_PORT` | Port for test server | `8080` |

## Common Test Utilities

### HTTP Client (`common.Client`)

```go
import "skeji/test/integration/common"

client := common.NewClient("http://localhost:8080")

resp := common.POST(t, "/api/v1/business-units", businessUnit)
resp := common.GET(t, "/api/v1/business-units/id/123")
resp := common.PATCH(t, "/api/v1/business-units/id/123", update)
resp := common.DELETE(t, "/api/v1/business-units/id/123")

common.AssertStatusCode(t, resp, http.StatusOK)
common.AssertContains(t, resp, "expected text")
```

### MongoDB Helper (`common.MongoHelper`)

```go
import "skeji/test/integration/common"

mongo := common.NewMongoHelper(t, mongoURI, dbName)
defer mongo.Close(t)

mongo.CleanDatabase(t)
mongo.CleanCollection(t, "Business_units")
count := mongo.CountDocuments(t, "Business_units")
```

### Test Environment (`common.TestEnv`)

```go
import "skeji/test/integration/common"

env := common.NewTestEnv()
mongo, client := env.Setup(t)
defer env.Cleanup(t, mongo)
```

## Service-Specific Fixtures

Each service package has its own `fixtures.go` with test data builders.

### Business Units Fixtures

```go
// In businessunits package - no import needed

bu := ValidBusinessUnit()
bu := MinimalBusinessUnit()
bu := EmptyBusinessUnit()

bu := NewBusinessUnitBuilder().
    WithName("My Business").
    WithCities("Tel Aviv").
    WithLabels("cafe").
    WithAdminPhone("+972501234567").
    Build()
```

## Test Patterns

### Setup/Teardown

Each test follows this pattern:

```go
func TestSomething(t *testing.T) {
    env := common.NewTestEnv()
    mongo, client := env.Setup(t)
    defer env.Cleanup(t, mongo)

    // Test logic
}
```

### Table-Driven Tests

```go
testCases := []struct {
    name string
    input string
    want int
}{
    {name: "valid", input: "data", want: 200},
    {name: "invalid", input: "bad", want: 400},
}

for _, tc := range testCases {
    t.Run(tc.name, func(t *testing.T) {
        // Test logic
    })
}
```

## Adding Tests for a New Service

To add integration tests for a new service:

1. **Create service directory**
   ```bash
   mkdir test/integration/myservice
   ```

2. **Create fixtures**
   ```go
   // test/integration/myservice/fixtures.go
   package myservice

   import "skeji/pkg/model"

   func ValidMyEntity() model.MyEntity {
       return model.MyEntity{
           Field: "value",
       }
   }
   ```

3. **Create test files**
   ```go
   // test/integration/myservice/create_test.go
   package myservice

   import (
       "testing"
       "skeji/test/integration/common"
   )

   func TestCreate_ValidEntity(t *testing.T) {
       env := common.NewTestEnv()
       mongo, client := env.Setup(t)
       defer env.Cleanup(t, mongo)

       entity := ValidMyEntity()
       resp := client.POST(t, "/api/v1/myservice", entity)
       common.AssertStatusCode(t, resp, http.StatusCreated)
   }
   ```

4. **Run tests**
   ```bash
   go test -v ./test/integration/myservice/... -count=1
   ```

## Debugging

### View Application Logs

```bash
# If using the script
tail -f /tmp/business-units-test.log

# Or check after test failure
cat /tmp/business-units-test.log
```

### Enable Verbose Logging

```bash
# Set log level when running manually
LOG_LEVEL=debug ./bin/business-units
```

### Print Response Details

```go
resp := client.GET(t, "/api/v1/business-units")
common.PrintResponse(t, resp)
```

## CI/CD Integration

The test script returns proper exit codes:
- `0` - All tests passed
- `1` - Some tests failed or setup error

Example GitHub Actions:

```yaml
- name: Run Integration Tests
  run: |
    make mongo-up
    kubectl port-forward -n mongo svc/mongo 27017:27017 &
    sleep 5
    ./scripts/run-integration-tests.sh
```

## Best Practices

1. **Always clean database between tests** - Use `env.Setup()` and `env.Cleanup()`
2. **Use unique identifiers** - Prevents conflicts in concurrent tests
3. **Test both success and error paths** - Validate errors as well as success
4. **Use descriptive test names** - `TestCreate_InvalidPhoneFormat` not `TestCreate1`
5. **Verify database state** - Don't just check HTTP responses
6. **Test edge cases** - Empty strings, max values, special characters
7. **Keep fixtures in service packages** - Don't pollute `common/` with service-specific data
8. **Reuse common utilities** - HTTP client, MongoDB helpers, environment setup

## Troubleshooting

### "Cannot connect to MongoDB"

```bash
# Check if MongoDB is running
mongosh --eval "db.adminCommand('ping')"

# Or start it
make mongo-up
kubectl port-forward -n mongo svc/mongo 27017:27017
```

### "Application failed to start"

```bash
# Check application logs
cat /tmp/business-units-test.log

# Manually test build
go build ./cmd/business-units
./business-units
```

### "Tests hang or timeout"

- Ensure no other process is using the test port
- Check firewall settings
- Verify MongoDB is accessible
- Check service-specific logs

## Future Enhancements

Planned improvements:

- [ ] Add tests for Booking service
- [ ] Add tests for Schedule service
- [ ] Add tests for Search service
- [ ] Test invalid JSON payloads
- [ ] Test concurrent requests
- [ ] Add performance/load tests
- [ ] Test authentication/authorization
- [ ] Test WhatsApp webhook signature verification
- [ ] Test idempotency keys
- [ ] Test rate limiting
