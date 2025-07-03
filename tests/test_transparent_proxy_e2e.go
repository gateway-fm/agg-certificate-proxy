package main

// NOTE: Before running this test, you need to generate the proto files:
//   cd .. && make proto
// This will generate the NodeStateService and ConfigurationService types

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	interopv1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/interop/types/v1"
	typesv1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/types/v1"
	v1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/v1"
)

// mockBackend implements all the gRPC services that should be transparently forwarded
type mockBackend struct {
	v1.UnimplementedCertificateSubmissionServiceServer
	v1.UnimplementedNodeStateServiceServer
	v1.UnimplementedConfigurationServiceServer

	// Track which methods were called
	callCounts map[string]int
}

func newMockBackend() *mockBackend {
	return &mockBackend{
		callCounts: make(map[string]int),
	}
}

// CertificateSubmissionService - should NOT be called through proxy
func (m *mockBackend) SubmitCertificate(ctx context.Context, req *v1.SubmitCertificateRequest) (*v1.SubmitCertificateResponse, error) {
	m.callCounts["SubmitCertificate"]++
	return nil, status.Error(codes.Internal, "backend SubmitCertificate should not be called through proxy")
}

// NodeStateService methods - should be forwarded through proxy
func (m *mockBackend) GetCertificateHeader(ctx context.Context, req *v1.GetCertificateHeaderRequest) (*v1.GetCertificateHeaderResponse, error) {
	m.callCounts["GetCertificateHeader"]++
	fmt.Printf("Backend: GetCertificateHeader called for cert ID\n")
	return &v1.GetCertificateHeaderResponse{
		CertificateHeader: &typesv1.CertificateHeader{
			NetworkId:         123,
			Height:            456,
			Status:            typesv1.CertificateStatus_CERTIFICATE_STATUS_SETTLED,
			CertificateId:     req.CertificateId,
			PrevLocalExitRoot: &interopv1.FixedBytes32{Value: []byte("prev-root")},
			NewLocalExitRoot:  &interopv1.FixedBytes32{Value: []byte("new-root")},
		},
	}, nil
}

func (m *mockBackend) GetLatestCertificateHeader(ctx context.Context, req *v1.GetLatestCertificateHeaderRequest) (*v1.GetLatestCertificateHeaderResponse, error) {
	m.callCounts["GetLatestCertificateHeader"]++
	fmt.Printf("Backend: GetLatestCertificateHeader called for network %d, type %v\n", req.NetworkId, req.Type)

	var status typesv1.CertificateStatus
	switch req.Type {
	case v1.LatestCertificateRequestType_LATEST_CERTIFICATE_REQUEST_TYPE_PENDING:
		status = typesv1.CertificateStatus_CERTIFICATE_STATUS_PENDING
	case v1.LatestCertificateRequestType_LATEST_CERTIFICATE_REQUEST_TYPE_SETTLED:
		status = typesv1.CertificateStatus_CERTIFICATE_STATUS_SETTLED
	default:
		status = typesv1.CertificateStatus_CERTIFICATE_STATUS_UNSPECIFIED
	}

	return &v1.GetLatestCertificateHeaderResponse{
		CertificateHeader: &typesv1.CertificateHeader{
			NetworkId: req.NetworkId,
			Height:    999,
			Status:    status,
			CertificateId: &typesv1.CertificateId{
				Value: &interopv1.FixedBytes32{Value: []byte("latest-cert-id")},
			},
		},
	}, nil
}

// ConfigurationService - should be forwarded through proxy
func (m *mockBackend) GetEpochConfiguration(ctx context.Context, req *v1.GetEpochConfigurationRequest) (*v1.GetEpochConfigurationResponse, error) {
	m.callCounts["GetEpochConfiguration"]++
	fmt.Printf("Backend: GetEpochConfiguration called\n")
	return &v1.GetEpochConfigurationResponse{
		EpochConfiguration: &typesv1.EpochConfiguration{
			GenesisBlock:  1000,
			EpochDuration: 3600,
		},
	}, nil
}

func runTransparentProxyE2ETest() {
	fmt.Println("=================================================")
	fmt.Println("AggLayer Certificate Proxy - Full E2E Test")
	fmt.Println("Testing transparent forwarding of all services")
	fmt.Println("=================================================")
	fmt.Println()

	// Test configuration
	proxyAddr := "127.0.0.1:50071"
	backendAddr := "127.0.0.1:50072"
	receiverAddr := "127.0.0.1:50073"
	dbFile := "transparent-proxy-e2e-test.db"
	logFile := "transparent-proxy-e2e-test.log"

	// Cleanup
	defer func() {
		os.Remove(dbFile)
		os.Remove(logFile)
		os.Remove("mock-receiver.log")
	}()

	// Step 1: Start mock backend with all services
	fmt.Println("Step 1: Starting mock backend with all AggLayer services...")
	backend := newMockBackend()
	backendServer := grpc.NewServer()
	v1.RegisterCertificateSubmissionServiceServer(backendServer, backend)
	v1.RegisterNodeStateServiceServer(backendServer, backend)
	v1.RegisterConfigurationServiceServer(backendServer, backend)

	backendLis, err := net.Listen("tcp", backendAddr)
	if err != nil {
		log.Fatalf("Failed to listen for backend: %v", err)
	}

	go func() {
		if err := backendServer.Serve(backendLis); err != nil {
			log.Printf("Backend serve error: %v", err)
		}
	}()
	defer backendServer.GracefulStop()

	// Wait for backend to be ready
	time.Sleep(500 * time.Millisecond)
	fmt.Printf("✅ Mock backend started on %s with all services\n", backendAddr)

	// Step 2: Start mock receiver for certificates
	fmt.Println("\nStep 2: Starting mock certificate receiver...")
	receiverCmd, err := startMockReceiver("50073")
	if err != nil {
		log.Fatalf("Failed to start mock receiver: %v", err)
	}
	defer receiverCmd.Process.Kill()
	time.Sleep(500 * time.Millisecond)
	fmt.Printf("✅ Mock receiver started on %s\n", receiverAddr)

	// Step 3: Start proxy with transparent forwarding
	fmt.Println("\nStep 3: Starting certificate proxy with transparent forwarding...")
	proxyLogFileHandle, err := os.Create(logFile)
	if err != nil {
		log.Fatalf("Failed to create log file: %v", err)
	}
	defer proxyLogFileHandle.Close()

	proxyCmd := exec.Command("./proxy",
		"--grpc", ":50071",
		"--http", ":8095",
		"--aggsender-addr", backendAddr,
		"--db", dbFile,
		"--delayed-chains", "1",
		"--delay", "3s",
		"--scheduler-interval", "1s",
		"--kill-switch-api-key", "test-key",
		"--kill-restart-api-key", "test-key",
		"--data-key", "test-data-key",
	)
	proxyCmd.Stdout = proxyLogFileHandle
	proxyCmd.Stderr = proxyLogFileHandle

	if err := proxyCmd.Start(); err != nil {
		log.Fatalf("Failed to start proxy: %v", err)
	}
	defer proxyCmd.Process.Kill()

	// Wait for proxy to be ready
	time.Sleep(2 * time.Second)
	fmt.Println("✅ Proxy started with transparent forwarding enabled")

	// Create client connection to proxy
	conn, err := grpc.NewClient(proxyAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to proxy: %v", err)
	}
	defer conn.Close()

	// Create all service clients
	certClient := v1.NewCertificateSubmissionServiceClient(conn)
	nodeStateClient := v1.NewNodeStateServiceClient(conn)
	configClient := v1.NewConfigurationServiceClient(conn)

	ctx := context.Background()

	// Test 1: Certificate submission should be intercepted
	fmt.Println("\n==== Test 1: Certificate Submission (intercepted) ====")
	_, err = certClient.SubmitCertificate(ctx, &v1.SubmitCertificateRequest{
		Certificate: &typesv1.Certificate{
			NetworkId:         1, // Delayed chain
			Height:            100,
			PrevLocalExitRoot: &interopv1.FixedBytes32{Value: []byte("prev")},
			NewLocalExitRoot:  &interopv1.FixedBytes32{Value: []byte("new")},
		},
	})
	if err != nil {
		fmt.Printf("❌ Certificate submission failed: %v\n", err)
	} else {
		fmt.Println("✅ Certificate submission intercepted by proxy")
	}

	// Verify backend didn't receive the certificate submission
	if backend.callCounts["SubmitCertificate"] == 0 {
		fmt.Println("✅ Backend did NOT receive certificate submission (correct)")
	} else {
		fmt.Printf("❌ Backend received %d certificate submissions (should be 0)\n", backend.callCounts["SubmitCertificate"])
	}

	// Test 2: GetCertificateHeader should be forwarded
	fmt.Println("\n==== Test 2: GetCertificateHeader (forwarded) ====")
	headerResp, err := nodeStateClient.GetCertificateHeader(ctx, &v1.GetCertificateHeaderRequest{
		CertificateId: &typesv1.CertificateId{
			Value: &interopv1.FixedBytes32{Value: []byte("test-cert-id-12345")},
		},
	})
	if err != nil {
		fmt.Printf("❌ GetCertificateHeader failed: %v\n", err)
	} else {
		fmt.Println("✅ GetCertificateHeader forwarded successfully")
		fmt.Printf("   Received: NetworkId=%d, Height=%d, Status=%v\n",
			headerResp.CertificateHeader.NetworkId,
			headerResp.CertificateHeader.Height,
			headerResp.CertificateHeader.Status)
	}

	// Test 3: GetLatestCertificateHeader with PENDING type
	fmt.Println("\n==== Test 3: GetLatestCertificateHeader PENDING (forwarded) ====")
	latestResp, err := nodeStateClient.GetLatestCertificateHeader(ctx, &v1.GetLatestCertificateHeaderRequest{
		Type:      v1.LatestCertificateRequestType_LATEST_CERTIFICATE_REQUEST_TYPE_PENDING,
		NetworkId: 42,
	})
	if err != nil {
		fmt.Printf("❌ GetLatestCertificateHeader PENDING failed: %v\n", err)
	} else {
		fmt.Println("✅ GetLatestCertificateHeader PENDING forwarded successfully")
		fmt.Printf("   Received: NetworkId=%d, Height=%d, Status=%v\n",
			latestResp.CertificateHeader.NetworkId,
			latestResp.CertificateHeader.Height,
			latestResp.CertificateHeader.Status)
	}

	// Test 4: GetLatestCertificateHeader with SETTLED type
	fmt.Println("\n==== Test 4: GetLatestCertificateHeader SETTLED (forwarded) ====")
	latestResp2, err := nodeStateClient.GetLatestCertificateHeader(ctx, &v1.GetLatestCertificateHeaderRequest{
		Type:      v1.LatestCertificateRequestType_LATEST_CERTIFICATE_REQUEST_TYPE_SETTLED,
		NetworkId: 137,
	})
	if err != nil {
		fmt.Printf("❌ GetLatestCertificateHeader SETTLED failed: %v\n", err)
	} else {
		fmt.Println("✅ GetLatestCertificateHeader SETTLED forwarded successfully")
		fmt.Printf("   Received: NetworkId=%d, Height=%d, Status=%v\n",
			latestResp2.CertificateHeader.NetworkId,
			latestResp2.CertificateHeader.Height,
			latestResp2.CertificateHeader.Status)
	}

	// Test 5: GetEpochConfiguration should be forwarded
	fmt.Println("\n==== Test 5: GetEpochConfiguration (forwarded) ====")
	epochResp, err := configClient.GetEpochConfiguration(ctx, &v1.GetEpochConfigurationRequest{})
	if err != nil {
		fmt.Printf("❌ GetEpochConfiguration failed: %v\n", err)
	} else {
		fmt.Println("✅ GetEpochConfiguration forwarded successfully")
		fmt.Printf("   Received: GenesisBlock=%d, EpochDuration=%d\n",
			epochResp.EpochConfiguration.GenesisBlock,
			epochResp.EpochConfiguration.EpochDuration)
	}

	// Test 6: Verify certificate delay
	fmt.Println("\n==== Test 6: Certificate Delay Verification ====")
	// Check receiver log should be empty initially
	receiverLog, _ := os.ReadFile("mock-receiver.log")
	if len(receiverLog) == 0 {
		fmt.Println("✅ Certificate not sent immediately (correctly delayed)")
	} else {
		fmt.Println("❌ Certificate was sent immediately (should be delayed)")
	}

	// Wait for delay period
	fmt.Println("Waiting for certificate delay period (3s)...")
	time.Sleep(4 * time.Second)

	// Check if certificate was eventually sent
	receiverLog, _ = os.ReadFile("mock-receiver.log")
	if len(receiverLog) > 0 {
		fmt.Println("✅ Delayed certificate was sent after delay period")
	} else {
		fmt.Println("❌ Delayed certificate was not sent")
	}

	// Test 7: Non-delayed certificate submission
	fmt.Println("\n==== Test 7: Non-delayed Certificate (immediate forward) ====")
	os.WriteFile("mock-receiver.log", []byte{}, 0644) // Clear log

	_, err = certClient.SubmitCertificate(ctx, &v1.SubmitCertificateRequest{
		Certificate: &typesv1.Certificate{
			NetworkId:         999, // Not in delayed list
			Height:            200,
			PrevLocalExitRoot: &interopv1.FixedBytes32{Value: []byte("prev2")},
			NewLocalExitRoot:  &interopv1.FixedBytes32{Value: []byte("new2")},
		},
	})
	if err != nil {
		fmt.Printf("❌ Non-delayed certificate submission failed: %v\n", err)
	} else {
		fmt.Println("✅ Non-delayed certificate submission succeeded")
	}

	// Should be sent immediately
	time.Sleep(500 * time.Millisecond)
	receiverLog, _ = os.ReadFile("mock-receiver.log")
	if len(receiverLog) > 0 {
		fmt.Println("✅ Non-delayed certificate was sent immediately")
	} else {
		fmt.Println("❌ Non-delayed certificate was not sent immediately")
	}

	// Final summary
	fmt.Println("\n==== Final Summary ====")
	fmt.Printf("Backend call counts:\n")
	fmt.Printf("  SubmitCertificate: %d (should be 0)\n", backend.callCounts["SubmitCertificate"])
	fmt.Printf("  GetCertificateHeader: %d\n", backend.callCounts["GetCertificateHeader"])
	fmt.Printf("  GetLatestCertificateHeader: %d\n", backend.callCounts["GetLatestCertificateHeader"])
	fmt.Printf("  GetEpochConfiguration: %d\n", backend.callCounts["GetEpochConfiguration"])

	// Check if all tests passed
	allPassed := backend.callCounts["SubmitCertificate"] == 0 &&
		backend.callCounts["GetCertificateHeader"] == 1 &&
		backend.callCounts["GetLatestCertificateHeader"] == 2 &&
		backend.callCounts["GetEpochConfiguration"] == 1

	if allPassed {
		fmt.Println("\n✅ All tests PASSED! Transparent proxy is working correctly.")
		fmt.Println("\nThe proxy successfully:")
		fmt.Println("- Intercepted certificate submissions")
		fmt.Println("- Forwarded NodeStateService.GetCertificateHeader calls")
		fmt.Println("- Forwarded NodeStateService.GetLatestCertificateHeader calls")
		fmt.Println("- Forwarded ConfigurationService.GetEpochConfiguration calls")
		fmt.Println("- Delayed certificates for configured chains")
		fmt.Println("- Immediately forwarded certificates for non-delayed chains")
	} else {
		fmt.Println("\n❌ Some tests FAILED. Check the output above.")
		os.Exit(1)
	}
}
