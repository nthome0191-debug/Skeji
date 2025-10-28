#!/usr/bin/env bash
set -euo pipefail

MONGO_NAMESPACE="mongo"
BASE_DIR="$(dirname "$0")"
MONGO_POD=""
MAX_RETRIES=20
RETRY_INTERVAL=5
MONGO_CMD="mongosh"

echo "=== 🧩 Checking mongo namespace '$MONGO_NAMESPACE' ==="
if ! kubectl get ns "$MONGO_NAMESPACE" &> /dev/null; then
    echo "Creating mongo namespace $MONGO_NAMESPACE"
    kubectl create namespace "$MONGO_NAMESPACE"
else
    echo "Namespace $MONGO_NAMESPACE already exists."
fi

echo "=== 🍃 Checking MongoDB deployment ==="
if ! kubectl get deploy mongo -n "$MONGO_NAMESPACE" &> /dev/null; then
    echo "Deploying MongoDB..."
    kubectl apply -n "$MONGO_NAMESPACE" -f "$BASE_DIR/mongo-pvc.yaml"
    kubectl apply -n "$MONGO_NAMESPACE" -f "$BASE_DIR/mongo-deployment.yaml"
    kubectl apply -n "$MONGO_NAMESPACE" -f "$BASE_DIR/mongo-service.yaml"
else
    echo "MongoDB already deployed."
fi

echo "=== 🔄 Waiting for MongoDB readiness ==="
kubectl rollout status deployment/mongo -n "$MONGO_NAMESPACE" --timeout=120s

echo "=== 🩺 Verifying MongoDB health ==="

MONGO_POD=$(kubectl get pods -n "$MONGO_NAMESPACE" -l app=mongo -o jsonpath="{.items[0].metadata.name}")

if ! kubectl exec -n "$MONGO_NAMESPACE" "$MONGO_POD" -- which mongosh &>/dev/null; then
    MONGO_CMD="mongo"
fi

for i in $(seq 1 $MAX_RETRIES); do
    echo "Attempt $i: checking connectivity inside pod $MONGO_POD ..."
    if kubectl exec -n "$MONGO_NAMESPACE" "$MONGO_POD" -- "$MONGO_CMD" --eval "db.adminCommand('ping')" &>/dev/null; then
        echo "✅ MongoDB is healthy and responding to connections."
        exit 0
    fi
    echo "MongoDB not ready yet... retrying in ${RETRY_INTERVAL}s"
    sleep "$RETRY_INTERVAL"
done

echo "❌ MongoDB did not become healthy after $((MAX_RETRIES * RETRY_INTERVAL)) seconds."
exit 1
