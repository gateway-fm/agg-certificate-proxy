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
	"google.golang.org/grpc/credentials/insecure"

	"github.com/ethereum/go-ethereum/common"
	interopv1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/interop/types/v1"
	typesv1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/types/v1"
	v1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/v1"
)

func runSuspiciousTest() {
	fmt.Println("=================================================")
	fmt.Println("AggLayer Certificate Proxy - Suspicious certificates test")
	fmt.Println("Testing that suspicious certificates are locked and not sent to the AggSender")
	fmt.Println("=================================================")
	fmt.Println()

	// Test configuration
	proxyAddr := "127.0.0.1:50071"
	backendAddr := "127.0.0.1:50072"
	dbFile := "suspicious-test.db"
	logFile := "suspicious-test.log"

	// Clean up any stale processes before starting
	fmt.Println("Cleaning up any existing processes...")
	exec.Command("pkill", "-f", "mock_receiver").Run()
	exec.Command("pkill", "-f", "proxy").Run()
	time.Sleep(500 * time.Millisecond)

	os.Remove(dbFile)
	os.Remove(logFile)

	// Cleanup
	defer func() {
		fmt.Println("Cleaning up any existing processes...")
		exec.Command("pkill", "-f", "mock_receiver").Run()
		exec.Command("pkill", "-f", "proxy").Run()
		// os.Remove(dbFile)
		// os.Remove(logFile)
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

	// Step 2: Start proxy with transparent forwarding
	fmt.Println("\nStep 2: Starting certificate proxy with certificate override...")
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
		"--delay", "1s", // really long time :)
		"--scheduler-interval", "500ms",
		"--kill-switch-api-key", "test-key",
		"--kill-restart-api-key", "test-key",
		"--data-key", "test-data-key",
		"--certificate-override-key", "test-certificate-override-key",
		"--supsicious-value", "1000",
		"--token-values", "1111111111111111111111111111111111111111:1,2222222222222222222222222222222222222222:2",
	)
	proxyCmd.Stdout = proxyLogFileHandle
	proxyCmd.Stderr = proxyLogFileHandle

	const wellKnownOne = "1111111111111111111111111111111111111111"
	const wellKnownTwo = "2222222222222222222222222222222222222222"
	const wellKnownThree = "3333333333333333333333333333333333333333"

	if err := proxyCmd.Start(); err != nil {
		log.Fatalf("Failed to start proxy: %v", err)
	}
	defer proxyCmd.Process.Kill()

	// Wait for proxy to be ready
	time.Sleep(2 * time.Second)
	fmt.Println("✅ Proxy started with certificate override enabled")

	// Create client connection to proxy
	conn, err := grpc.NewClient(proxyAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to proxy: %v", err)
	}
	defer conn.Close()

	// Create all service clients
	certClient := v1.NewCertificateSubmissionServiceClient(conn)

	ctx := context.Background()

	// Test 1: Certificate submission should be intercepted
	fmt.Println("\n==== Test 1: No exits - goes through immediately")
	_, err = certClient.SubmitCertificate(ctx, &v1.SubmitCertificateRequest{
		Certificate: &typesv1.Certificate{
			NetworkId:           1, // Delayed chain
			Height:              100,
			BridgeExits:         []*interopv1.BridgeExit{},
			ImportedBridgeExits: []*interopv1.ImportedBridgeExit{},
			PrevLocalExitRoot:   &interopv1.FixedBytes32{Value: []byte("prev")},
			NewLocalExitRoot:    &interopv1.FixedBytes32{Value: []byte("new")},
			Metadata:            &interopv1.FixedBytes32{Value: nil},
		},
	})
	if err != nil {
		fmt.Printf("❌ Certificate submission failed: %v\n", err)
	}

	if backend.callCounts["SubmitCertificate"] == 1 {
		fmt.Println("✅ Backend received immediately as expected")
	} else {
		fmt.Printf("❌ Backend received %d certificate submissions (should be 0)\n", backend.callCounts["SubmitCertificate"])
	}

	fmt.Println("\n==== Test 2: Unknown token and lower than suspicious value - locked away")
	_, err = certClient.SubmitCertificate(ctx, &v1.SubmitCertificateRequest{
		Certificate: &typesv1.Certificate{
			NetworkId:           1, // Delayed chain
			Height:              100,
			BridgeExits:         generateBridgeExits([]string{"0x3333333333333333333333333333333333333333"}, []uint64{100}),
			ImportedBridgeExits: []*interopv1.ImportedBridgeExit{},
			PrevLocalExitRoot:   &interopv1.FixedBytes32{Value: []byte("prev")},
			NewLocalExitRoot:    &interopv1.FixedBytes32{Value: []byte("new")},
			Metadata:            &interopv1.FixedBytes32{Value: nil},
		},
	})
	if err != nil {
		fmt.Printf("❌ Certificate submission failed: %v\n", err)
	}

	if backend.callCounts["SubmitCertificate"] == 1 {
		fmt.Println("✅ certificate held in storage")
	} else {
		fmt.Printf("❌ Backend received %d certificate submissions (should be 0)\n", backend.callCounts["SubmitCertificate"])
	}

	time.Sleep(2 * time.Second)

	if backend.callCounts["SubmitCertificate"] == 2 {
		fmt.Println("✅ certificate processed as expected")
	} else {
		fmt.Printf("❌ Backend received %d certificate submissions (should be 0)\n", backend.callCounts["SubmitCertificate"])
	}

	fmt.Println("\n==== Test 3: Known token and lower than suspicious value - straight through")
	_, err = certClient.SubmitCertificate(ctx, &v1.SubmitCertificateRequest{
		Certificate: &typesv1.Certificate{
			NetworkId:           1, // Delayed chain
			Height:              100,
			BridgeExits:         generateBridgeExits([]string{wellKnownOne}, []uint64{100}),
			ImportedBridgeExits: []*interopv1.ImportedBridgeExit{},
			PrevLocalExitRoot:   &interopv1.FixedBytes32{Value: []byte("prev")},
			NewLocalExitRoot:    &interopv1.FixedBytes32{Value: []byte("new")},
			Metadata:            &interopv1.FixedBytes32{Value: nil},
		},
	})
	if err != nil {
		fmt.Printf("❌ Certificate submission failed: %v\n", err)
	}
	time.Sleep(500 * time.Millisecond)

	if backend.callCounts["SubmitCertificate"] == 3 {
		fmt.Println("✅ certificate processed immediately as expected")
	} else {
		fmt.Printf("❌ Backend received %d certificate submissions (should be 3)\n", backend.callCounts["SubmitCertificate"])
	}

	fmt.Println("\n==== Test 4: Multiple known tokens and lower than suspicious value - straight through")
	_, err = certClient.SubmitCertificate(ctx, &v1.SubmitCertificateRequest{
		Certificate: &typesv1.Certificate{
			NetworkId:           1, // Delayed chain
			Height:              100,
			BridgeExits:         generateBridgeExits([]string{wellKnownOne, wellKnownTwo}, []uint64{100, 100}),
			ImportedBridgeExits: []*interopv1.ImportedBridgeExit{},
			PrevLocalExitRoot:   &interopv1.FixedBytes32{Value: []byte("prev")},
			NewLocalExitRoot:    &interopv1.FixedBytes32{Value: []byte("new")},
			Metadata:            &interopv1.FixedBytes32{Value: nil},
		},
	})
	if err != nil {
		fmt.Printf("❌ Certificate submission failed: %v\n", err)
	}
	time.Sleep(500 * time.Millisecond)

	if backend.callCounts["SubmitCertificate"] == 4 {
		fmt.Println("✅ certificate processed immediately as expected")
	} else {
		fmt.Printf("❌ Backend received %d certificate submissions (should be 3)\n", backend.callCounts["SubmitCertificate"])
	}

	fmt.Println("\n==== Test 4: Multiple known tokens and one unknown token - known tokens lower than limit - locked away")
	_, err = certClient.SubmitCertificate(ctx, &v1.SubmitCertificateRequest{
		Certificate: &typesv1.Certificate{
			NetworkId:           1, // Delayed chain
			Height:              100,
			BridgeExits:         generateBridgeExits([]string{wellKnownOne, wellKnownTwo, wellKnownThree}, []uint64{100, 100, 100}),
			ImportedBridgeExits: []*interopv1.ImportedBridgeExit{},
			PrevLocalExitRoot:   &interopv1.FixedBytes32{Value: []byte("prev")},
			NewLocalExitRoot:    &interopv1.FixedBytes32{Value: []byte("new")},
			Metadata:            &interopv1.FixedBytes32{Value: nil},
		},
	})
	if err != nil {
		fmt.Printf("❌ Certificate submission failed: %v\n", err)
	}
	time.Sleep(500 * time.Millisecond)

	if backend.callCounts["SubmitCertificate"] == 4 {
		fmt.Println("✅ certificate held in storage")
	} else {
		fmt.Printf("❌ Backend received %d certificate submissions (should be 4)\n", backend.callCounts["SubmitCertificate"])
	}

	time.Sleep(2 * time.Second)

	if backend.callCounts["SubmitCertificate"] == 5 {
		fmt.Println("✅ certificate processed as expected")
	} else {
		fmt.Printf("❌ Backend received %d certificate submissions (should be 6)\n", backend.callCounts["SubmitCertificate"])
	}

	fmt.Println("\n==== Test 5: Single known token over the limit - locked away")
	_, err = certClient.SubmitCertificate(ctx, &v1.SubmitCertificateRequest{
		Certificate: &typesv1.Certificate{
			NetworkId:           1, // Delayed chain
			Height:              100,
			BridgeExits:         generateBridgeExits([]string{wellKnownTwo}, []uint64{10000}),
			ImportedBridgeExits: []*interopv1.ImportedBridgeExit{},
			PrevLocalExitRoot:   &interopv1.FixedBytes32{Value: []byte("prev")},
			NewLocalExitRoot:    &interopv1.FixedBytes32{Value: []byte("new")},
			Metadata:            &interopv1.FixedBytes32{Value: nil},
		},
	})
	if err != nil {
		fmt.Printf("❌ Certificate submission failed: %v\n", err)
	}
	time.Sleep(500 * time.Millisecond)

	if backend.callCounts["SubmitCertificate"] == 5 {
		fmt.Println("✅ certificate held in storage")
	} else {
		fmt.Printf("❌ Backend received %d certificate submissions (should be 4)\n", backend.callCounts["SubmitCertificate"])
	}

	time.Sleep(2 * time.Second)

	if backend.callCounts["SubmitCertificate"] == 6 {
		fmt.Println("✅ certificate processed as expected")
	} else {
		fmt.Printf("❌ Backend received %d certificate submissions (should be 6)\n", backend.callCounts["SubmitCertificate"])
	}

	fmt.Println("\n==== Test 6: Known token - value is multipleied correctly - locked away")
	_, err = certClient.SubmitCertificate(ctx, &v1.SubmitCertificateRequest{
		Certificate: &typesv1.Certificate{
			NetworkId:           1, // Delayed chain
			Height:              100,
			BridgeExits:         generateBridgeExits([]string{wellKnownTwo}, []uint64{501}), // should multiply to 1002 and be locked away
			ImportedBridgeExits: []*interopv1.ImportedBridgeExit{},
			PrevLocalExitRoot:   &interopv1.FixedBytes32{Value: []byte("prev")},
			NewLocalExitRoot:    &interopv1.FixedBytes32{Value: []byte("new")},
			Metadata:            &interopv1.FixedBytes32{Value: nil},
		},
	})
	if err != nil {
		fmt.Printf("❌ Certificate submission failed: %v\n", err)
	}
	time.Sleep(500 * time.Millisecond)

	if backend.callCounts["SubmitCertificate"] == 6 {
		fmt.Println("✅ certificate held in storage")
	} else {
		fmt.Printf("❌ Backend received %d certificate submissions (should be 4)\n", backend.callCounts["SubmitCertificate"])
	}

	time.Sleep(2 * time.Second)

	if backend.callCounts["SubmitCertificate"] == 7 {
		fmt.Println("✅ certificate processed as expected")
	} else {
		fmt.Printf("❌ Backend received %d certificate submissions (should be 6)\n", backend.callCounts["SubmitCertificate"])
	}
}

func generateBridgeExits(addresses []string, amounts []uint64) []*interopv1.BridgeExit {
	var result []*interopv1.BridgeExit

	for idx, address := range addresses {
		amountBytes := uint64ToBytes(amounts[idx])
		addy := common.HexToAddress(address)
		result = append(result, &interopv1.BridgeExit{
			TokenInfo: &interopv1.TokenInfo{
				OriginNetwork:      1,
				OriginTokenAddress: &interopv1.FixedBytes20{Value: addy.Bytes()},
			},
			DestNetwork: 1,
			DestAddress: &interopv1.FixedBytes20{Value: []byte(addresses[idx])},
			Amount:      &interopv1.FixedBytes32{Value: amountBytes},
			Metadata:    &interopv1.FixedBytes32{Value: nil},
		})
	}

	return result
}
