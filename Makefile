.PHONY: local-up mongo-up kind-up kind-down migrate

local-up:
	@echo "🚀 Spinning up full local environment..."
	bash deployment/local/scripts/local_env_up.sh

kind-up:
	@echo "🔹 Starting kind cluster..."
	bash deployment/local/kind/setup.sh

kind-down:
	@echo "🧹 Deleting kind cluster..."
	kind delete cluster --name skeji-local || true
	@echo "✅ Kind cluster deleted."

mongo-up:
	@echo "🔹 Deploying MongoDB..."
	bash deployment/local/mongo/setup.sh

migrate:
	@echo "🏗️ Running migrations..."
	bash deployment/local/migrate/setup.sh
	@echo "✅ Migration completed."
