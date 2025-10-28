.PHONY: local-up mongo-up kind-up

local-up:
	bash deployment/local/scripts/local_env_up.sh

kind-up:
	bash deployment/local/kind/setup.sh

mongo-up:
	bash deployment/local/mongo/setup.sh
