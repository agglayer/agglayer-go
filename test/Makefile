.PHONY: run
run: stop ## Run all the services needed to perform an E2E test
	docker compose -f docker-compose.yaml up -d l1 zkevm-prover zkevm-node
	docker compose -f docker-compose.yaml up -d --build beethoven

.PHONY: stop
stop: ## Stop all the services needed to perform an E2E test
	docker compose -f docker-compose.yaml down
