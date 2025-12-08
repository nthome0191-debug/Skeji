#!/usr/bin/env bash
set -e

MIGRATION_NAMESPACE="jobs"
IMAGE_NAME="skeji-migrate:latest"
DOCKERFILE="build/docker/migrate.Dockerfile"
JOB_FILE="deployment/local/migrate/migrate-job.yaml"
JOB_NAME="skeji-migrate"

echo "🔹 Ensuring $MIGRATION_NAMESPACE namespace exists..."
kubectl get ns $MIGRATION_NAMESPACE >/dev/null 2>&1 || kubectl create namespace $MIGRATION_NAMESPACE

kubectl delete job $JOB_NAME -n $MIGRATION_NAMESPACE --ignore-not-found=true
docker build -t $IMAGE_NAME -f $DOCKERFILE .
kind load docker-image $IMAGE_NAME --name kind-hera-local
kubectl apply -n $MIGRATION_NAMESPACE -f $JOB_FILE

echo "⏳ Waiting for migration job to complete or fail..."

for i in {1..90}; do
    succeeded=$(kubectl get job $JOB_NAME -n $MIGRATION_NAMESPACE -o jsonpath='{.status.succeeded}' 2>/dev/null || echo "")
    failed=$(kubectl get job $JOB_NAME -n $MIGRATION_NAMESPACE -o jsonpath='{.status.failed}' 2>/dev/null || echo "")

    if [[ "$succeeded" == "1" ]]; then
        echo "✅ Migration job completed successfully."
        kubectl logs job/$JOB_NAME -n $MIGRATION_NAMESPACE
        exit 0
    fi

    if [[ -n "$failed" && "$failed" -gt 0 ]]; then
        echo "❌ Migration job failed."
        kubectl logs job/$JOB_NAME -n $MIGRATION_NAMESPACE
        exit 1
    fi

    sleep 2
done

echo "⚠️ Timeout waiting for migration job to finish — printing logs for context"
kubectl logs job/$JOB_NAME -n $MIGRATION_NAMESPACE || true
exit 1
