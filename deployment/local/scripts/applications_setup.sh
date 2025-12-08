#!/usr/bin/env bash
set -euo pipefail

APP_NAMESPACE="apps"
KIND_CLUSTER="hera-local"

print_usage() {
    echo "❌ Missing app names."
    echo "Usage: $0 <app1> [<app2> ...]"
    echo "Example: $0 business-units schedules bookings"
    exit 1
}

ensure_namespace() {
    echo "🔹 Ensuring namespace '$APP_NAMESPACE' exists..."
    kubectl get ns "$APP_NAMESPACE" >/dev/null 2>&1 || kubectl create namespace "$APP_NAMESPACE"
}

build_image() {
    local app_name=$1
    local image_name="skeji/${app_name}:latest"
    local dockerfile="build/docker/${app_name}.Dockerfile"

    echo "📦 Building Docker image for $app_name..."
    if [[ ! -f "$dockerfile" ]]; then
        echo "❌ Missing Dockerfile: $dockerfile"
        exit 1
    fi

    docker build -t "$image_name" -f "$dockerfile" .
}

load_image_into_kind() {
    local app_name=$1
    local image_name="skeji/${app_name}:latest"

    echo "📥 Loading $app_name image into Kind cluster..."
    kind load docker-image "$image_name" --name "$KIND_CLUSTER"
}

apply_k8s_manifests() {
    local app_name=$1
    local deployment_file="deployment/local/${app_name}/${app_name}-deployment.yaml"
    local service_file="deployment/local/${app_name}/${app_name}-service.yaml"

    echo "📄 Applying manifests for $app_name..."
    for file in "$deployment_file" "$service_file"; do
        if [[ ! -f "$file" ]]; then
            echo "❌ Missing manifest: $file"
            exit 1
        fi
        kubectl apply -n "$APP_NAMESPACE" -f "$file"
    done
}

wait_for_rollout() {
    local app_name=$1
    echo "⏳ Waiting for rollout of $app_name..."
    if ! kubectl rollout status "deployment/${app_name}" -n "$APP_NAMESPACE" --timeout=180s; then
        echo "❌ Rollout failed or timed out for $app_name."
        kubectl get pods -n "$APP_NAMESPACE" -l "app=${app_name}"
        show_pod_logs "$app_name"
        exit 1
    fi
}

show_pod_logs() {
    local app_name=$1
    local pod_name
    pod_name=$(kubectl get pods -n "$APP_NAMESPACE" -l "app=${app_name}" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)

    if [[ -n "$pod_name" ]]; then
        echo "📜 Logs from pod $pod_name:"
        kubectl logs "$pod_name" -n "$APP_NAMESPACE" --tail=50 || true
    else
        echo "⚠️ No running pods found for $app_name."
    fi
}

deploy_app() {
    local app_name=$1
    echo ""
    echo "🚀 Deploying app: $app_name"
    echo "────────────────────────────────────────────"

    build_image "$app_name"
    load_image_into_kind "$app_name"
    apply_k8s_manifests "$app_name"
    wait_for_rollout "$app_name"
    show_pod_logs "$app_name"

    echo "✅ $app_name deployed successfully!"
    echo "────────────────────────────────────────────"
}

main() {
    if [[ $# -lt 1 ]]; then
        print_usage
    fi

    ensure_namespace

    for app in "$@"; do
        deploy_app "$app"
    done

    echo "🎉 All specified apps deployed successfully!"
}

main "$@"
