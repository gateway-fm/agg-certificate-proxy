package certificate

import (
	"context"
	"fmt"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TransparentProxy implements a gRPC proxy that forwards all requests
// except for certificate submissions which are handled by our custom logic
type TransparentProxy struct {
	backendAddr string
	backendConn *grpc.ClientConn
}

// NewTransparentProxy creates a new transparent proxy
func NewTransparentProxy(backendAddr string) (*TransparentProxy, error) {
	if backendAddr == "" {
		return nil, fmt.Errorf("backend address cannot be empty")
	}

	// Create connection to backend agglayer
	conn, err := grpc.NewClient(backendAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(50*1024*1024), // 50MB
			grpc.MaxCallSendMsgSize(50*1024*1024), // 50MB
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to backend at %s: %w", backendAddr, err)
	}

	return &TransparentProxy{
		backendAddr: backendAddr,
		backendConn: conn,
	}, nil
}

// Close closes the backend connection
func (p *TransparentProxy) Close() error {
	if p.backendConn != nil {
		return p.backendConn.Close()
	}
	return nil
}

// TransparentUnaryHandler creates a UnaryServerInterceptor that forwards requests
func (p *TransparentProxy) TransparentUnaryHandler() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// This interceptor is only called for registered services (certificate submission)
		// All other services are handled by the UnknownServiceHandler
		slog.Info("handling request", "method", info.FullMethod)
		return handler(ctx, req)
	}
}

// TransparentStreamHandler creates a StreamServerInterceptor that forwards streaming requests
func (p *TransparentProxy) TransparentStreamHandler() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// This interceptor is only called for registered services
		// All other services are handled by the UnknownServiceHandler
		slog.Info("handling stream request", "method", info.FullMethod)
		return handler(srv, ss)
	}
}

// UnknownServiceHandler returns a handler for unknown services using the transparent handler
func (p *TransparentProxy) UnknownServiceHandler() grpc.StreamHandler {
	// Create a director that always forwards to our backend
	director := DefaultDirector(p.backendConn)

	// Return the transparent handler that will forward all unknown service calls
	return TransparentHandler(director)
}
