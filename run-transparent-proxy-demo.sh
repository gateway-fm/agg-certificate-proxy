#!/bin/bash
# Transparent Proxy Demo Script
# This script demonstrates the transparent proxy functionality

set -e

echo "========================================"
echo "AggLayer Certificate Proxy Demo"
echo "Transparent Forwarding Configuration"
echo "========================================"
echo

# Step 1: Generate proto files
echo "Step 1: Generating proto files..."
if ! make proto; then
    echo "❌ Failed to generate proto files"
    echo "Make sure you have protoc and the Go plugins installed"
    echo "Run: make proto-tools"
    exit 1
fi
echo "✅ Proto files generated"
echo

# Step 2: Build the proxy
echo "Step 2: Building proxy..."
make build
echo "✅ Proxy built"
echo

# Step 3: Run the tests
echo "Step 3: Running transparent proxy E2E test..."
cd tests
make transparent-proxy-e2e
cd ..

echo
echo "========================================"
echo "Demo Complete!"
echo "========================================"
echo
echo "To run the proxy with transparent forwarding manually:"
echo
echo "./proxy \\"
echo "  --grpc :50051 \\"
echo "  --http :8080 \\"
echo "  --backend-addr agglayer.example.com:50052 \\"
echo "  --aggsender-addr aggsender.example.com:50053 \\"
echo "  --delayed-chains 1,137 \\"
echo "  --delay 48h \\"
echo "  --kill-switch-api-key your-key \\"
echo "  --kill-restart-api-key your-key \\"
echo "  --data-key your-data-key"
echo
echo "This will:"
echo "- Intercept certificate submissions on chains 1,137 with 48h delay"
echo "- Forward all other gRPC requests to agglayer.example.com:50052"
echo "- Send delayed certificates to aggsender.example.com:50053" 