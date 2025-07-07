package certificate

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// transparentHandler handles the transparent forwarding of gRPC calls
type transparentHandler struct {
	director StreamDirector
}

// StreamDirector is a function that returns a connection to forward the request to
type StreamDirector func(ctx context.Context, fullMethodName string) (context.Context, *grpc.ClientConn, error)

// TransparentHandler creates a handler that transparently forwards all unknown gRPC calls
func TransparentHandler(director StreamDirector) grpc.StreamHandler {
	return (&transparentHandler{director: director}).handler
}

func (h *transparentHandler) handler(srv interface{}, serverStream grpc.ServerStream) error {
	// Get method from context
	fullMethodName, ok := grpc.Method(serverStream.Context())
	if !ok {
		return status.Errorf(codes.Internal, "failed to get method from context")
	}

	slog.Info("transparent handler forwarding request", "method", fullMethodName)

	// Get the outgoing context and connection
	outgoingCtx, clientConn, err := h.director(serverStream.Context(), fullMethodName)
	if err != nil {
		return err
	}

	// Create a client stream
	clientCtx, clientCancel := context.WithCancel(outgoingCtx)
	defer clientCancel()

	clientStream, err := grpc.NewClientStream(clientCtx, &grpc.StreamDesc{
		ServerStreams: true,
		ClientStreams: true,
	}, clientConn, fullMethodName)
	if err != nil {
		return err
	}

	// Relay the data between streams
	s2cErrChan := h.forwardServerToClient(serverStream, clientStream)
	c2sErrChan := h.forwardClientToServer(clientStream, serverStream)

	// Wait for one of the streams to finish
	for {
		select {
		case s2cErr := <-s2cErrChan:
			if s2cErr == io.EOF {
				// Client has finished sending, close the client send stream
				clientStream.CloseSend()
			} else {
				// Error from client send to server recv
				clientCancel()
				return status.Errorf(codes.Internal, "failed forwarding server to client: %v", s2cErr)
			}
		case c2sErr := <-c2sErrChan:
			// Server recv reports the status of the RPC
			serverStream.SetTrailer(clientStream.Trailer())
			if c2sErr != io.EOF {
				return c2sErr
			}
			return nil
		}
	}
}

func (h *transparentHandler) forwardClientToServer(src grpc.ClientStream, dst grpc.ServerStream) chan error {
	errChan := make(chan error, 1)
	go func() {
		f := &frame{}
		for {
			if err := src.RecvMsg(f); err != nil {
				errChan <- err
				return
			}
			if err := dst.SendMsg(f); err != nil {
				errChan <- err
				return
			}
		}
	}()
	return errChan
}

func (h *transparentHandler) forwardServerToClient(src grpc.ServerStream, dst grpc.ClientStream) chan error {
	errChan := make(chan error, 1)
	go func() {
		f := &frame{}
		for {
			if err := src.RecvMsg(f); err != nil {
				errChan <- err
				return
			}
			if err := dst.SendMsg(f); err != nil {
				errChan <- err
				return
			}
		}
	}()
	return errChan
}

// frame is used to read and write raw byte frames
type frame struct {
	payload []byte
}

func (f *frame) Reset() {
	f.payload = nil
}

func (f *frame) String() string {
	return string(f.payload)
}

func (f *frame) ProtoMessage() {}

func (f *frame) Marshal() ([]byte, error) {
	return f.payload, nil
}

func (f *frame) Unmarshal(data []byte) error {
	f.payload = data
	return nil
}

// CodecWithParent creates a codec that uses a parent codec for marshaling/unmarshaling
func CodecWithParent(parent grpc.Codec) grpc.Codec {
	return &rawCodec{parent}
}

type rawCodec struct {
	parentCodec grpc.Codec
}

func (c *rawCodec) Marshal(v interface{}) ([]byte, error) {
	if f, ok := v.(*frame); ok {
		return f.Marshal()
	}
	return c.parentCodec.Marshal(v)
}

func (c *rawCodec) Unmarshal(data []byte, v interface{}) error {
	if f, ok := v.(*frame); ok {
		return f.Unmarshal(data)
	}
	return c.parentCodec.Unmarshal(data, v)
}

func (c *rawCodec) String() string {
	return fmt.Sprintf("proxy>%s", c.parentCodec.String())
}

// DefaultDirector creates a simple director that forwards all calls to a single backend
func DefaultDirector(backendConn *grpc.ClientConn) StreamDirector {
	return func(ctx context.Context, fullMethodName string) (context.Context, *grpc.ClientConn, error) {
		md, _ := metadata.FromIncomingContext(ctx)
		outCtx := metadata.NewOutgoingContext(ctx, md.Copy())
		return outCtx, backendConn, nil
	}
}
