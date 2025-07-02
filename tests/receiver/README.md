# Mock gRPC Receiver

This is a mock gRPC server that implements the AggLayer Certificate Submission Service. It's used by the integration tests to simulate an aggsender.

## Features

- Accepts certificates via gRPC
- Logs all received certificates with timestamps
- Configurable port and log file via command-line flags

## Usage

```bash
go run main.go -port 50052 -log mock-receiver.log
```

## Command-line Options

- `-port`: Port to listen on (default: 50052)
- `-log`: Path to log file (default: mock-receiver.log)

## Integration with Tests

The test files in the parent directory automatically build and run this receiver as needed. No manual intervention is required. 