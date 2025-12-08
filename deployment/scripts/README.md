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
| `GIT_REPO` | **Yes** | - | Git repository URL |
| `ARGOCD_AUTH_TOKEN` | **Yes** | - | ArgoCD authentication token |
| `SERVICES` | **Yes** | - | Space-separated list of services to deploy |
| `GIT_REVISION` | No | `main` | Git branch/tag/commit |
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
    - export GIT_REPO="https://github.com/your-org/skeji"
    - export GIT_REVISION="${CI_COMMIT_SHA}"
    - export ARGOCD_AUTH_TOKEN="${ARGOCD_TOKEN}"
    - export SERVICES="business-units schedules bookings maestro"
    - ./deployment/scripts/setup-argocd.sh
```

### Local Usage

```bash
export GIT_REPO="https://github.com/your-org/skeji"
export ARGOCD_AUTH_TOKEN="your-token-here"
export SERVICES="business-units schedules bookings maestro"
./deployment/scripts/setup-argocd.sh
```

### What it does

1. ✅ Validates prerequisites (kubectl, argocd CLI)
2. 🔍 Verifies ArgoCD is running (installed by Hera)
3. 🔐 Authenticates with ArgoCD using token
4. 📂 Applies AppProject from `deployment/argocd/project.yaml`
5. 🚀 Applies ArgoCD Applications from `deployment/argocd/*.yaml`
6. 🔄 Triggers initial sync for all services
7. ✅ GitOps auto-sync enabled (configured in manifests)

### After running

All services are deployed and GitOps is enabled. Future changes:

```bash
git add .
git commit -m "Update service configuration"
git push
```

ArgoCD will automatically sync changes to the cluster.

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
