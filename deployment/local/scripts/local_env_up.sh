#!/usr/bin/env bash
set -e

echo "=== Spinning up local environment ==="

# 1️⃣ Setup cluster
bash deployment/local/kind/setup.sh

# 2️⃣ Deploy MongoDB
bash deployment/local/mongo/setup.sh

# 3️⃣ Run migrations
# go run deployment/local/mongo/migrate.go

echo "✅ Local environment ready!"
