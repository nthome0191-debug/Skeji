#!/usr/bin/env bash
set -euo pipefail

MONGO_NAMESPACE="mongo"
BASE_DIR="$(dirname "$0")"
MAX_RETRIES=20
RETRY_INTERVAL=5
MONGO_CMD="mongosh"

main() {
  check_namespace
  deploy_statefulset
  deploy_service
  wait_for_statefulset_ready
  wait_for_pods_ready
  detect_mongo_cmd
  ensure_replica_set
  wait_for_primary
  ensure_root_user
  verify_authentication
  echo "✅ MongoDB replica set initialized, PRIMARY ready, root user created, and authentication verified."
}

check_namespace() {
  echo "=== 🧩 Checking mongo namespace '$MONGO_NAMESPACE' ==="
  if ! kubectl get ns "$MONGO_NAMESPACE" &> /dev/null; then
    echo "Creating mongo namespace $MONGO_NAMESPACE"
    kubectl create namespace "$MONGO_NAMESPACE"
  else
    echo "Namespace $MONGO_NAMESPACE already exists."
  fi
}

deploy_statefulset() {
  echo "=== 🍃 Checking MongoDB StatefulSet ==="
  if ! kubectl get statefulset mongo -n "$MONGO_NAMESPACE" &> /dev/null; then
    echo "Deploying MongoDB replica set..."
    kubectl apply -n "$MONGO_NAMESPACE" -f "$BASE_DIR/mongo-replicaset.yaml"
  else
    echo "MongoDB StatefulSet already exists."
  fi
}

deploy_service() {
  echo "=== 🍃 Checking MongoDB service ==="
  if ! kubectl get svc mongo -n "$MONGO_NAMESPACE" &> /dev/null; then
    kubectl apply -n "$MONGO_NAMESPACE" -f "$BASE_DIR/mongo-service.yaml"
  else
    echo "MongoDB service already exists."
  fi
}

wait_for_statefulset_ready() {
  echo "=== 🔄 Waiting for MongoDB StatefulSet readiness ==="
  kubectl rollout status statefulset/mongo -n "$MONGO_NAMESPACE" --timeout=300s
}

wait_for_pods_ready() {
  echo "=== 🩺 Waiting for all MongoDB pods to be ready ==="
  for i in $(seq 1 $MAX_RETRIES); do
    READY=$(kubectl get pods -n "$MONGO_NAMESPACE" -l app=mongo -o jsonpath='{.items[*].status.containerStatuses[0].ready}' 2>/dev/null || echo "")
    COUNT=$(echo "$READY" | tr -cd 't' | wc -c | xargs)
    if [ "$COUNT" -eq 3 ]; then
      echo "✅ All 3 MongoDB pods are ready."
      return
    fi
    echo "MongoDB pods not ready yet... ($COUNT/3 ready)"
    sleep "$RETRY_INTERVAL"
  done
  echo "❌ Pods failed to become ready."
  exit 1
}

detect_mongo_cmd() {
  local pod
  pod=$(get_first_pod)
  if ! kubectl exec -n "$MONGO_NAMESPACE" "$pod" -- which mongosh &>/dev/null; then
    MONGO_CMD="mongo"
  fi
}

ensure_replica_set() {
  local pod
  pod=$(get_first_pod)

  echo "=== ⚙️ Checking if replica set is initialized ==="
  if kubectl exec -n "$MONGO_NAMESPACE" "$pod" -- "$MONGO_CMD" --quiet --eval "try { rs.status().ok } catch(e) { 0 }" | grep -q "1"; then
    echo "Replica set already initialized."
    return
  fi

  echo "=== 🚀 Initializing MongoDB replica set ==="
  kubectl exec -n "$MONGO_NAMESPACE" "$pod" -- "$MONGO_CMD" --quiet --eval "
  rs.initiate({
    _id: 'rs0',
    members: [
      { _id: 0, host: 'mongo-0.mongo.mongo.svc.cluster.local:27017' },
      { _id: 1, host: 'mongo-1.mongo.mongo.svc.cluster.local:27017' },
      { _id: 2, host: 'mongo-2.mongo.mongo.svc.cluster.local:27017' }
    ]
  });
  "
}

wait_for_primary() {
  echo "=== 🕓 Waiting for PRIMARY election ==="
  local pod primary=""
  pod=$(get_first_pod)

  for i in $(seq 1 $MAX_RETRIES); do
    primary=$(kubectl exec -n "$MONGO_NAMESPACE" "$pod" -- "$MONGO_CMD" --quiet --eval "
      try {
        rs.status().members.filter(m => m.stateStr === 'PRIMARY')[0].name
      } catch(e) { '' }
    " | tr -d '[:space:]')
    if [ -n "$primary" ]; then
      echo "✅ PRIMARY elected: $primary"
      PRIMARY_POD=$(echo "$primary" | cut -d'.' -f1)
      export PRIMARY_POD
      return
    fi
    echo "No PRIMARY yet... retrying in ${RETRY_INTERVAL}s"
    sleep "$RETRY_INTERVAL"
  done

  echo "❌ No PRIMARY elected after $((MAX_RETRIES * RETRY_INTERVAL)) seconds."
  exit 1
}

ensure_root_user() {
  echo "=== 🔐 Ensuring root user exists on PRIMARY ($PRIMARY_POD) ==="
  local exists
  exists=$(kubectl exec -n "$MONGO_NAMESPACE" "$PRIMARY_POD" -- "$MONGO_CMD" --quiet --eval "
    try {
      db.getSiblingDB('admin').system.users.find({user:'root'}).count()
    } catch(e) { 0 }
  ")

  if [ "$exists" -eq 0 ]; then
    echo "Creating root user..."
    kubectl exec -n "$MONGO_NAMESPACE" "$PRIMARY_POD" -- "$MONGO_CMD" --quiet --eval "
      db.getSiblingDB('admin').createUser({
        user: 'root',
        pwd: 'rootpassword',
        roles: [ { role: 'root', db: 'admin' } ]
      });
    "
  else
    echo "Root user already exists."
  fi
}

verify_authentication() {
  echo "=== 🧩 Verifying cluster authentication ==="
  kubectl exec -n "$MONGO_NAMESPACE" "$PRIMARY_POD" -- "$MONGO_CMD" "mongodb://root:rootpassword@mongo-0.mongo.mongo.svc.cluster.local:27017,mongo-1.mongo.mongo.svc.cluster.local:27017,mongo-2.mongo.mongo.svc.cluster.local:27017/admin?replicaSet=rs0" --quiet --eval "db.runCommand({ping:1})"
}

get_first_pod() {
  kubectl get pods -n "$MONGO_NAMESPACE" -l app=mongo -o jsonpath="{.items[0].metadata.name}"
}

main "$@"
