#!/bin/bash

set -e

VERBOSE=false
while [[ $# -gt 0 ]]; do
    case "$1" in
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        *)
            shift
            ;;
    esac
done

set_colors() {
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    NC='\033[0m'
}

set_variables() {
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
    TEST_SERVER_PORT=${TEST_SERVER_PORT:-8080}
    TEST_MONGO_URI=${TEST_MONGO_URI:-"mongodb://root:rootpassword@localhost:27017/?directConnection=true&authSource=admin"}
    TEST_DB_NAME=${TEST_DB_NAME:-"skeji_test"}
    APP_BINARY="$PROJECT_ROOT/bin/business-units"
    APP_PID_FILE="/tmp/business-units-test.pid"
    PORT_FORWARD_PID_FILE="/tmp/mongo-port-forward.pid"
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

print_logo() {
    echo -e "${GREEN}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║  Integration Test Orchestration                            ║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

cleanup() {
    echo ""
    echo -e "${YELLOW}Cleaning up...${NC}"

    if [ -f "$APP_PID_FILE" ]; then
        APP_PID=$(cat "$APP_PID_FILE")
        if ps -p "$APP_PID" > /dev/null 2>&1; then
            echo "Stopping application (PID: $APP_PID)..."
            kill -TERM -"$APP_PID" 2>/dev/null || true
            sleep 2
            if ps -p "$APP_PID" > /dev/null 2>&1; then
                echo "Force killing application..."
                kill -9 -"$APP_PID" 2>/dev/null || true
            fi
        fi
        rm -f "$APP_PID_FILE"
    fi

    if [ -f "$PORT_FORWARD_PID_FILE" ]; then
        PF_PID=$(cat "$PORT_FORWARD_PID_FILE")
        if ps -p "$PF_PID" > /dev/null 2>&1; then
            echo "Stopping MongoDB port forwarding (PID: $PF_PID)..."
            kill -TERM "$PF_PID" 2>/dev/null || true
            sleep 1
            if ps -p "$PF_PID" > /dev/null 2>&1; then
                echo "Force killing port-forward..."
                kill -9 "$PF_PID" 2>/dev/null || true
            fi
        fi
        rm -f "$PORT_FORWARD_PID_FILE"
    fi

    pkill -f business-units || true
    echo -e "${GREEN}Cleanup complete${NC}"
}

check_existing_environment() {
    echo -e "${BLUE}=== Checking for existing environment ===${NC}"

    if pgrep -f "kubectl port-forward.*mongo.*27017" > /dev/null 2>&1; then
        if timeout 2 bash -c "cat < /dev/null > /dev/tcp/localhost/27017" 2>/dev/null; then
            echo -e "${GREEN}✅ Port forward active and MongoDB accessible${NC}"
            return 0
        fi
        pkill -f "kubectl port-forward.*mongo.*27017" 2>/dev/null || true
        sleep 1
    fi

    return 1
}

setup_kind() {
    echo -e "${BLUE}=== Setting up Kind cluster ===${NC}"
    cd "$PROJECT_ROOT"
    make kind-up
}

setup_mongodb() {
    echo -e "${BLUE}=== Setting up MongoDB ===${NC}"
    cd "$PROJECT_ROOT"
    make mongo-up
}

setup_port_forward() {
    echo -e "${BLUE}=== Setting up MongoDB port forwarding ===${NC}"

    pkill -f "kubectl port-forward.*mongo.*27017" 2>/dev/null || true
    sleep 1

    FIRST_POD=$(kubectl get pods -n mongo -l app=mongo -o name | head -1 | sed 's|pod/||')
    PRIMARY_HOST=$(kubectl exec -n mongo "$FIRST_POD" -- mongosh --quiet --eval "
      rs.status().members.filter(m => m.stateStr === 'PRIMARY')[0].name
    " 2>/dev/null | tr -d '[:space:]')
    PRIMARY_POD=$(echo "$PRIMARY_HOST" | cut -d'.' -f1)

    echo "Port-forwarding to primary pod: $PRIMARY_POD"
    kubectl port-forward -n mongo "$PRIMARY_POD" 27017:27017 > /tmp/mongo-port-forward.log 2>&1 &
    echo $! > "$PORT_FORWARD_PID_FILE"

    sleep 3
    echo -e "${GREEN}✅ Port forward established${NC}"
}

run_migrations() {
    echo -e "${BLUE}=== Running migrations on test database ===${NC}"
    cd "$PROJECT_ROOT"
    MONGO_URI="$TEST_MONGO_URI" \
    MONGO_DATABASE_NAME="$TEST_DB_NAME" \
    go run ./cmd/migrate/main.go
    echo -e "${GREEN}Migrations complete${NC}"
}

build_app() {
    echo -e "${BLUE}=== Building application ===${NC}"
    cd "$PROJECT_ROOT"
    go build -o "$APP_BINARY" ./cmd/business-units
    echo -e "${GREEN}Build complete${NC}"
}

start_app() {
    echo -e "${BLUE}=== Starting application on port $TEST_SERVER_PORT ===${NC}"

    if $VERBOSE; then
        (
            PORT="$TEST_SERVER_PORT" \
            MONGO_URI="$TEST_MONGO_URI" \
            MONGO_DATABASE_NAME="$TEST_DB_NAME" \
            LOG_LEVEL="info" \
            exec "$APP_BINARY"
        ) 2>&1 | tee /tmp/business-units-test.log &
    else
        (
            PORT="$TEST_SERVER_PORT" \
            MONGO_URI="$TEST_MONGO_URI" \
            MONGO_DATABASE_NAME="$TEST_DB_NAME" \
            LOG_LEVEL="info" \
            exec "$APP_BINARY"
        ) > /tmp/business-units-test.log 2>&1 &
    fi

    APP_PID=$!
    echo $APP_PID > "$APP_PID_FILE"
    echo "Application started with PID: $APP_PID"

    echo -e "${YELLOW}Waiting for application to be ready...${NC}"

    MAX_WAIT=30
    for i in $(seq 1 $MAX_WAIT); do
        if curl -s "http://localhost:$TEST_SERVER_PORT/health" > /dev/null 2>&1; then
            echo -e "${GREEN}Application is ready!${NC}"
            return 0
        fi
        sleep 1
    done

    echo -e "${RED}Application failed to start within $MAX_WAIT seconds${NC}"
    echo "Application logs:"
    tail -n 50 /tmp/business-units-test.log
    exit 1
}

run_tests() {
    echo -e "${BLUE}=== Running integration tests ===${NC}"
    cd "$PROJECT_ROOT"
    TEST_SERVER_URL="http://localhost:$TEST_SERVER_PORT" \
    TEST_MONGO_URI="$TEST_MONGO_URI" \
    TEST_DB_NAME="$TEST_DB_NAME" \
    TEST_SERVER_PORT="$TEST_SERVER_PORT" \
    go test -v ./test/integration/... -count=1
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
    print_logo

    if check_existing_environment; then
        echo -e "${GREEN}Reusing existing environment${NC}"
    else
        echo -e "${YELLOW}Setting up new environment${NC}"
        setup_kind
        setup_mongodb
        setup_port_forward
    fi

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
