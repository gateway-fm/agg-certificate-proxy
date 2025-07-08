package main

import (
	"context"
	"crypto/rand"
	"io"
	"log"
	"log/slog"
	mrand "math/rand"
	"time"

	interopv1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/interop/types/v1"
	typesv1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/types/v1"
	v1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func sendRandomCertificate() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, "127.0.0.1:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := v1.NewCertificateSubmissionServiceClient(conn)

	exits := []*interopv1.BridgeExit{}

	for i := 0; i < mrand.Intn(5); i++ {
		exit := &interopv1.BridgeExit{
			LeafType: 321,
			TokenInfo: &interopv1.TokenInfo{
				OriginNetwork:      123,
				OriginTokenAddress: &interopv1.FixedBytes20{Value: []byte(randomAddress())},
			},
			DestNetwork: uint32(mrand.Intn(10)),
			DestAddress: &interopv1.FixedBytes20{Value: []byte(randomAddress())},
			Amount:      &interopv1.FixedBytes32{Value: randomAmount()},
		}
		exits = append(exits, exit)
	}

	// Create simple test certificate
	req := &v1.SubmitCertificateRequest{
		Certificate: &typesv1.Certificate{
			NetworkId:           1,
			Height:              1,
			PrevLocalExitRoot:   &interopv1.FixedBytes32{Value: make([]byte, 32)},
			NewLocalExitRoot:    &interopv1.FixedBytes32{Value: make([]byte, 32)},
			BridgeExits:         exits,
			ImportedBridgeExits: []*interopv1.ImportedBridgeExit{},
		},
	}

	_, err = client.SubmitCertificate(ctx, req)
	if err != nil {
		log.Fatalf("failed to submit certificate: %v", err)
	}

	slog.Info("sent random certificate")
}

func randomAddress() []byte {
	b := make([]byte, 20)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		log.Fatal(err)
	}
	return b
}

func randomAmount() []byte {
	b := make([]byte, 8)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		log.Fatal(err)
	}
	return b
}
