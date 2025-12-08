.PHONY: \
	local-up \
	mongo-up \
	kind-up \
	kind-down \
	migrate \
	business-units-up \
	schedules-up \
	bookings-up \
	maestro-up \
	notifications-up \
	test-integration \
	test-integration-app-verbose \
	test-integration-business-units \
	test-integration-app-verbose-business-units \
	test-integration-schedules \
	test-integration-app-verbose-schedules
	test-integration-bookings \
	test-integration-app-verbose-bookings \
# 	test-integration-notifications \
# 	test-integration-app-verbose-notifications \
	swagger \
	swagger-schedules \
	swagger-businessunits \
	swagger-all

# === Local Environment =======================================================

local-up:
	@echo "🚀 Spinning up full local environment..."
	bash deployment/local/scripts/local_env_up.sh

kind-up:
	@echo "🔹 Starting kind cluster..."
	bash deployment/local/kind/setup.sh

kind-down:
	@echo "🧹 Deleting kind cluster..."
	bash test/scripts/setup-infra.sh --clean
	kind delete cluster --name hera-local || true
	@echo "✅ Kind cluster deleted."

mongo-up:
	@echo "🍃 Deploying MongoDB..."
	bash deployment/local/mongo/setup.sh

migrate:
	@echo "🏗️ Running migrations..."
	bash deployment/local/migrate/setup.sh
	@echo "✅ Migration completed."

# === Individual App Deployments ==============================================

business-units-up:
	@echo "🏢 Deploying Business Units service..."
	go run cmd/business-units/main.go
	@echo "✅ Business Units service deployed successfully."

schedules-up:
	@echo "📅 Deploying Schedules service..."
	go run cmd/schedules/main.go
	@echo "✅ Schedules service deployed successfully."

bookings-up:
	@echo "📘 Deploying Bookings service..."
	go run cmd/bookings/main.go
	@echo "✅ Bookings service deployed successfully."

maestro-up:
	@echo "🎭 Deploying Maestro service..."
	go run cmd/maestro/main.go
	@echo "✅ Maestro service deployed successfully."

# notifications-up:
# 	@echo "🔔 Deploying Notifications service..."
# 	bash deployment/local/notifications/setup.sh
# 	@echo "✅ Notifications service deployed successfully."
# === Integration Tests =======================================================

test-integration:
	@echo "🧪 Preparing test infrastructure..."
	bash test/scripts/setup-infra.sh --setup
	@echo "🧪 Running all integration tests sequentially..."
	make test-integration-business-units
	make test-integration-schedules
	make test-integration-bookings
# 	make test-integration-notifications
	bash test/scripts/setup-infra.sh --clean
	@echo "✅ All integration tests completed."

test-integration-app-verbose:
	@echo "🧪 Preparing test infrastructure (verbose mode)..."
	bash test/scripts/setup-infra.sh --setup
	@echo "🧪 Running all integration tests (verbose mode)..."
	make test-integration-app-verbose-business-units
	make test-integration-app-verbose-schedules
	make test-integration-app-verbose-bookings
# 	make test-integration-app-verbose-notifications
	bash test/scripts/setup-infra.sh --clean
	@echo "✅ All verbose integration tests completed."

# === Per-App Integration Tests ===============================================

test-integration-business-units:
	@echo "🧪 Running integration tests for Business Units app..."
	bash test/scripts/run-app-and-tests.sh business-units

test-integration-app-verbose-business-units:
	@echo "🧪 Running integration tests (verbose) for Business Units app..."
	bash test/scripts/run-app-and-tests.sh business-units --verbose

test-integration-schedules:
	@echo "🧪 Running integration tests for Schedules app..."
	bash test/scripts/run-app-and-tests.sh schedules

test-integration-app-verbose-schedules:
	@echo "🧪 Running integration tests (verbose) for Schedules app..."
	bash test/scripts/run-app-and-tests.sh schedules --verbose

test-integration-bookings:
	@echo "🧪 Running integration tests for Bookings app..."
	bash test/scripts/run-app-and-tests.sh bookings

test-integration-app-verbose-bookings:
	@echo "🧪 Running integration tests (verbose) for Bookings app..."
	bash test/scripts/run-app-and-tests.sh bookings --verbose

# test-integration-notifications:
# 	@echo "🧪 Running integration tests for Notifications app..."
# 	bash test/scripts/run-app-and-tests.sh notifications

# test-integration-app-verbose-notifications:
# 	@echo "🧪 Running integration tests (verbose) for Notifications app..."
# 	bash test/scripts/run-app-and-tests.sh notifications --verbose

# === Dev Tools =======================================================
swagger:
	cd internal/bookings && swag init \
		--generalInfo ../../cmd/bookings/main.go \
		--dir ./ \
		--output docs \
		--parseDependency \
		--parseInternal

swagger-schedules:
	cd internal/schedules && swag init \
		--generalInfo ../../cmd/schedules/main.go \
		--dir ./ \
		--output docs \
		--parseDependency \
		--parseInternal

swagger-businessunits:
	cd internal/businessunits && swag init \
		--generalInfo ../../cmd/business-units/main.go \
		--dir ./ \
		--output docs \
		--parseDependency \
		--parseInternal

swagger-all:
	@echo "📚 Generating Swagger documentation for all services..."
	make swagger
	make swagger-schedules
	make swagger-businessunits
	@echo "✅ All Swagger docs generated successfully."

