#!/bin/bash
# Transparent Proxy Demo Script
# This script demonstrates the transparent proxy functionality

set -e

echo "======================================="
echo "Transparent Proxy Demo"
echo "======================================="
echo ""
echo "This demo shows how the proxy transparently forwards all gRPC requests"
echo "while intercepting and delaying certificate submissions."
echo ""

# Check if required dependencies are available
if ! command -v grpcurl &> /dev/null; then
    echo "Error: grpcurl is not installed. Install it with:"
    echo "  brew install grpcurl (macOS)"
    echo "  go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest"
    exit 1
fi

echo "Configuration:"
echo "- Proxy listening on :50051"
echo "- Backend AggLayer at agglayer.example.com:50052"
echo "- Delayed chains: 1, 137"
echo "- Delay duration: 48h"
echo ""

echo "Starting proxy with command:"
echo "go run cmd/proxy/main.go \\"
echo "  --grpc :50051 \\"
echo "  --http :8080 \\"
echo "  --aggsender-addr agglayer.example.com:50052 \\"
echo "  --delayed-chains 1,137 \\"
echo "  --delay 48h \\"
echo "  --scheduler-interval 30s \\"
echo "  --kill-switch-api-key your-kill-key \\"
echo "  --kill-restart-api-key your-restart-key \\"
echo "  --data-key your-data-key"
echo ""

echo "The proxy will:"
echo "- Intercept certificate submissions for chains 1,137 and delay them 48h"
echo "- Forward all other gRPC requests transparently to agglayer.example.com:50052"
echo "- Forward delayed certificates to the same backend after the delay period"
echo ""

echo "Example requests:"
echo ""
echo "1. Certificate submission (will be intercepted and delayed):"
echo "   grpcurl -plaintext localhost:50051 agglayer.CertificateSubmissionService/SubmitCertificate"
echo ""
echo "2. Other services (will be transparently forwarded):"
echo "   grpcurl -plaintext localhost:50051 agglayer.NodeStateService/GetCertificateHeader"
echo "   grpcurl -plaintext localhost:50051 agglayer.ConfigurationService/GetEpochConfiguration"
echo ""

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
echo "  --aggsender-addr agglayer.example.com:50052 \\"
echo "  --delayed-chains 1,137 \\"
echo "  --delay 48h \\"
echo "  --kill-switch-api-key your-key \\"
echo "  --kill-restart-api-key your-key \\"
echo "  --data-key your-data-key"
echo
echo "This will:"
echo "- Intercept certificate submissions on chains 1,137 with 48h delay"
echo "- Forward all other gRPC requests to agglayer.example.com:50052"
echo "- Forward delayed certificates to the same backend after the delay period" 