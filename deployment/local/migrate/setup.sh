#!/usr/bin/env bash
set -e

NAMESPACE="mongo"
IMAGE_NAME="skeji-migrate:latest"
DOCKERFILE="build/docker/migrate.Dockerfile"
JOB_FILE="deployment/local/migrate/migrate-job.yaml"
JOB_NAME="skeji-migrate"

kubectl delete job $JOB_NAME -n $NAMESPACE --ignore-not-found=true
docker build -t $IMAGE_NAME -f $DOCKERFILE .
kind load docker-image $IMAGE_NAME --name skeji-local
kubectl apply -n $NAMESPACE -f $JOB_FILE

echo "⏳ Waiting for migration job to complete or fail..."

for i in {1..90}; do
    succeeded=$(kubectl get job $JOB_NAME -n $NAMESPACE -o jsonpath='{.status.succeeded}' 2>/dev/null || echo "")
    failed=$(kubectl get job $JOB_NAME -n $NAMESPACE -o jsonpath='{.status.failed}' 2>/dev/null || echo "")

    if [[ "$succeeded" == "1" ]]; then
        echo "✅ Migration job completed successfully."
        kubectl logs job/$JOB_NAME -n $NAMESPACE
        exit 0
    fi

    if [[ -n "$failed" && "$failed" -gt 0 ]]; then
        echo "❌ Migration job failed."
        kubectl logs job/$JOB_NAME -n $NAMESPACE
        exit 1
    fi

    sleep 2
done

echo "⚠️ Timeout waiting for migration job to finish — printing logs for context"
kubectl logs job/$JOB_NAME -n $NAMESPACE || true
exit 1
