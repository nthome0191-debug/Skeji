#!/bin/bash

# Integration Test Runner Script
# This script sets up the test environment, starts the application, and runs integration tests

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
TEST_SERVER_PORT=${TEST_SERVER_PORT:-8080}
TEST_MONGO_URI=${TEST_MONGO_URI:-"mongodb://localhost:27017"}
TEST_DB_NAME=${TEST_DB_NAME:-"skeji_test"}
APP_BINARY="./bin/business-units"
APP_PID_FILE="/tmp/business-units-test.pid"

# Cleanup function
cleanup() {
    echo -e "${YELLOW}Cleaning up...${NC}"

    # Stop the application if running
    if [ -f "$APP_PID_FILE" ]; then
        APP_PID=$(cat "$APP_PID_FILE")
        if kill -0 "$APP_PID" 2>/dev/null; then
            echo "Stopping application (PID: $APP_PID)..."
            kill "$APP_PID"
            wait "$APP_PID" 2>/dev/null || true
        fi
        rm -f "$APP_PID_FILE"
    fi

    echo -e "${GREEN}Cleanup complete${NC}"
}

# Register cleanup function
trap cleanup EXIT INT TERM

# Check if MongoDB is running
echo -e "${YELLOW}Checking MongoDB connection...${NC}"
if ! mongosh "$TEST_MONGO_URI" --eval "db.adminCommand('ping')" >/dev/null 2>&1; then
    echo -e "${RED}Error: Cannot connect to MongoDB at $TEST_MONGO_URI${NC}"
    echo "Please ensure MongoDB is running:"
    echo "  - Local: mongod or brew services start mongodb-community"
    echo "  - Kind: make mongo-up && kubectl port-forward -n mongo svc/mongo 27017:27017"
    exit 1
fi
echo -e "${GREEN}MongoDB is accessible${NC}"

# Build the application
echo -e "${YELLOW}Building application...${NC}"
go build -o "$APP_BINARY" ./cmd/business-units
echo -e "${GREEN}Build complete${NC}"

# Run migrations
echo -e "${YELLOW}Running migrations on test database...${NC}"
MONGO_URI="$TEST_MONGO_URI" MONGO_DATABASE_NAME="$TEST_DB_NAME" go run ./cmd/migrate/main.go
echo -e "${GREEN}Migrations complete${NC}"

# Start the application in the background
echo -e "${YELLOW}Starting application on port $TEST_SERVER_PORT...${NC}"
PORT="$TEST_SERVER_PORT" \
MONGO_URI="$TEST_MONGO_URI" \
MONGO_DATABASE_NAME="$TEST_DB_NAME" \
LOG_LEVEL="info" \
"$APP_BINARY" > /tmp/business-units-test.log 2>&1 &

APP_PID=$!
echo $APP_PID > "$APP_PID_FILE"
echo "Application started with PID: $APP_PID"

# Wait for the application to be ready
echo -e "${YELLOW}Waiting for application to be ready...${NC}"
MAX_WAIT=30
WAIT_COUNT=0
while [ $WAIT_COUNT -lt $MAX_WAIT ]; do
    if curl -s "http://localhost:$TEST_SERVER_PORT/health" > /dev/null 2>&1; then
        echo -e "${GREEN}Application is ready!${NC}"
        break
    fi
    WAIT_COUNT=$((WAIT_COUNT + 1))
    if [ $WAIT_COUNT -eq $MAX_WAIT ]; then
        echo -e "${RED}Application failed to start within $MAX_WAIT seconds${NC}"
        echo "Application logs:"
        cat /tmp/business-units-test.log
        exit 1
    fi
    sleep 1
done

# Run the integration tests
echo -e "${YELLOW}Running integration tests...${NC}"
TEST_SERVER_URL="http://localhost:$TEST_SERVER_PORT" \
TEST_MONGO_URI="$TEST_MONGO_URI" \
TEST_DB_NAME="$TEST_DB_NAME" \
TEST_SERVER_PORT="$TEST_SERVER_PORT" \
go test -v ./test/integration/... -count=1

TEST_EXIT_CODE=$?

# Show results
echo ""
if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}✅ All tests passed!${NC}"
else
    echo -e "${RED}❌ Some tests failed${NC}"
    echo "Application logs:"
    tail -n 50 /tmp/business-units-test.log
fi

exit $TEST_EXIT_CODE
