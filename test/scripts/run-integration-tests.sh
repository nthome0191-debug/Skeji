#!/bin/bash

set -e

set_colors() {
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    NC='\033[0m'
}

set_variables() {
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
    TEST_SERVER_PORT=${TEST_SERVER_PORT:-8080}
    TEST_MONGO_URI=${TEST_MONGO_URI:-"mongodb://localhost:27017/?directConnection=true"}
    TEST_DB_NAME=${TEST_DB_NAME:-"skeji_test"}
    APP_BINARY="$PROJECT_ROOT/bin/business-units"
    APP_PID_FILE="/tmp/business-units-test.pid"
    PORT_FORWARD_PID_FILE="/tmp/mongo-port-forward.pid"
    CLUSTER_NAME="skeji-local"
    CREATED_CLUSTER=false
    STARTED_PORT_FORWARD=false
    STARTED_APP=false
}

load_env_file() {
    if [ -f "$PROJECT_ROOT/.env.test" ]; then
        echo -e "${YELLOW}Loading environment variables from .env.test${NC}"
        set -a
        source "$PROJECT_ROOT/.env.test"
        set +a
        echo -e "${GREEN}Environment variables loaded${NC}"
    else
        echo -e "${YELLOW}No .env.test file found, using defaults${NC}"
    fi
}

cleanup() {
    echo ""
    echo -e "${YELLOW}Cleaning up...${NC}"

    if [ -f "$APP_PID_FILE" ]; then
        APP_PID=$(cat "$APP_PID_FILE")
        if ps -p "$APP_PID" > /dev/null 2>&1; then
            echo "Stopping application (PID: $APP_PID)..."
            kill "$APP_PID" 2>/dev/null || true
            wait "$APP_PID" 2>/dev/null || true
        fi
        rm -f "$APP_PID_FILE"
    fi

    if [ -f "$PORT_FORWARD_PID_FILE" ]; then
        PF_PID=$(cat "$PORT_FORWARD_PID_FILE")
        if ps -p "$PF_PID" > /dev/null 2>&1; then
            echo "Stopping MongoDB port forwarding (PID: $PF_PID)..."
            kill "$PF_PID" 2>/dev/null || true
        fi
        rm -f "$PORT_FORWARD_PID_FILE"
    fi

    echo -e "${GREEN}Cleanup complete${NC}"
}

check_kind_cluster() {
    echo -e "${BLUE}=== Checking Kind cluster ===${NC}"
    if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
        echo -e "${GREEN}Kind cluster '$CLUSTER_NAME' already exists${NC}"
        return 0
    else
        echo -e "${YELLOW}Kind cluster '$CLUSTER_NAME' not found${NC}"
        return 1
    fi
}

setup_kind() {
    echo -e "${BLUE}=== Setting up Kind cluster ===${NC}"
    if ! check_kind_cluster; then
        echo "Creating Kind cluster..."
        bash "$PROJECT_ROOT/deployment/local/kind/setup.sh"
        CREATED_CLUSTER=true
        echo -e "${GREEN}Kind cluster created${NC}"
    fi
}

check_mongodb() {
    echo -e "${BLUE}=== Checking MongoDB deployment ===${NC}"
    if kubectl get statefulset mongo -n mongo &> /dev/null; then
        READY=$(kubectl get pods -n mongo -l app=mongo -o jsonpath='{.items[*].status.containerStatuses[0].ready}' 2>/dev/null || echo "")
        COUNT=$(echo "$READY" | tr -cd 't' | wc -c | xargs)
        if [ "$COUNT" -eq 3 ]; then
            echo -e "${GREEN}MongoDB is deployed and all 3 pods are ready${NC}"
            return 0
        else
            echo -e "${YELLOW}MongoDB pods not all ready ($COUNT/3)${NC}"
            return 1
        fi
    else
        echo -e "${YELLOW}MongoDB not deployed${NC}"
        return 1
    fi
}

setup_mongodb() {
    echo -e "${BLUE}=== Setting up MongoDB ===${NC}"
    if ! check_mongodb; then
        echo "Deploying MongoDB..."
        bash "$PROJECT_ROOT/deployment/local/mongo/setup.sh"
        echo -e "${GREEN}MongoDB deployed${NC}"
    fi
}

setup_port_forward() {
    echo -e "${BLUE}=== Setting up MongoDB port forwarding ===${NC}"

    lsof -ti:27017 | xargs kill -9 2>/dev/null || true
    sleep 1

    kubectl port-forward -n mongo svc/mongo 27017:27017 > /tmp/mongo-port-forward.log 2>&1 &
    PF_PID=$!
    echo $PF_PID > "$PORT_FORWARD_PID_FILE"
    STARTED_PORT_FORWARD=true

    echo "Waiting for port forward to be ready..."
    MAX_WAIT=10
    for i in $(seq 1 $MAX_WAIT); do
        if lsof -i:27017 > /dev/null 2>&1; then
            echo -e "${GREEN}Port forwarding established (PID: $PF_PID)${NC}"
            return 0
        fi
        sleep 1
    done

    echo -e "${RED}Port forwarding failed to start${NC}"
    cat /tmp/mongo-port-forward.log
    exit 1
}

run_migrations() {
    echo -e "${BLUE}=== Running migrations on test database ===${NC}"
    MONGO_URI="$TEST_MONGO_URI" \
    MONGO_DATABASE_NAME="$TEST_DB_NAME" \
    go run "$PROJECT_ROOT/cmd/migrate/main.go"
    echo -e "${GREEN}Migrations complete${NC}"
}

build_app() {
    echo -e "${BLUE}=== Building application ===${NC}"
    go build -o "$APP_BINARY" "$PROJECT_ROOT/cmd/business-units"
    echo -e "${GREEN}Build complete${NC}"
}

start_app() {
    echo -e "${BLUE}=== Starting application on port $TEST_SERVER_PORT ===${NC}"

    PORT="$TEST_SERVER_PORT" \
    MONGO_URI="$TEST_MONGO_URI" \
    MONGO_DATABASE_NAME="$TEST_DB_NAME" \
    LOG_LEVEL="info" \
    "$APP_BINARY" > /tmp/business-units-test.log 2>&1 &

    APP_PID=$!
    echo $APP_PID > "$APP_PID_FILE"
    STARTED_APP=true
    echo "Application started with PID: $APP_PID"

    echo -e "${YELLOW}Waiting for application to be ready...${NC}"
    MAX_WAIT=30
    WAIT_COUNT=0
    while [ $WAIT_COUNT -lt $MAX_WAIT ]; do
        if curl -s "http://localhost:$TEST_SERVER_PORT/health" > /dev/null 2>&1; then
            echo -e "${GREEN}Application is ready!${NC}"
            return 0
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
}

run_tests() {
    echo -e "${BLUE}=== Running integration tests ===${NC}"
    TEST_SERVER_URL="http://localhost:$TEST_SERVER_PORT" \
    TEST_MONGO_URI="$TEST_MONGO_URI" \
    TEST_DB_NAME="$TEST_DB_NAME" \
    TEST_SERVER_PORT="$TEST_SERVER_PORT" \
    go test -v "$PROJECT_ROOT/test/integration/..." -count=1
}

show_results() {
    local exit_code=$1
    echo ""
    if [ $exit_code -eq 0 ]; then
        echo -e "${GREEN}╔════════════════════════════════════════════════════════════╗${NC}"
        echo -e "${GREEN}║  ✅ All tests passed!                                      ║${NC}"
        echo -e "${GREEN}╚════════════════════════════════════════════════════════════╝${NC}"
    else
        echo -e "${RED}╔════════════════════════════════════════════════════════════╗${NC}"
        echo -e "${RED}║  ❌ Some tests failed                                       ║${NC}"
        echo -e "${RED}╚════════════════════════════════════════════════════════════╝${NC}"
        echo ""
        echo "Application logs (last 50 lines):"
        tail -n 50 /tmp/business-units-test.log
    fi
}

main() {
    set_colors
    set_variables
    load_env_file

    trap cleanup EXIT INT TERM

    echo -e "${GREEN}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║  Integration Test Orchestration                            ║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════════════════════════╝${NC}"
    echo ""

    setup_kind
    setup_mongodb
    setup_port_forward
    run_migrations
    build_app
    start_app

    echo ""
    run_tests
    TEST_EXIT_CODE=$?

    show_results $TEST_EXIT_CODE
    exit $TEST_EXIT_CODE
}

main "$@"
