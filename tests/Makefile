# AggLayer Certificate Proxy Tests

.PHONY: help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@echo '  help            Show this help message'
	@echo '  build-proxy     Build the proxy binary'
	@echo '  kill-switch     Run kill switch test'
	@echo '  passthrough     Run passthrough test'
	@echo '  all             Run all tests'
	@echo '  clean           Clean up test artifacts'

.PHONY: build-proxy
build-proxy: ## Build the proxy binary
	@echo "Building proxy..."
	@cd .. && go build -o proxy cmd/proxy/main.go

.PHONY: kill-switch
kill-switch: build-proxy ## Run kill switch test
	@echo "Running kill switch test..."
	@cd .. && go run tests/*.go -kill-switch

.PHONY: passthrough
passthrough: build-proxy ## Run passthrough test
	@echo "Running passthrough test..."
	@cd .. && go run tests/*.go -passthrough

.PHONY: all
all: kill-switch passthrough ## Run all tests

.PHONY: clean
clean: ## Clean up test artifacts
	@echo "Cleaning up..."
	@cd .. && rm -f kill-switch-test.db kill-switch-test.log mock-receiver.log mock_receiver
	@cd .. && rm -f passthrough-test.db passthrough-test.log passthrough-receiver.log simple_receiver
	@cd .. && rm -f proxy
	@echo "Clean complete" 