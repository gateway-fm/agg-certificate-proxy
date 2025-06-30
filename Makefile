# Directory containing proto files (from submodule)
PROTO_DIR_INTEROP=interop/proto
PROTO_DIR_NODE=../agglayer/proto
# Find all proto files in the interop proto directory
PROTO_FILES=\
	$(PROTO_DIR_INTEROP)/agglayer/interop/types/v1/aggchain.proto \
	$(PROTO_DIR_INTEROP)/agglayer/interop/types/v1/bridge_exit.proto \
	$(PROTO_DIR_INTEROP)/agglayer/interop/types/v1/bytes.proto \
	$(PROTO_DIR_INTEROP)/agglayer/interop/types/v1/claim.proto \
	$(PROTO_DIR_INTEROP)/agglayer/interop/types/v1/imported_bridge_exit.proto \
	$(PROTO_DIR_INTEROP)/agglayer/interop/types/v1/merkle_proof.proto \
	$(PROTO_DIR_NODE)/agglayer/node/v1/certificate_submission.proto \
	$(PROTO_DIR_NODE)/agglayer/node/types/v1/certificate.proto \
	$(PROTO_DIR_NODE)/agglayer/node/types/v1/certificate_id.proto

# Output directory for generated Go code
PROTO_OUT=./pkg/proto

# Ensure Go bin is in PATH for protoc plugins
GOBIN_PATH=$(shell go env GOPATH)/bin

# Default values
DB_PATH ?= certs.db
HTTP_PORT ?= 8080
GRPC_PORT ?= 50051
DELAYED_CHAINS ?= 1,137
DELAY ?= 48h
AGGSENDER_ADDR ?= localhost:50052

# Mappings for each proto file to the Go package path
PROTO_MAPPINGS=\
	--go_opt=Magglayer/interop/types/v1/aggchain.proto=github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/interop/types/v1 \
	--go_opt=Magglayer/interop/types/v1/bridge_exit.proto=github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/interop/types/v1 \
	--go_opt=Magglayer/interop/types/v1/bytes.proto=github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/interop/types/v1 \
	--go_opt=Magglayer/interop/types/v1/claim.proto=github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/interop/types/v1 \
	--go_opt=Magglayer/interop/types/v1/imported_bridge_exit.proto=github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/interop/types/v1 \
	--go_opt=Magglayer/interop/types/v1/merkle_proof.proto=github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/interop/types/v1 \
	--go-grpc_opt=Magglayer/interop/types/v1/aggchain.proto=github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/interop/types/v1 \
	--go-grpc_opt=Magglayer/interop/types/v1/bridge_exit.proto=github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/interop/types/v1 \
	--go-grpc_opt=Magglayer/interop/types/v1/bytes.proto=github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/interop/types/v1 \
	--go-grpc_opt=Magglayer/interop/types/v1/claim.proto=github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/interop/types/v1 \
	--go-grpc_opt=Magglayer/interop/types/v1/imported_bridge_exit.proto=github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/interop/types/v1 \
	--go-grpc_opt=Magglayer/interop/types/v1/merkle_proof.proto=github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/interop/types/v1 \
	--go_opt=Magglayer/node/v1/certificate_submission.proto=github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/v1 \
	--go-grpc_opt=Magglayer/node/v1/certificate_submission.proto=github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/v1 \
	--go_opt=Magglayer/node/types/v1/certificate.proto=github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/types/v1 \
	--go_opt=Magglayer/node/types/v1/certificate_id.proto=github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/types/v1 \
	--go-grpc_opt=Magglayer/node/types/v1/certificate.proto=github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/types/v1 \
	--go-grpc_opt=Magglayer/node/types/v1/certificate_id.proto=github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/types/v1

.PHONY: help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@echo '  help                Show this help message'
	@echo '  build               Build the proxy binary'
	@echo '  run                 Run the proxy with default settings'
	@echo '  run-custom          Run the proxy with custom delayed chains (use DELAYED_CHAINS=x,y,z)'
	@echo '  run-no-delay        Run the proxy with no delayed chains (all pass through)'
	@echo '  proto               Generate Go code from proto files'
	@echo '  proto-tools         Install required protobuf tools'
	@echo '  clean               Remove build artifacts and database'
	@echo '  test                Run tests'
	@echo '  update-submodules   Update git submodules'
	@echo ''
	@echo 'Configuration:'
	@echo '  DB_PATH=$(DB_PATH)'
	@echo '  HTTP_PORT=$(HTTP_PORT)'
	@echo '  GRPC_PORT=$(GRPC_PORT)'
	@echo '  DELAYED_CHAINS=$(DELAYED_CHAINS)'
	@echo '  DELAY=$(DELAY)'
	@echo '  AGGSENDER_ADDR=$(AGGSENDER_ADDR)'
	@echo ''
	@echo 'Examples:'
	@echo '  make run-custom DELAYED_CHAINS=1,10,137 DELAY=24h'
	@echo '  make run HTTP_PORT=8081 GRPC_PORT=50052'
	@echo '  make run DELAY=10m  # 10 minutes delay'
	@echo '  make run DELAY=1h30m  # 1.5 hours delay'
	@echo '  make run AGGSENDER_ADDR=localhost:50053  # Different aggsender port'

.PHONY: proto-tools
proto-tools:
	@command -v protoc-gen-go >/dev/null 2>&1 || { \
		echo "protoc-gen-go not found. Installing..."; \
		go install google.golang.org/protobuf/cmd/protoc-gen-go@latest; \
	}
	@command -v protoc-gen-go-grpc >/dev/null 2>&1 || { \
		echo "protoc-gen-go-grpc not found. Installing..."; \
		go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest; \
	}

.PHONY: proto
proto: proto-tools
	@echo "Generating Go code from protos..."
	@mkdir -p $(PROTO_OUT)
	PATH="$(GOBIN_PATH):$$PATH" \
	protoc --go_out=$(PROTO_OUT) --go_opt=module=github.com/gateway-fm/agg-certificate-proxy/pkg/proto \
		--go-grpc_out=$(PROTO_OUT) --go-grpc_opt=module=github.com/gateway-fm/agg-certificate-proxy/pkg/proto \
		-I$(PROTO_DIR_INTEROP) -I$(PROTO_DIR_NODE) \
		$(PROTO_MAPPINGS) $(PROTO_FILES)

.PHONY: build
build: ## Build the proxy binary
	@echo "Building agg-certificate-proxy..."
	@go build -o proxy ./cmd/proxy

.PHONY: test
test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

.PHONY: clean
clean: ## Remove build artifacts and database
	@echo "Cleaning up..."
	@rm -f proxy
	@rm -f $(DB_PATH)
	@rm -f proxy.log
	@echo "Clean complete"

.PHONY: update-submodules
update-submodules: ## Update git submodules
	git submodule update --init --recursive 

.PHONY: run
run: build ## Run the proxy with default settings
	@echo "Starting proxy with default settings..."
	@echo "Delayed chains: $(DELAYED_CHAINS)"
	@echo "Delay: $(DELAY)"
	@echo "Aggsender address: $(AGGSENDER_ADDR)"
	./proxy -db=$(DB_PATH) -http=:$(HTTP_PORT) -grpc=:$(GRPC_PORT) -delayed-chains="$(DELAYED_CHAINS)" -delay=$(DELAY) -aggsender-addr=$(AGGSENDER_ADDR)

.PHONY: run-custom
run-custom: build ## Run the proxy with custom delayed chains (use DELAYED_CHAINS=x,y,z)
	@echo "Starting proxy with custom settings..."
	@echo "Delayed chains: $(DELAYED_CHAINS)"
	@echo "Delay: $(DELAY)"
	@echo "Aggsender address: $(AGGSENDER_ADDR)"
	./proxy -db=$(DB_PATH) -http=:$(HTTP_PORT) -grpc=:$(GRPC_PORT) -delayed-chains="$(DELAYED_CHAINS)" -delay=$(DELAY) -aggsender-addr=$(AGGSENDER_ADDR)

.PHONY: run-no-delay
run-no-delay: build ## Run the proxy with no delayed chains (all pass through)
	@echo "Starting proxy with no delayed chains (all pass through)..."
	@echo "Aggsender address: $(AGGSENDER_ADDR)"
	./proxy -db=$(DB_PATH) -http=:$(HTTP_PORT) -grpc=:$(GRPC_PORT) -delayed-chains="" -delay=$(DELAY) -aggsender-addr=$(AGGSENDER_ADDR)

.PHONY: run-background
run-background: build ## Run the proxy in the background
	@echo "Starting proxy in background..."
	@echo "Delayed chains: $(DELAYED_CHAINS)"
	@echo "Delay: $(DELAY)"
	@echo "Aggsender address: $(AGGSENDER_ADDR)"
	@./proxy -db=$(DB_PATH) -http=:$(HTTP_PORT) -grpc=:$(GRPC_PORT) -delayed-chains="$(DELAYED_CHAINS)" -delay=$(DELAY) -aggsender-addr=$(AGGSENDER_ADDR) > proxy.log 2>&1 &
	@echo "Proxy started in background. Check proxy.log for output."

.PHONY: stop
stop: ## Stop the background proxy
	@echo "Stopping proxy..."
	@pkill -f "./proxy" || true
	@echo "Proxy stopped"

.PHONY: logs
logs: ## Show proxy logs
	@tail -f proxy.log

.PHONY: status
status: ## Check proxy status
	@ps aux | grep "[.]\/proxy" || echo "Proxy is not running" 