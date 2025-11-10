#!/usr/bin/env bash
set -euo pipefail

ACTION="${1:-}"
if [[ "$ACTION" != "--setup" && "$ACTION" != "--clean" ]]; then
  echo "Usage: $0 [--setup | --clean]"
  exit 1
fi

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
    TEST_MONGO_URI=${TEST_MONGO_URI:-"mongodb://root:rootpassword@localhost:27017/?directConnection=true&authSource=admin"}
    TEST_DB_NAME=${TEST_DB_NAME:-"skeji_test"}
    APP_PID_FILE="/tmp/business-units-test.pid"
    PORT_FORWARD_PID_FILE="/tmp/mongo-port-forward.pid"
    LOCK_FILE="/tmp/skeji_integration.lock"
}

cleanup_runtime() {
    echo -e "${YELLOW}🧹 Cleaning up test runtime...${NC}"

    # Stop app
    if [ -f "$APP_PID_FILE" ]; then
        APP_PID=$(cat "$APP_PID_FILE")
        if ps -p "$APP_PID" &>/dev/null; then
            echo "Stopping app PID $APP_PID"
            kill -TERM "$APP_PID" || true
            sleep 1
            kill -9 "$APP_PID" 2>/dev/null || true
        fi
        rm -f "$APP_PID_FILE"
    fi

    # Stop port-forward
    if [ -f "$PORT_FORWARD_PID_FILE" ]; then
        PF_PID=$(cat "$PORT_FORWARD_PID_FILE")
        if ps -p "$PF_PID" &>/dev/null; then
            echo "Stopping Mongo port-forward PID $PF_PID"
            kill -TERM "$PF_PID" || true
            sleep 1
            kill -9 "$PF_PID" 2>/dev/null || true
        fi
        rm -f "$PORT_FORWARD_PID_FILE"
    fi

    pkill -f "business-units" 2>/dev/null || true
    rm -f "$LOCK_FILE"
    echo -e "${GREEN}✅ Cleanup complete${NC}"
}

setup_kind() {
    echo -e "${BLUE}=== Setting up Kind cluster ===${NC}"
    cd "$PROJECT_ROOT"
    make kind-up
}

setup_mongo() {
    echo -e "${BLUE}=== Setting up MongoDB ===${NC}"
    cd "$PROJECT_ROOT"
    make mongo-up
}

setup_port_forward() {
    echo -e "${BLUE}=== Setting up MongoDB port-forward ===${NC}"
    pkill -f "kubectl port-forward.*mongo.*27017" 2>/dev/null || true
    sleep 1
    local FIRST_POD PRIMARY PRIMARY_POD
    FIRST_POD=$(kubectl get pods -n mongo -l app=mongo -o name | head -1 | sed 's|pod/||')
    PRIMARY=$(kubectl exec -n mongo "$FIRST_POD" -- mongosh --quiet --eval \
      'rs.status().members.find(m => m.stateStr==="PRIMARY").name' 2>/dev/null | tr -d '[:space:]')
    PRIMARY_POD=$(echo "$PRIMARY" | cut -d'.' -f1)

    echo "Port-forwarding to primary pod: $PRIMARY_POD"
    kubectl port-forward -n mongo "$PRIMARY_POD" 27017:27017 > /tmp/mongo-port-forward.log 2>&1 &
    echo $! > "$PORT_FORWARD_PID_FILE"
    sleep 3
    echo -e "${GREEN}✅ Port-forward ready${NC}"
}

run_migrations() {
    echo -e "${BLUE}=== Running migrations ===${NC}"
    cd "$PROJECT_ROOT"
    MONGO_URI="$TEST_MONGO_URI" \
    MONGO_DATABASE_NAME="$TEST_DB_NAME" \
    go run ./cmd/migrate/main.go
    echo -e "${GREEN}✅ Migrations done${NC}"
}

main() {
    set_colors
    set_vars

    case "$ACTION" in
        --setup)
            echo -e "${BLUE}🔧 Setting up infra...${NC}"
            setup_kind
            setup_mongo
            setup_port_forward
            run_migrations
            echo -e "${GREEN}✅ Infra ready${NC}"
            ;;
        --clean)
            cleanup_runtime
            ;;
    esac
}

main "$@"
