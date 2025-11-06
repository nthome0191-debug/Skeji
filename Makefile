.PHONY: local-up mongo-up kind-up kind-down migrate business-units-up test-integration

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

business-units-up:
	@echo "🏢 Deploying Business Units service..."
	bash deployment/local/business-units/setup.sh
	@echo "✅ Business Units service deployed successfully."

test-integration:
	@echo "🧪 Running integration tests..."
	@echo "💡 Tip: Customize test config in .env.test"
	bash scripts/run-integration-tests.sh
