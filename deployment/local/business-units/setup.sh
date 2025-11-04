#!/usr/bin/env bash
set -e

APP_NAMESPACE="apps"
IMAGE_NAME="skeji/business-units:latest"
DOCKERFILE="build/docker/business-units.Dockerfile"
DEPLOYMENT_FILE="deployment/local/business-units/business-units-deployment.yaml"
SERVICE_FILE="deployment/local/business-units/business-units-service.yaml"
DEPLOYMENT_NAME="business-units"
SERVICE_NAME="business-units"

echo "🔹 Ensuring $APP_NAMESPACE namespace exists..."
kubectl get ns $APP_NAMESPACE >/dev/null 2>&1 || kubectl create namespace $APP_NAMESPACE

echo "🚀 Building Docker image..."
docker build -t $IMAGE_NAME -f $DOCKERFILE .

echo "📦 Loading image into Kind cluster..."
kind load docker-image $IMAGE_NAME --name skeji-local

echo "📄 Applying Kubernetes manifests..."
kubectl apply -n $APP_NAMESPACE -f $DEPLOYMENT_FILE
kubectl apply -n $APP_NAMESPACE -f $SERVICE_FILE

echo "⏳ Waiting for deployment rollout to complete..."

if ! kubectl rollout status deployment/$DEPLOYMENT_NAME -n $APP_NAMESPACE --timeout=180s; then
    echo "❌ Deployment rollout failed or timed out. Fetching logs..."
    kubectl get pods -n $APP_NAMESPACE -l app=$DEPLOYMENT_NAME
    POD_NAME=$(kubectl get pods -n $APP_NAMESPACE -l app=$DEPLOYMENT_NAME -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
    if [[ -n "$POD_NAME" ]]; then
        echo "📜 Logs from pod $POD_NAME:"
        kubectl logs "$POD_NAME" -n $APP_NAMESPACE || true
    fi
    exit 1
fi

echo "✅ Business Units service deployed successfully!"

POD_NAME=$(kubectl get pods -n $APP_NAMESPACE -l app=$DEPLOYMENT_NAME -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)

if [[ -n "$POD_NAME" ]]; then
    echo "📜 Fetching logs from pod $POD_NAME..."
    kubectl logs "$POD_NAME" -n $APP_NAMESPACE --tail=50
else
    echo "⚠️ No running pods found for $DEPLOYMENT_NAME."
fi
