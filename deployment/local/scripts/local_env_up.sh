#!/usr/bin/env bash
set -e

if [ -f ".env.local" ]; then
    export $(grep -v '^#' .env.local | xargs)
    echo "Loaded local environment variables from .env.local"
fi

echo "=== Spinning up local environment ==="

# 1️⃣ Setup cluster
bash deployment/local/kind/setup.sh

# 2️⃣ Deploy MongoDB
bash deployment/local/mongo/setup.sh

# 3️⃣ Run migrations
bash deployment/local/migrate/setup.sh

echo "✅ Local environment ready!"
