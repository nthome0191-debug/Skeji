# Skeji Deployment

Kubernetes deployment configurations for Skeji microservices using Helm and ArgoCD.

## Structure

```
deployment/
├── charts/          # Helm charts for each microservice
├── values/          # Environment-specific values files
├── argocd/          # ArgoCD Application and AppProject manifests
└── local/           # Local development environment setup
```

## Services

- **business-units** - Manages business registration and profiles
- **schedules** - Manages availability windows and time slots
- **bookings** - Handles booking creation and lifecycle
- **maestro** - Orchestration service

## Quick Start

### Deploy with ArgoCD

```bash
kubectl apply -f deployment/argocd/project.yaml
kubectl apply -f deployment/argocd/
```

### Deploy with Helm

```bash
helm install business-units deployment/charts/business-units \
  -f deployment/values/business-units-dev.yaml \
  -n apps --create-namespace

helm install schedules deployment/charts/schedules \
  -f deployment/values/schedules-dev.yaml \
  -n apps

helm install bookings deployment/charts/bookings \
  -f deployment/values/bookings-dev.yaml \
  -n apps

helm install maestro deployment/charts/maestro \
  -f deployment/values/maestro-dev.yaml \
  -n apps
```

## Configuration

All services share common configuration structure:

- **MongoDB**: `mongodb://mongo.mongo.svc.cluster.local:27017`
- **Port**: 8080
- **Health checks**: `/health` (liveness), `/ready` (readiness)

Environment-specific overrides in `values/*.yaml` files.

## Docker Images

Update image repository in values files to point to your registry:

```yaml
image:
  repository: your-project/service-name
  tag: "v1.0.0"
```

## Environments

Three environments are configured:

| Environment | Database | Log Level | Resources |
|------------|----------|-----------|-----------|
| **dev** | `skeji` | debug | 50m CPU / 64Mi RAM |
| **staging** | `skeji-staging` | info | 100m CPU / 128Mi RAM |
| **prod** | `skeji-prod` | warn | 200m CPU / 256Mi RAM |

Values files: `deployment/values/{env}/{service}.yaml`

## Scaling

### Add New Service

1. Create Helm chart: `deployment/charts/new-service/`
2. Create values files: `deployment/values/{dev,staging,prod}/new-service.yaml`
3. Edit `deployment/argocd/applicationset.yaml` - add to services list
4. Push to Git → ApplicationSet auto-generates 3 Applications

### Add New Environment

1. Create values directory: `deployment/values/qa/`
2. Create values for all services: `{service}.yaml`
3. Edit `deployment/argocd/applicationset.yaml` - add to environments list
4. Push to Git → ApplicationSet auto-generates 4 Applications

## ArgoCD Sync Policy

All applications use automated sync with:
- Auto-pruning enabled
- Self-healing enabled
- Retry backoff: 5s → 3m (max)
- Create namespace automatically
