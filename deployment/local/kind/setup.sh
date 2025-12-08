#!/usr/bin/env bash
set -euo pipefail

CLUSTER_NAME="kind-hera-local"
BASE_DIR="$(dirname "$0")"

check_binaries() {
  echo "=== Checking required binaries ==="
  command -v kind >/dev/null 2>&1 || { echo "kind not found"; exit 1; }
  command -v kubectl >/dev/null 2>&1 || { echo "kubectl not found"; exit 1; }
  command -v docker >/dev/null 2>&1 || { echo "docker not found"; exit 1; }
  command -v jq >/dev/null 2>&1 || { echo "jq not found"; exit 1; }
}

check_docker() {
  echo "=== Checking Docker daemon ==="
  docker info >/dev/null 2>&1 || { echo "docker daemon not running"; exit 1; }
  echo "docker daemon running"
}

cluster_exists() {
  kind get clusters | grep -q "$CLUSTER_NAME"
}

check_ports_if_needed() {
  if cluster_exists; then
    echo "=== Cluster exists, skipping host port check ==="
    return
  fi

  echo "=== Checking port availability (27017, 9094) ==="
  lsof -Pi :27017 -sTCP:LISTEN >/dev/null 2>&1 && { echo "port 27017 already in use"; exit 1; }
  lsof -Pi :9094 -sTCP:LISTEN >/dev/null 2>&1 && { echo "port 9094 already in use"; exit 1; }
  echo "ports available"
}

check_or_create_cluster() {
  echo "=== Checking Kind cluster existence ==="
  if ! cluster_exists; then
    echo "creating kind cluster $CLUSTER_NAME"
    kind create cluster --name "$CLUSTER_NAME" --config "$BASE_DIR/cluster.yaml"
  else
    echo "kind cluster $CLUSTER_NAME already exists"
  fi
}

check_cluster_api() {
  echo "=== Checking cluster API availability ==="
  kubectl cluster-info >/dev/null 2>&1 || { echo "cluster api not responding"; exit 1; }
  echo "cluster api responding"
}

verify_node_count() {
  echo "=== Verifying node count ==="
  NODE_COUNT=$(kubectl get nodes --no-headers | wc -l | tr -d ' ')
  if [ "$NODE_COUNT" -ne 5 ]; then
    echo "wrong node count: expected 5"
    exit 1
  fi
  echo "node count correct"
}

verify_node_roles() {
  echo "=== Ensuring node roles ==="
  CONTROL=$(kubectl get nodes -o json | jq '.items[] | select(.metadata.labels."node-role.kubernetes.io/control-plane" != null)' | jq -s 'length')
  if [ "$CONTROL" -ne 1 ]; then
    echo "expected exactly one control-plane node"
    exit 1
  fi
  WORKERS=$(kubectl get nodes -o json | jq '.items[] | select(.metadata.labels."node-role.kubernetes.io/control-plane" == null)' | jq -s 'length')
  if [ "$WORKERS" -ne 4 ]; then
    echo "expected 4 worker nodes"
    exit 1
  fi
  echo "node role configuration correct"
}

verify_node_labels() {
  echo "=== Checking required node labels ==="
  APP=$(kubectl get nodes -l skeji.io/type=app --no-headers | wc -l | tr -d ' ')
  INFRA=$(kubectl get nodes -l skeji.io/type=infra --no-headers | wc -l | tr -d ' ')
  if [ "$APP" -ne 1 ]; then
    echo "missing or invalid app node label"
    exit 1
  fi
  if [ "$INFRA" -ne 3 ]; then
    echo "missing or invalid infra node labels"
    exit 1
  fi
  echo "node labels correct"
}

main() {
  echo "=== Spinning up local environment ==="
  check_binaries
  check_docker
  check_ports_if_needed
  check_or_create_cluster
  check_cluster_api
  verify_node_count
  verify_node_roles
  verify_node_labels
  echo "=== Setup complete ==="
}

main
