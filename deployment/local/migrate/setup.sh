#!/usr/bin/env bash
set -e

NAMESPACE="mongo"
IMAGE_NAME="skeji-migrate:latest"
DOCKERFILE="build/docker/migrate.Dockerfile"
JOB_FILE="deployment/local/migrate/migrate-mongo-cronjob.yaml"

docker build -t $IMAGE_NAME -f $DOCKERFILE .
kind load docker-image $IMAGE_NAME --name skeji-local
kubectl apply -n $NAMESPACE -f $JOB_FILE
