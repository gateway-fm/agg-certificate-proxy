# Build stage
FROM golang:1.24 AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y \
    git \
    ca-certificates \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 go build \
    -ldflags="-w -s -extldflags=-static" \
    -o /app/proxy \
    ./cmd/proxy

# Runtime stage
FROM gcr.io/distroless/static-debian12:nonroot

# Copy the binary from builder stage
COPY --from=builder /app/proxy /app/proxy

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Set working directory
WORKDIR /app

# Expose the default ports
# 50051 for gRPC server
# 8080 for HTTP server
EXPOSE 50051 8080

# Run the application
ENTRYPOINT ["/app/proxy"]
