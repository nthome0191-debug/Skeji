#!/usr/bin/env bash
set -euo pipefail

CLUSTER_NAME="skeji-local"
BASE_DIR="$(dirname "$0")"

echo "=== 🧩 Checking Kind installation ==="
if ! command -v kind &> /dev/null; then
    echo "Kind not found. Installing..."
    brew install kind || {
        echo "❌ Failed to install kind. Please install manually."
        exit 1
    }
else
    echo "Kind is installed."
fi

echo "=== 🧩 Checking Kind cluster ==="
if ! kind get clusters | grep -q "$CLUSTER_NAME"; then
    echo "Creating Kind cluster: $CLUSTER_NAME"
    kind create cluster --name "$CLUSTER_NAME" --config "$BASE_DIR/cluster.yaml"
else
    echo "Kind cluster $CLUSTER_NAME already exists."
fi
