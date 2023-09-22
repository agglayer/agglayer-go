.PHONY: run
run: ## Run all the services needed to perform an E2E test
	docker-compose up -d zkevm-state-db zkevm-pool-db silencer-db l1
	sleep 1
	docker-compose up -d zkevm-prover silencer
	sleep 3
	docker-compose up -d zkevm-node

.PHONY: stop
stop: ## Stop all the services needed to perform an E2E test
	docker-compose down