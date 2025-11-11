#!/usr/bin/env bash
set -euo pipefail

APP_NAME=""
VERBOSE=false
TEST_TARGET=""

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -t|--test)
                TEST_TARGET="$2"
                shift 2
                ;;
            -*)
                echo "Unknown flag: $1"
                echo "Usage: $0 <app-name> [-v|--verbose] [-t|--test <test-target>]"
                exit 1
                ;;
            *)
                APP_NAME="$1"
                shift
                ;;
        esac
    done

    if [[ -z "$APP_NAME" ]]; then
        echo "Usage: $0 <app-name> [-v|--verbose] [-t|--test <test-target>]"
        exit 1
    fi
}

set_colors() {
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    NC='\033[0m'
}

set_vars() {
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

    if [[ -f "$PROJECT_ROOT/.env.test" ]]; then
        echo -e "${YELLOW}Loading environment from .env.test${NC}"
        set -a
        source "$PROJECT_ROOT/.env.test"
        set +a
    fi

    TEST_SERVER_PORT=${TEST_SERVER_PORT:-8080}
    TEST_MONGO_URI=${TEST_MONGO_URI:-"mongodb://localhost:27017/?directConnection=true"}
    TEST_DB_NAME=${TEST_DB_NAME:-"skeji_test"}

    APP_BINARY="$PROJECT_ROOT/bin/$APP_NAME"
    APP_PID_FILE="/tmp/${APP_NAME}-test.pid"
}

build_app() {
    echo -e "${BLUE}=== Building $APP_NAME ===${NC}"
    cd "$PROJECT_ROOT"
    go clean -cache -testcache >/dev/null 2>&1 || true
    rm -f "$APP_BINARY"
    go build -o "$APP_BINARY" "./cmd/$APP_NAME"
    echo -e "${GREEN}✅ Build complete${NC}"
}

start_app() {
    echo -e "${BLUE}=== Starting $APP_NAME ===${NC}"
    local LOG_FILE="/tmp/${APP_NAME}-test.log"

    if lsof -ti:$TEST_SERVER_PORT &>/dev/null; then
        echo -e "${YELLOW}Port $TEST_SERVER_PORT in use, freeing it...${NC}"
        lsof -ti:$TEST_SERVER_PORT | xargs kill -9 2>/dev/null || true
        sleep 1
    fi

    export MONGO_URI="mongodb://root:rootpassword@localhost:27017/?directConnection=true&authSource=admin"
    export MONGO_DATABASE_NAME="$TEST_DB_NAME"
    export PORT="$TEST_SERVER_PORT"
    export LOG_LEVEL="${LOG_LEVEL:-info}"

    if $VERBOSE; then
        ("$APP_BINARY") 2>&1 | tee "$LOG_FILE" &
    else
        ("$APP_BINARY") >"$LOG_FILE" 2>&1 &
    fi
    echo $! > "$APP_PID_FILE"

    echo -e "${YELLOW}Waiting for readiness...${NC}"
    for _ in {1..30}; do
        if curl -fs "http://localhost:$TEST_SERVER_PORT/health" >/dev/null 2>&1; then
            echo -e "${GREEN}✅ $APP_NAME ready${NC}"
            sleep 1
            for _ in {1..10}; do
                if nc -z localhost "$TEST_SERVER_PORT" >/dev/null 2>&1; then
                    return
                fi
                sleep 0.5
            done
            return
        fi
        sleep 1
    done

    echo -e "${RED}❌ $APP_NAME failed to start${NC}"
    tail -n 50 "$LOG_FILE"
    exit 1
}

run_tests() {
    echo -e "${BLUE}=== Running integration tests for $APP_NAME ===${NC}"
    cd "$PROJECT_ROOT"
    local test_path="./test/integration/${APP_NAME}/..."
    local test_cmd="go test -v \"$test_path\" -count=1"

    if [[ -n "$TEST_TARGET" ]]; then
        echo -e "${YELLOW}Running specific test target: ${TEST_TARGET}${NC}"
        test_cmd="go test -v \"$test_path\" -run ${TEST_TARGET} -count=1"
    fi

    TEST_SERVER_URL="http://localhost:$TEST_SERVER_PORT" \
    TEST_MONGO_URI="$TEST_MONGO_URI" \
    TEST_DB_NAME="$TEST_DB_NAME" \
    bash -c "$test_cmd"
}

show_results() {
    local code=$1
    if [ $code -eq 0 ]; then
        echo -e "${GREEN}✅ All $APP_NAME tests passed${NC}"
    else
        echo -e "${RED}❌ Some $APP_NAME tests failed${NC}"
        tail -n 50 "/tmp/${APP_NAME}-test.log" || true
    fi
}

cleanup_app() {
    if [[ -f "$APP_PID_FILE" ]]; then
        local pid
        pid=$(cat "$APP_PID_FILE")
        if ps -p "$pid" &>/dev/null; then
            echo -e "${YELLOW}Stopping $APP_NAME (PID $pid)...${NC}"
            kill -TERM "$pid" || true
            sleep 1
            kill -9 "$pid" 2>/dev/null || true
        fi
        rm -f "$APP_PID_FILE"
        echo -e "${GREEN}✅ $APP_NAME stopped and cleaned up${NC}"
    fi
}

main() {
    set_colors
    parse_args "$@"
    set_vars
    trap cleanup_app EXIT INT TERM
    build_app
    start_app
    run_tests
    show_results $?
}

main "$@"
