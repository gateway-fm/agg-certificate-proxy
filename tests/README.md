# AggLayer Certificate Proxy Tests

## Structure

- `main.go` - Test runner entry point
- `test_kill_switch.go` - Kill switch functionality tests
- `test_passthrough.go` - Passthrough functionality tests
- `test_graceful_shutdown.go` - Graceful shutdown tests
- `receiver/` - Mock gRPC receiver used by tests
  - `main.go` - Mock aggsender implementation

## Quick Start

```bash
# Run all tests
make all

# Run specific test
make kill-switch        # Test emergency stop functionality
make passthrough        # Test non-delayed certificate forwarding
make graceful-shutdown  # Test graceful shutdown behavior

# Clean up
make clean
```

## Tests

- **Kill Switch**: Verifies 3-call activation, persistence across restarts, and reactivation
- **Passthrough**: Verifies non-delayed chains (e.g., chain 10) forward immediately
- **Graceful Shutdown**: Verifies:
  - Health endpoint returns 503 during shutdown
  - Scheduler waits for running tasks to complete
  - HTTP server waits for active requests
  - Proper shutdown sequence is followed

## Troubleshooting

If port conflicts occur:
```bash
pkill -f proxy
lsof -i :50051
``` 