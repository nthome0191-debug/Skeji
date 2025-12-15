#!/bin/bash
set -e

ENV="${ENV:-local}"
ARGOCD_NAMESPACE="${ARGOCD_NAMESPACE:-argocd}"
ARGOCD_SERVER="${ARGOCD_SERVER:-argocd-server.argocd.svc.cluster.local}"
ARGOCD_AUTH_TOKEN="${ARGOCD_AUTH_TOKEN}"
APP_NAME="${APP_NAME:-skeji}"
DEPLOYMENT_PATH="${DEPLOYMENT_PATH:-deployment}"
ARGOCD_VERSION="${ARGOCD_VERSION:-v2.12.3}"

echo "🚀 Deploying $APP_NAME microservices to ArgoCD using ApplicationSet"
echo ""

if [ -z "$ARGOCD_AUTH_TOKEN" ]; then
    echo "❌ ARGOCD_AUTH_TOKEN environment variable is required"
    echo "   Example: export ARGOCD_AUTH_TOKEN='eyJhbGc...'"
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

echo "📦 Applying ApplicationSet..."
kubectl apply -f ${DEPLOYMENT_PATH}/argocd/applicationset-${ENV}.yaml
echo "✅ ApplicationSet applied"
echo ""

echo "⏳ Waiting for ApplicationSet to generate Applications..."
sleep 5

echo ""
echo "✅ Deployment complete!"
echo ""
echo "📊 Application status:"
argocd app list --grpc-web
echo ""
echo "🔄 GitOps enabled - ApplicationSet will auto-generate and sync applications"
echo ""
