#!/usr/bin/env bash
set -e

if [ -f ".env.local" ]; then
    export $(grep -v '^#' .env.local | xargs)
    echo "Loaded local environment variables from .env.local"
fi

echo "=== Spinning up local environment ==="

bash deployment/local/kind/setup.sh
bash deployment/local/mongo/setup.sh
bash deployment/local/migrate/setup.sh
bash deployment/local/business-units/setup.sh

echo "✅ Local environment ready!"

skaffold dev -p kind