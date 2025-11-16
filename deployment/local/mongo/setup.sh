#!/usr/bin/env bash
set -euo pipefail

MONGO_NAMESPACE="mongo"
BASE_DIR="$(dirname "$0")"
RETRIES=20
INTERVAL=5
MONGO_CMD="mongosh"

main() {
  ensure_namespace
  apply_statefulset
  apply_service
  wait_statefulset_rollout
  wait_pods_ready
  detect_mongo_binary
  init_replica_set
  wait_primary_election
  ensure_root_user
  verify_auth
  echo "MongoDB replica set ready"
}

ensure_namespace() {
  echo "=== Checking namespace $MONGO_NAMESPACE ==="
  if kubectl get ns "$MONGO_NAMESPACE" >/dev/null 2>&1; then
    echo "namespace exists"
  else
    kubectl create namespace "$MONGO_NAMESPACE"
  fi
}

apply_statefulset() {
  echo "=== Checking MongoDB StatefulSet ==="
  if kubectl -n "$MONGO_NAMESPACE" get statefulset mongo >/dev/null 2>&1; then
    echo "statefulset already exists"
  else
    kubectl apply -n "$MONGO_NAMESPACE" -f "$BASE_DIR/mongo-replicaset.yaml"
  fi
}

apply_service() {
  echo "=== Checking MongoDB service ==="
  if kubectl -n "$MONGO_NAMESPACE" get svc mongo >/dev/null 2>&1; then
    echo "service already exists"
  else
    kubectl apply -n "$MONGO_NAMESPACE" -f "$BASE_DIR/mongo-service.yaml"
  fi
}

wait_statefulset_rollout() {
  echo "=== Waiting for StatefulSet rollout ==="
  kubectl rollout status statefulset/mongo -n "$MONGO_NAMESPACE" --timeout=300s
}

wait_pods_ready() {
  echo "=== Waiting for MongoDB pods ==="
  for _ in $(seq 1 "$RETRIES"); do
    READY=$(kubectl get pods -n "$MONGO_NAMESPACE" -l app=mongo -o jsonpath='{.items[*].status.containerStatuses[0].ready}' 2>/dev/null || echo "")
    COUNT=$(echo "$READY" | tr -cd 't' | wc -c | xargs)
    if [ "$COUNT" -eq 3 ]; then
      echo "all 3 pods ready"
      return
    fi
    echo "pods not ready ($COUNT/3)"
    sleep "$INTERVAL"
  done
  echo "pods did not become ready"
  exit 1
}

detect_mongo_binary() {
  POD=$(first_pod)
  if kubectl exec -n "$MONGO_NAMESPACE" "$POD" -- which mongosh >/dev/null 2>&1; then
    MONGO_CMD="mongosh"
  else
    MONGO_CMD="mongo"
  fi
}

init_replica_set() {
  POD=$(first_pod)
  echo "=== Checking replica set ==="
  if kubectl exec -n "$MONGO_NAMESPACE" "$POD" -- "$MONGO_CMD" --quiet --eval "try{rs.status().ok}catch(e){0}" | grep -q "1"; then
    echo "replica set already initialized"
    return
  fi

  echo "initializing replica set"
  kubectl exec -n "$MONGO_NAMESPACE" "$POD" -- "$MONGO_CMD" --quiet --eval "
    rs.initiate({
      _id: 'rs0',
      members: [
        { _id: 0, host: 'mongo-0.mongo.mongo.svc.cluster.local:27017' },
        { _id: 1, host: 'mongo-1.mongo.mongo.svc.cluster.local:27017' },
        { _id: 2, host: 'mongo-2.mongo.mongo.svc.cluster.local:27017' }
      ]
    })
  "
}

wait_primary_election() {
  echo "=== Waiting for PRIMARY ==="
  POD=$(first_pod)
  for _ in $(seq 1 "$RETRIES"); do
    PRIMARY=$(kubectl exec -n "$MONGO_NAMESPACE" "$POD" -- "$MONGO_CMD" --quiet --eval "
      try{rs.status().members.filter(m=>m.stateStr==='PRIMARY')[0].name}catch(e){''}
    " | tr -d '[:space:]')
    if [ -n "$PRIMARY" ]; then
      PRIMARY_POD=$(echo "$PRIMARY" | cut -d'.' -f1)
      export PRIMARY_POD
      echo "PRIMARY is $PRIMARY_POD"
      return
    fi
    echo "waiting for PRIMARY"
    sleep "$INTERVAL"
  done
  echo "no PRIMARY elected"
  exit 1
}

ensure_root_user() {
  echo "=== Ensuring root user exists ==="
  EXISTS=$(kubectl exec -n "$MONGO_NAMESPACE" "$PRIMARY_POD" -- "$MONGO_CMD" --quiet --eval "
    try{db.getSiblingDB('admin').system.users.find({user:'root'}).count()}catch(e){0}
  ")
  if [ "$EXISTS" -eq 0 ]; then
    echo "creating root user"
    kubectl exec -n "$MONGO_NAMESPACE" "$PRIMARY_POD" -- "$MONGO_CMD" --quiet --eval "
      db.getSiblingDB('admin').createUser({user:'root',pwd:'rootpassword',roles:[{role:'root',db:'admin'}]})
    "
  else
    echo "root user exists"
  fi
}

verify_auth() {
  echo "=== Verifying authentication ==="
  kubectl exec -n "$MONGO_NAMESPACE" "$PRIMARY_POD" -- "$MONGO_CMD" \
    "mongodb://root:rootpassword@mongo-0.mongo.mongo.svc.cluster.local:27017,mongo-1.mongo.mongo.svc.cluster.local:27017,mongo-2.mongo.mongo.svc.cluster.local:27017/admin?replicaSet=rs0" \
    --quiet --eval "db.runCommand({ping:1})"
}

first_pod() {
  kubectl get pods -n "$MONGO_NAMESPACE" -l app=mongo -o jsonpath='{.items[0].metadata.name}'
}

main "$@"
