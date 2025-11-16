#!/usr/bin/env bash
set -euo pipefail

KAFKA_NAMESPACE="kafka"
KAFKA_NAME="skeji-kafka"
BASE_DIR="$(dirname "$0")"
RETRIES=30
INTERVAL=10

main() {
  ensure_namespace
  check_strimzi
  ensure_strimzi_operator
  apply_kafka_cluster
  wait_kafka_pods_ready
  wait_kafka_cr_ready
  apply_topics
  echo "Kafka cluster ready"
}

ensure_namespace() {
  echo "=== Checking namespace $KAFKA_NAMESPACE ==="
  if kubectl get ns "$KAFKA_NAMESPACE" >/dev/null 2>&1; then
    echo "namespace exists"
  else
    kubectl create namespace "$KAFKA_NAMESPACE"
  fi
}

check_strimzi() {
  echo "=== Checking Strimzi operator installation ==="
  if kubectl get deployment -n kafka 2>/dev/null | grep -q "strimzi-cluster-operator"; then
    echo "strimzi already installed"
  else
    echo "installing strimzi"
    kubectl create namespace kafka >/dev/null 2>&1 || true
    kubectl apply -f https://strimzi.io/install/latest?namespace=kafka -n kafka
  fi
}

ensure_strimzi_operator() {
  echo "=== Checking Strimzi Cluster Operator in $KAFKA_NAMESPACE ==="
  if ! kubectl -n "$KAFKA_NAMESPACE" get deployment strimzi-cluster-operator >/dev/null 2>&1; then
    echo "strimzi-cluster-operator not found in namespace $KAFKA_NAMESPACE"
    exit 1
  fi
  kubectl -n "$KAFKA_NAMESPACE" rollout status deployment/strimzi-cluster-operator --timeout=300s
}

apply_kafka_cluster() {
  echo "=== Applying Kafka cluster and node pool ==="
  kubectl apply -n "$KAFKA_NAMESPACE" -f "$BASE_DIR/kafka-cluster.yaml"
}

wait_kafka_pods_ready() {
  echo "=== Waiting for Kafka pods ==="
  for _ in $(seq 1 "$RETRIES"); do
    READY=$(kubectl get pods -n "$KAFKA_NAMESPACE" -l strimzi.io/cluster="$KAFKA_NAME" -o jsonpath='{.items[*].status.containerStatuses[0].ready}' 2>/dev/null || echo "")
    COUNT=$(echo "$READY" | tr -cd 't' | wc -c | xargs)
    if [ "$COUNT" -ge 3 ]; then
      echo "Kafka pods ready ($COUNT)"
      return
    fi
    echo "Kafka pods not ready ($COUNT)"
    sleep "$INTERVAL"
  done
  echo "Kafka pods did not become ready"
  exit 1
}

wait_kafka_cr_ready() {
  echo "=== Waiting for Kafka CR Ready condition ==="
  for _ in $(seq 1 "$RETRIES"); do
    STATUS=$(kubectl get kafka "$KAFKA_NAME" -n "$KAFKA_NAMESPACE" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "")
    if [ "$STATUS" = "True" ]; then
      echo "Kafka CR Ready=True"
      return
    fi
    echo "Kafka CR not ready yet"
    sleep "$INTERVAL"
  done
  echo "Kafka CR did not reach Ready=True"
  exit 1
}

apply_topics() {
  echo "=== Applying Kafka topics ==="
  if [ -f "$BASE_DIR/kafka-topics.yaml" ]; then
    kubectl apply -n "$KAFKA_NAMESPACE" -f "$BASE_DIR/kafka-topics.yaml"
  else
    echo "no kafka-topics.yaml found, skipping"
  fi
}

main "$@"
