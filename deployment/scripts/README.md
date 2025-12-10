# Deployment Scripts

## setup-argocd.sh

Production-ready script for deploying Skeji microservices to ArgoCD.

Designed to run in CI/CD pipelines after Hera infrastructure deployment.

### Prerequisites

- Kubernetes cluster with ArgoCD installed (via Hera platform layer)
- `kubectl` configured with cluster access
- ArgoCD authentication token injected as secret

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `ARGOCD_AUTH_TOKEN` | **Yes** | - | ArgoCD authentication token |
| `APP_NAME` | No | `skeji` | Application name |
| `DEPLOYMENT_PATH` | No | `deployment` | Path to deployment configs in repo |
| `ARGOCD_NAMESPACE` | No | `argocd` | ArgoCD namespace |
| `ARGOCD_SERVER` | No | `argocd-server.argocd.svc.cluster.local` | ArgoCD server address |
| `ARGOCD_VERSION` | No | `v2.12.3` | ArgoCD CLI version to install |

### Usage in CI/CD

```yaml
deploy:
  stage: deploy
  script:
    - export ARGOCD_AUTH_TOKEN="${ARGOCD_TOKEN}"
    - ./deployment/scripts/setup-argocd.sh
```

### Local Usage

```bash
export ARGOCD_AUTH_TOKEN="your-token-here"
./deployment/scripts/setup-argocd.sh
```

### What it does

1. ✅ Validates prerequisites (kubectl, argocd CLI)
2. 🔍 Verifies ArgoCD is running (installed by Hera)
3. 🔐 Authenticates with ArgoCD using token
4. 📂 Applies AppProject from `deployment/argocd/project.yaml`
5. 📦 Applies ApplicationSet from `deployment/argocd/applicationset.yaml`
6. 🎯 ApplicationSet auto-generates 12 Applications (4 services × 3 environments)
7. ✅ GitOps auto-sync enabled for all generated Applications

### After running

ApplicationSet generates 12 Applications automatically:
- `business-units-dev`, `business-units-staging`, `business-units-prod`
- `schedules-dev`, `schedules-staging`, `schedules-prod`
- `bookings-dev`, `bookings-staging`, `bookings-prod`
- `maestro-dev`, `maestro-staging`, `maestro-prod`

Future changes are automatically synced:

```bash
git add .
git commit -m "Update service configuration"
git push
```

ArgoCD ApplicationSet detects changes and syncs to cluster.

### Adding New Service

1. Create chart: `deployment/charts/new-service/`
2. Create values: `deployment/values/{dev,staging,prod}/new-service.yaml`
3. Edit `deployment/argocd/applicationset.yaml` - add service to list
4. Push to Git - ApplicationSet auto-generates 3 new Applications

### Adding New Environment

1. Create values: `deployment/values/qa/*.yaml` for all services
2. Edit `deployment/argocd/applicationset.yaml` - add environment to list
3. Push to Git - ApplicationSet auto-generates 4 new Applications

### Get ArgoCD Token

```bash
kubectl create token argocd-server -n argocd --duration=876000h
```

### Check Application Status

```bash
argocd app list
argocd app get business-units
argocd app sync business-units
```
