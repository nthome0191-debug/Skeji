# Integration Tests

This directory contains comprehensive integration tests for the Business Units service.

## Overview

The integration test suite validates the complete HTTP API by making real requests to a running instance of the Business Units service connected to MongoDB.

## Test Coverage

### Endpoints Tested

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

## Running Tests

### Prerequisites

1. **MongoDB must be running** at `mongodb://localhost:27017` (or set `TEST_MONGO_URI`)
2. **Go 1.24+** installed
3. **Kind cluster** (optional, if using Kubernetes MongoDB)

### Quick Start

```bash
# Run all integration tests
./scripts/run-integration-tests.sh
```

This script will:
1. Check MongoDB connectivity
2. Build the application
3. Run migrations on test database
4. Start the application in the background
5. Run all integration tests
6. Clean up (stop app, etc.)

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
go test -v ./test/integration/... -count=1
```

### Running Specific Tests

```bash
# Run only create tests
go test -v ./test/integration -run TestCreate -count=1

# Run only search tests
go test -v ./test/integration -run TestSearch -count=1

# Run a specific test case
go test -v ./test/integration -run TestCreate_ValidBusinessUnit -count=1
```

## Environment Variables

Configure tests using environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `TEST_SERVER_URL` | Base URL of the test server | `http://localhost:8080` |
| `TEST_MONGO_URI` | MongoDB connection string | `mongodb://localhost:27017` |
| `TEST_DB_NAME` | Test database name | `skeji_test` |
| `TEST_SERVER_PORT` | Port for test server | `8080` |

## Test Structure

```
test/integration/
├── README.md                    # This file
├── create_test.go              # POST /api/v1/business-units tests
├── get_test.go                 # GET by ID and List tests
├── search_test.go              # Search endpoint tests
├── update_delete_test.go       # PATCH and DELETE tests
└── testutil/
    ├── client.go               # HTTP client helpers
    ├── mongo.go                # MongoDB test utilities
    ├── fixtures.go             # Test data builders
    └── env.go                  # Environment configuration
```

## Test Utilities

### HTTP Client (`testutil.Client`)

```go
client := testutil.NewClient("http://localhost:8080")

// Make requests
resp := client.POST(t, "/api/v1/business-units", businessUnit)
resp := client.GET(t, "/api/v1/business-units/id/123")
resp := client.PATCH(t, "/api/v1/business-units/id/123", update)
resp := client.DELETE(t, "/api/v1/business-units/id/123")

// Assertions
testutil.AssertStatusCode(t, resp, http.StatusOK)
testutil.AssertContains(t, resp, "expected text")
```

### MongoDB Helper (`testutil.MongoHelper`)

```go
mongo := testutil.NewMongoHelper(t, mongoURI, dbName)
defer mongo.Close(t)

// Clean database
mongo.CleanDatabase(t)

// Clean specific collection
mongo.CleanCollection(t, "Business_units")

// Count documents
count := mongo.CountDocuments(t, "Business_units")
```

### Test Data Builders (`testutil.BusinessUnitBuilder`)

```go
// Use predefined fixtures
bu := testutil.ValidBusinessUnit()
bu := testutil.MinimalBusinessUnit()
bu := testutil.EmptyBusinessUnit()

// Or build custom
bu := testutil.NewBusinessUnitBuilder().
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
    // Setup
    env := testutil.NewTestEnv()
    mongo, client := env.Setup(t)
    defer env.Cleanup(t, mongo)

    // Test logic
    // ...
}
```

### Table-Driven Tests

For testing multiple scenarios:

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
testutil.PrintResponse(t, resp)  // Shows status, body, headers
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
2. **Use unique phone numbers** - Prevents conflicts in multi-admin tests
3. **Test both success and error paths** - Validate errors as well as success
4. **Use descriptive test names** - `TestCreate_InvalidPhoneFormat` not `TestCreate1`
5. **Verify database state** - Don't just check HTTP responses
6. **Test edge cases** - Empty strings, max values, special characters

## Extending Tests

To add new test cases:

1. Create test data in `testutil/fixtures.go` if needed
2. Add test function in appropriate file (create_test.go, etc.)
3. Follow existing patterns for setup/teardown
4. Run tests: `./scripts/run-integration-tests.sh`

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

- Ensure no other process is using port 8080
- Check firewall settings
- Verify MongoDB is accessible

## Future Enhancements

Planned improvements:

- [ ] Add tests for invalid JSON payloads
- [ ] Test concurrent requests
- [ ] Test database constraints (unique indexes, etc.)
- [ ] Add performance/load tests
- [ ] Test authentication/authorization (when implemented)
- [ ] Test WhatsApp webhook signature verification
- [ ] Test idempotency keys
- [ ] Test rate limiting
