# Maestro Deployment

This directory contains Kubernetes manifests for deploying the Maestro orchestration service to the local Kind cluster.

## Files

- **maestro-deployment.yaml**: Deployment configuration for Maestro pods
- **maestro-service.yaml**: Service definition for exposing Maestro within the cluster

## Configuration

### Environment Variables

- `MAESTRO_PORT`: Port the service listens on (default: 8090)
- `API_BASE_URL`: Base URL for backend API services (points to business-units service which acts as gateway)

### Resources

- **Requests**: 100m CPU, 128Mi memory
- **Limits**: 500m CPU, 512Mi memory
- **Replicas**: 2 (for high availability)

### Health Checks

- **Readiness Probe**: `GET /api/v1/maestro/health` on port 8090
- **Liveness Probe**: `GET /api/v1/maestro/health` on port 8090

## Deployment

### Via Local Environment Script

The easiest way to deploy Maestro along with all other services:

```bash
make local-up
```

This will:
1. Start Kind cluster
2. Deploy Kafka
3. Deploy MongoDB
4. Run migrations
5. Deploy all services including Maestro

### Via Individual Deployment

To deploy only Maestro:

```bash
bash deployment/local/scripts/applications_setup.sh maestro
```

This will:
1. Build the Docker image from `build/docker/maestro.Dockerfile`
2. Load the image into the Kind cluster
3. Apply Kubernetes manifests
4. Wait for successful rollout

### Run Locally (Development)

To run Maestro locally without Kubernetes:

```bash
make maestro-up
# or
API_BASE_URL=http://localhost:8080 go run cmd/maestro/main.go
```

## Service Access

### Within the Cluster

Other services can access Maestro at:
```
http://maestro.apps.svc.cluster.local
```

### From Host Machine

Port forward to access Maestro from your local machine:

```bash
kubectl port-forward -n apps svc/maestro 8090:80
```

Then access at:
```
http://localhost:8090/api/v1/maestro/health
```

## API Endpoints

- `POST /api/v1/maestro/execute` - Execute a flow
- `GET /api/v1/maestro/flows` - List available flows
- `GET /api/v1/maestro/health` - Health check

See `internal/maestro/api/README.md` for full API documentation.

## Troubleshooting

### Check Pod Status

```bash
kubectl get pods -n apps -l app=maestro
```

### View Logs

```bash
# Get logs from all maestro pods
kubectl logs -n apps -l app=maestro

# Follow logs from a specific pod
kubectl logs -n apps -f <pod-name>
```

### Restart Deployment

```bash
kubectl rollout restart deployment/maestro -n apps
```

### Check Service Connectivity

From within another pod:
```bash
kubectl exec -n apps <some-pod> -- wget -O- http://maestro.apps.svc.cluster.local/api/v1/maestro/health
```
