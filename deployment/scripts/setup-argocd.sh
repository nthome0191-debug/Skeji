#!/bin/bash
set -e

ARGOCD_NAMESPACE="${ARGOCD_NAMESPACE:-argocd}"
ARGOCD_SERVER="${ARGOCD_SERVER:-argocd-server.argocd.svc.cluster.local}"
ARGOCD_AUTH_TOKEN="${ARGOCD_AUTH_TOKEN}"
GIT_REPO="${GIT_REPO}"
GIT_REVISION="${GIT_REVISION:-main}"
APP_NAME="${APP_NAME:-skeji}"
DEPLOYMENT_PATH="${DEPLOYMENT_PATH:-deployment}"
ARGOCD_VERSION="${ARGOCD_VERSION:-v2.12.3}"
SERVICES="${SERVICES}"

echo "🚀 Deploying $APP_NAME microservices to ArgoCD"
echo ""

if [ -z "$GIT_REPO" ]; then
    echo "❌ GIT_REPO environment variable is required"
    echo "   Example: export GIT_REPO='https://github.com/your-org/skeji'"
    exit 1
fi

if [ -z "$ARGOCD_AUTH_TOKEN" ]; then
    echo "❌ ARGOCD_AUTH_TOKEN environment variable is required"
    echo "   Example: export ARGOCD_AUTH_TOKEN='eyJhbGc...'"
    exit 1
fi

if [ -z "$SERVICES" ]; then
    echo "❌ SERVICES environment variable is required"
    echo "   Example: export SERVICES='business-units schedules bookings maestro'"
    exit 1
fi

IFS=' ' read -r -a SERVICES_ARRAY <<< "$SERVICES"
if [ ${#SERVICES_ARRAY[@]} -eq 0 ]; then
    echo "❌ SERVICES must contain at least one service name"
    exit 1
fi

if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl not found. Please install kubectl first."
    exit 1
fi

if ! command -v argocd &> /dev/null; then
    echo "📦 Installing argocd CLI..."
    curl -sSL -o /usr/local/bin/argocd https://github.com/argoproj/argo-cd/releases/download/${ARGOCD_VERSION}/argocd-linux-amd64
    chmod +x /usr/local/bin/argocd
    echo "✅ argocd CLI installed"
fi

echo "🔍 Validating ArgoCD installation..."
if ! kubectl get namespace $ARGOCD_NAMESPACE &> /dev/null; then
    echo "❌ ArgoCD namespace '$ARGOCD_NAMESPACE' not found. Ensure Hera platform layer is deployed."
    exit 1
fi

if ! kubectl get deployment argocd-server -n $ARGOCD_NAMESPACE &> /dev/null; then
    echo "❌ ArgoCD server not found in namespace '$ARGOCD_NAMESPACE'"
    exit 1
fi

echo "✅ ArgoCD installation validated"
echo ""

echo "🔐 Authenticating with ArgoCD..."
argocd login $ARGOCD_SERVER --auth-token="$ARGOCD_AUTH_TOKEN" --grpc-web --insecure
echo "✅ Authenticated"
echo ""

echo "📂 Applying AppProject..."
kubectl apply -f ${DEPLOYMENT_PATH}/argocd/project.yaml
echo "✅ AppProject applied"
echo ""

echo "🚀 Deploying microservices to ArgoCD..."
echo "   Services: ${SERVICES_ARRAY[*]}"
echo ""
for service in "${SERVICES_ARRAY[@]}"; do
    ARGOCD_APP_FILE="${DEPLOYMENT_PATH}/argocd/${service}.yaml"

    if [ ! -f "$ARGOCD_APP_FILE" ]; then
        echo "  ⚠️  Skipping $service - manifest not found: $ARGOCD_APP_FILE"
        continue
    fi

    echo "  📦 Applying $service application..."
    kubectl apply -f "$ARGOCD_APP_FILE"
    echo "  ✅ $service application applied"
done

echo ""
echo "⏳ Triggering initial sync for all services..."
for service in "${SERVICES_ARRAY[@]}"; do
    argocd app sync $service --grpc-web
done

echo ""
echo "✅ Deployment complete!"
echo ""
echo "📊 Application status:"
argocd app list --grpc-web | grep -E "NAME|${SERVICES_ARRAY[*]// /|}"
echo ""
echo "🔄 GitOps enabled - changes pushed to $GIT_REPO will auto-sync to cluster"
echo ""
