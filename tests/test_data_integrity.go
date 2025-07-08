package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"

	interopv1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/interop/types/v1"
	typesv1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/types/v1"
	v1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/v1"
)

// dataIntegrityReceiver captures the exact protobuf messages it receives
type dataIntegrityReceiver struct {
	v1.UnimplementedCertificateSubmissionServiceServer
	mu               sync.Mutex
	receivedMessages [][]byte
	receivedRequests []*v1.SubmitCertificateRequest
	logFile          *os.File
}

func newDataIntegrityReceiver() *dataIntegrityReceiver {
	return &dataIntegrityReceiver{
		receivedMessages: make([][]byte, 0),
		receivedRequests: make([]*v1.SubmitCertificateRequest, 0),
	}
}

func (r *dataIntegrityReceiver) SubmitCertificate(ctx context.Context, req *v1.SubmitCertificateRequest) (*v1.SubmitCertificateResponse, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Marshal the received request to bytes for comparison
	receivedBytes, err := proto.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal received request: %w", err)
	}

	r.receivedMessages = append(r.receivedMessages, receivedBytes)
	r.receivedRequests = append(r.receivedRequests, req)

	networkID := uint32(0)
	if req.Certificate != nil {
		networkID = req.Certificate.NetworkId
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf("[%s] RECEIVED CERTIFICATE for network %d (message %d bytes)\n",
		timestamp, networkID, len(receivedBytes))

	fmt.Print(msg)
	if r.logFile != nil {
		r.logFile.WriteString(msg)
		r.logFile.Sync()
	}

	return &v1.SubmitCertificateResponse{
		CertificateId: &typesv1.CertificateId{
			Value: &interopv1.FixedBytes32{Value: []byte("data-integrity-test-id")},
		},
	}, nil
}

func (r *dataIntegrityReceiver) getReceivedMessages() [][]byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([][]byte(nil), r.receivedMessages...)
}

func (r *dataIntegrityReceiver) getReceivedRequests() []*v1.SubmitCertificateRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]*v1.SubmitCertificateRequest(nil), r.receivedRequests...)
}

// createComplexCertificateRequest creates a certificate with complex data to test integrity
func createComplexCertificateRequest(networkID uint32, height uint64) *v1.SubmitCertificateRequest {
	// Create random data for various fields
	prevExitRoot := make([]byte, 32)
	newExitRoot := make([]byte, 32)
	metadata := make([]byte, 32)
	signature := make([]byte, 65)
	customData := make([]byte, 100)

	rand.Read(prevExitRoot)
	rand.Read(newExitRoot)
	rand.Read(metadata)
	rand.Read(signature)
	rand.Read(customData)

	// Create complex bridge exits
	bridgeExits := []*interopv1.BridgeExit{
		{
			LeafType: 1,
			TokenInfo: &interopv1.TokenInfo{
				OriginNetwork:      123,
				OriginTokenAddress: &interopv1.FixedBytes20{Value: []byte("token-address-123456")},
			},
			DestNetwork: 456,
			DestAddress: &interopv1.FixedBytes20{Value: []byte("dest-address-789012")},
			Amount:      &interopv1.FixedBytes32{Value: []byte("amount-value-1234567890123456")},
		},
		{
			LeafType: 2,
			TokenInfo: &interopv1.TokenInfo{
				OriginNetwork:      789,
				OriginTokenAddress: &interopv1.FixedBytes20{Value: []byte("token-address-abcdef")},
			},
			DestNetwork: 101112,
			DestAddress: &interopv1.FixedBytes20{Value: []byte("dest-address-fedcba")},
			Amount:      &interopv1.FixedBytes32{Value: []byte("amount-value-fedcba0987654321")},
		},
	}

	// Create complex imported bridge exits
	importedBridgeExits := []*interopv1.ImportedBridgeExit{
		{
			BridgeExit: &interopv1.BridgeExit{
				LeafType: 3,
				TokenInfo: &interopv1.TokenInfo{
					OriginNetwork:      999,
					OriginTokenAddress: &interopv1.FixedBytes20{Value: []byte("imported-token-12345")},
				},
				DestNetwork: networkID,
				DestAddress: &interopv1.FixedBytes20{Value: []byte("imported-dest-67890")},
				Amount:      &interopv1.FixedBytes32{Value: []byte("imported-amount-0011223344556677")},
			},
			GlobalIndex: &interopv1.FixedBytes32{Value: []byte("global-index-8899aabbccddeeff")},
		},
	}

	// Create complex aggchain data
	aggchainData := &interopv1.AggchainData{
		Data: &interopv1.AggchainData_Signature{
			Signature: &interopv1.FixedBytes65{Value: signature},
		},
	}

	return &v1.SubmitCertificateRequest{
		Certificate: &typesv1.Certificate{
			NetworkId:           networkID,
			Height:              height,
			PrevLocalExitRoot:   &interopv1.FixedBytes32{Value: prevExitRoot},
			NewLocalExitRoot:    &interopv1.FixedBytes32{Value: newExitRoot},
			BridgeExits:         bridgeExits,
			ImportedBridgeExits: importedBridgeExits,
			Metadata:            &interopv1.FixedBytes32{Value: metadata},
			AggchainData:        aggchainData,
			CustomChainData:     customData,
			L1InfoTreeLeafCount: &[]uint32{12345}[0],
		},
	}
}

func runDataIntegrityTest() {
	fmt.Println("========================================================")
	fmt.Println("AggLayer Certificate Proxy - Data Integrity Test")
	fmt.Println("Testing that messages are preserved exactly when forwarded")
	fmt.Println("========================================================")
	fmt.Println()

	// Test configuration
	proxyAddr := "127.0.0.1:50081"
	receiverAddr := "127.0.0.1:50082"
	dbFile := "data-integrity-test.db"
	logFile := "data-integrity-test.log"

	// Cleanup
	defer func() {
		os.Remove(dbFile)
		os.Remove(logFile)
	}()

	// Step 1: Start data integrity receiver
	fmt.Println("Step 1: Starting data integrity receiver...")
	receiver := newDataIntegrityReceiver()

	// Open log file
	logFileHandle, err := os.Create("receiver-" + logFile)
	if err != nil {
		log.Fatalf("Failed to create receiver log file: %v", err)
	}
	defer logFileHandle.Close()
	receiver.logFile = logFileHandle

	receiverServer := grpc.NewServer()
	v1.RegisterCertificateSubmissionServiceServer(receiverServer, receiver)

	receiverLis, err := net.Listen("tcp", receiverAddr)
	if err != nil {
		log.Fatalf("Failed to listen for receiver: %v", err)
	}

	go func() {
		if err := receiverServer.Serve(receiverLis); err != nil {
			log.Printf("Receiver serve error: %v", err)
		}
	}()
	defer receiverServer.GracefulStop()

	// Wait for receiver to be ready
	time.Sleep(500 * time.Millisecond)
	fmt.Printf("✅ Data integrity receiver started on %s\n", receiverAddr)

	// Step 2: Start proxy
	fmt.Println("\nStep 2: Starting certificate proxy...")
	proxyLogFileHandle, err := os.Create("proxy-" + logFile)
	if err != nil {
		log.Fatalf("Failed to create proxy log file: %v", err)
	}
	defer proxyLogFileHandle.Close()

	proxyCmd := exec.Command("./proxy",
		"--grpc", ":50081",
		"--http", ":8096",
		"--aggsender-addr", receiverAddr,
		"--db", dbFile,
		"--delayed-chains", "1",
		"--delay", "2s",
		"--scheduler-interval", "300ms",
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
	fmt.Println("✅ Proxy started")

	// Create client connection to proxy
	conn, err := grpc.NewClient(proxyAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to proxy: %v", err)
	}
	defer conn.Close()

	client := v1.NewCertificateSubmissionServiceClient(conn)
	ctx := context.Background()

	// Test 1: Immediate forwarding (non-delayed chain)
	fmt.Println("\n==== Test 1: Immediate Forwarding (Non-Delayed Chain) ====")
	originalReq1 := createComplexCertificateRequest(999, 100) // Non-delayed chain

	// Marshal original request for comparison
	originalBytes1, err := proto.Marshal(originalReq1)
	if err != nil {
		log.Fatalf("Failed to marshal original request: %v", err)
	}

	fmt.Printf("Sending certificate with %d bytes of data...\n", len(originalBytes1))

	_, err = client.SubmitCertificate(ctx, originalReq1)
	if err != nil {
		log.Fatalf("Failed to submit certificate: %v", err)
	}

	// Wait a bit for processing
	time.Sleep(1 * time.Second)

	// Check if message was received
	receivedMessages := receiver.getReceivedMessages()
	if len(receivedMessages) != 1 {
		log.Fatalf("Expected 1 message, got %d", len(receivedMessages))
	}

	fmt.Printf("✅ Message received with %d bytes\n", len(receivedMessages[0]))

	// Compare byte-for-byte
	if !bytes.Equal(originalBytes1, receivedMessages[0]) {
		fmt.Printf("❌ Message integrity FAILED!\n")
		fmt.Printf("Original: %d bytes\n", len(originalBytes1))
		fmt.Printf("Received: %d bytes\n", len(receivedMessages[0]))

		// Find first difference
		minLen := len(originalBytes1)
		if len(receivedMessages[0]) < minLen {
			minLen = len(receivedMessages[0])
		}

		for i := 0; i < minLen; i++ {
			if originalBytes1[i] != receivedMessages[0][i] {
				fmt.Printf("First difference at byte %d: original=0x%02x, received=0x%02x\n",
					i, originalBytes1[i], receivedMessages[0][i])
				break
			}
		}

		os.Exit(1)
	} else {
		fmt.Println("✅ Message integrity preserved - exact byte match!")
	}

	// Test 2: Delayed forwarding
	fmt.Println("\n==== Test 2: Delayed Forwarding ====")
	originalReq2 := createComplexCertificateRequest(1, 200) // Delayed chain

	// Marshal original request for comparison
	originalBytes2, err := proto.Marshal(originalReq2)
	if err != nil {
		log.Fatalf("Failed to marshal original request: %v", err)
	}

	fmt.Printf("Sending delayed certificate with %d bytes of data...\n", len(originalBytes2))

	_, err = client.SubmitCertificate(ctx, originalReq2)
	if err != nil {
		log.Fatalf("Failed to submit certificate: %v", err)
	}

	// Should not be received immediately
	time.Sleep(500 * time.Millisecond)
	receivedMessages = receiver.getReceivedMessages()
	if len(receivedMessages) != 1 {
		log.Fatalf("Expected 1 message (delayed certificate should not be sent yet), got %d", len(receivedMessages))
	}
	fmt.Println("✅ Delayed certificate not sent immediately")

	// Wait for delay period
	fmt.Println("Waiting for delay period...")
	time.Sleep(3 * time.Second)

	// Check if delayed message was received
	receivedMessages = receiver.getReceivedMessages()
	if len(receivedMessages) != 2 {
		log.Fatalf("Expected 2 messages after delay, got %d", len(receivedMessages))
	}

	fmt.Printf("✅ Delayed message received with %d bytes\n", len(receivedMessages[1]))

	// Compare byte-for-byte for delayed message
	if !bytes.Equal(originalBytes2, receivedMessages[1]) {
		fmt.Printf("❌ Delayed message integrity FAILED!\n")
		fmt.Printf("Original: %d bytes\n", len(originalBytes2))
		fmt.Printf("Received: %d bytes\n", len(receivedMessages[1]))

		// Find first difference
		minLen := len(originalBytes2)
		if len(receivedMessages[1]) < minLen {
			minLen = len(receivedMessages[1])
		}

		for i := 0; i < minLen; i++ {
			if originalBytes2[i] != receivedMessages[1][i] {
				fmt.Printf("First difference at byte %d: original=0x%02x, received=0x%02x\n",
					i, originalBytes2[i], receivedMessages[1][i])
				break
			}
		}

		os.Exit(1)
	} else {
		fmt.Println("✅ Delayed message integrity preserved - exact byte match!")
	}

	// Test 3: Verify field-by-field integrity
	fmt.Println("\n==== Test 3: Field-by-Field Integrity Verification ====")
	receivedRequests := receiver.getReceivedRequests()

	// Verify immediate message
	if !verifyFieldIntegrity(originalReq1, receivedRequests[0]) {
		fmt.Println("❌ Immediate message field integrity FAILED!")
		os.Exit(1)
	}
	fmt.Println("✅ Immediate message field integrity verified")

	// Verify delayed message
	if !verifyFieldIntegrity(originalReq2, receivedRequests[1]) {
		fmt.Println("❌ Delayed message field integrity FAILED!")
		os.Exit(1)
	}
	fmt.Println("✅ Delayed message field integrity verified")

	// Final summary
	fmt.Println("\n==== Summary ====")
	fmt.Printf("✅ Total messages sent: 2\n")
	fmt.Printf("✅ Total messages received: %d\n", len(receivedMessages))
	fmt.Printf("✅ Immediate forwarding: PASSED\n")
	fmt.Printf("✅ Delayed forwarding: PASSED\n")
	fmt.Printf("✅ Byte-level integrity: PASSED\n")
	fmt.Printf("✅ Field-level integrity: PASSED\n")
	fmt.Println("\n🎉 All data integrity tests PASSED!")
	fmt.Println("The proxy preserves message integrity perfectly.")
}

func verifyFieldIntegrity(original, received *v1.SubmitCertificateRequest) bool {
	if original.Certificate.NetworkId != received.Certificate.NetworkId {
		fmt.Printf("NetworkId mismatch: %d vs %d\n", original.Certificate.NetworkId, received.Certificate.NetworkId)
		return false
	}

	if original.Certificate.Height != received.Certificate.Height {
		fmt.Printf("Height mismatch: %d vs %d\n", original.Certificate.Height, received.Certificate.Height)
		return false
	}

	if !bytes.Equal(original.Certificate.PrevLocalExitRoot.Value, received.Certificate.PrevLocalExitRoot.Value) {
		fmt.Printf("PrevLocalExitRoot mismatch\n")
		return false
	}

	if !bytes.Equal(original.Certificate.NewLocalExitRoot.Value, received.Certificate.NewLocalExitRoot.Value) {
		fmt.Printf("NewLocalExitRoot mismatch\n")
		return false
	}

	if len(original.Certificate.BridgeExits) != len(received.Certificate.BridgeExits) {
		fmt.Printf("BridgeExits length mismatch: %d vs %d\n", len(original.Certificate.BridgeExits), len(received.Certificate.BridgeExits))
		return false
	}

	if len(original.Certificate.ImportedBridgeExits) != len(received.Certificate.ImportedBridgeExits) {
		fmt.Printf("ImportedBridgeExits length mismatch: %d vs %d\n", len(original.Certificate.ImportedBridgeExits), len(received.Certificate.ImportedBridgeExits))
		return false
	}

	if !bytes.Equal(original.Certificate.Metadata.Value, received.Certificate.Metadata.Value) {
		fmt.Printf("Metadata mismatch\n")
		return false
	}

	if !bytes.Equal(original.Certificate.CustomChainData, received.Certificate.CustomChainData) {
		fmt.Printf("CustomChainData mismatch\n")
		return false
	}

	if original.Certificate.L1InfoTreeLeafCount != nil && received.Certificate.L1InfoTreeLeafCount != nil {
		if *original.Certificate.L1InfoTreeLeafCount != *received.Certificate.L1InfoTreeLeafCount {
			fmt.Printf("L1InfoTreeLeafCount mismatch: %d vs %d\n", *original.Certificate.L1InfoTreeLeafCount, *received.Certificate.L1InfoTreeLeafCount)
			return false
		}
	}

	return true
}
