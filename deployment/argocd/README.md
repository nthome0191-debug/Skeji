# ArgoCD Configuration

This directory contains ArgoCD manifests for deploying Skeji microservices using ApplicationSet.

## Structure

```
argocd/
├── project.yaml          # AppProject definition
└── applicationset.yaml   # Generates all Applications
```

## ApplicationSet

The ApplicationSet uses a **matrix generator** to automatically create Applications for:

**Environments:**
- `dev`
- `staging`
- `prod`

**Services:**
- `business-units`
- `schedules`
- `bookings`
- `maestro`

**Result:** 12 Applications automatically generated (4 services × 3 environments)

## Application Naming

Applications are named: `{service}-{environment}`

Examples:
- `business-units-dev`
- `schedules-prod`
- `bookings-staging`

## Adding New Service

Edit `applicationset.yaml` and add to the services list:

```yaml
- list:
    elements:
    - service: business-units
    - service: schedules
    - service: bookings
    - service: maestro
    - service: new-service  # Add here
```

Then create:
- `deployment/charts/new-service/`
- `deployment/values/dev/new-service.yaml`
- `deployment/values/staging/new-service.yaml`
- `deployment/values/prod/new-service.yaml`

## Adding New Environment

Edit `applicationset.yaml` and add to the environments list:

```yaml
- list:
    elements:
    - env: dev
      cluster: https://kubernetes.default.svc
    - env: staging
      cluster: https://kubernetes.default.svc
    - env: prod
      cluster: https://kubernetes.default.svc
    - env: qa  # Add here
      cluster: https://kubernetes.default.svc
```

Then create `deployment/values/qa/*.yaml` files for all services.

## Multi-Cluster Deployment

To deploy different environments to different clusters, update the `cluster` field:

```yaml
- env: dev
  cluster: https://kubernetes.default.svc
- env: staging
  cluster: https://staging-cluster-api.example.com
- env: prod
  cluster: https://prod-cluster-api.example.com
```

## Deployment

Apply both files:

```bash
kubectl apply -f deployment/argocd/project.yaml
kubectl apply -f deployment/argocd/applicationset.yaml
```

Or use the setup script:

```bash
export ARGOCD_AUTH_TOKEN="your-token"
./deployment/scripts/setup-argocd.sh
```

## Verify

List all generated Applications:

```bash
argocd app list | grep skeji
```

View specific Application:

```bash
argocd app get business-units-dev
```

## Sync Policy

All Applications have automated sync enabled with:
- **Prune:** Delete resources removed from Git
- **Self-heal:** Revert manual changes to match Git
- **Create namespace:** Auto-create target namespace

## References

- [ArgoCD ApplicationSet Documentation](https://argo-cd.readthedocs.io/en/stable/user-guide/application-set/)
- [Matrix Generator](https://argo-cd.readthedocs.io/en/stable/operator-manual/applicationset/Generators-Matrix/)
