package certificate

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

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
		// Check if this is a certificate submission - let it pass through to our handler
		if strings.Contains(info.FullMethod, "CertificateSubmissionService/SubmitCertificate") {
			return handler(ctx, req)
		}

		// For all other requests, proxy to backend
		slog.Info("proxying unary request to backend", "method", info.FullMethod, "backend", p.backendAddr)

		// Since we're using the unknown service handler for non-certificate services,
		// this interceptor will only be called for registered services (certificate submission)
		// All other services will be handled by the UnknownServiceHandler
		return handler(ctx, req)
	}
}

// TransparentStreamHandler creates a StreamServerInterceptor that forwards streaming requests
func (p *TransparentProxy) TransparentStreamHandler() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Check if this is a certificate submission stream - let it pass through
		if strings.Contains(info.FullMethod, "CertificateSubmissionService") {
			return handler(srv, ss)
		}

		// For all other streams, they will be handled by UnknownServiceHandler
		slog.Info("stream request detected", "method", info.FullMethod, "backend", p.backendAddr)
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
