# AggLayer Certificate Proxy Tests

.PHONY: help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@echo '  help               Show this help message'
	@echo '  build-proxy        Build the proxy binary'
	@echo '  kill-switch        Run kill switch test'
	@echo '  passthrough        Run passthrough test'
	@echo '  graceful-shutdown  Run graceful shutdown test'
	@echo '  metrics            Run metrics test'
	@echo '  all                Run all tests'
	@echo '  clean              Clean up test artifacts'

.PHONY: build-proxy
build-proxy: ## Build the proxy binary
	@echo "Building proxy..."
	@cd .. && go build -o tests/proxy cmd/proxy/main.go

.PHONY: kill-switch
kill-switch: build-proxy ## Run kill switch test
	@echo "Running kill switch test..."
	@go run . -kill-switch

.PHONY: passthrough
passthrough: build-proxy ## Run passthrough test
	@echo "Running passthrough test..."
	@go run . -passthrough

.PHONY: graceful-shutdown
graceful-shutdown: build-proxy ## Run graceful shutdown test
	@echo "Running graceful shutdown test..."
	@go run . -graceful-shutdown

.PHONY: metrics
metrics: build-proxy ## Run metrics test
	@echo "Running metrics test..."
	@go run . -metrics

.PHONY: transparent-proxy-e2e
transparent-proxy-e2e: build-proxy ## Run transparent proxy test
	@echo "Running transparent proxy test..."
	@go run . -transparent-proxy

.PHONY: all
all: kill-switch passthrough graceful-shutdown metrics transparent-proxy ## Run all tests

.PHONY: clean
clean: ## Clean up test artifacts
	@echo "Cleaning up..."
	@rm -f kill-switch-test.db kill-switch-test.log mock-receiver.log
	@rm -f passthrough-test.db passthrough-test.log passthrough-receiver.log
	@rm -f graceful-shutdown-test.db graceful-shutdown-test.log
	@rm -f proxy receiver/receiver
	@echo "Clean complete" 