package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	interopv1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/interop/types/v1"
	typesv1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/types/v1"
	v1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/v1"
)

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// runKillSwitchTest runs the comprehensive kill switch test
func runKillSwitchTest() {
	fmt.Println("===========================================")
	fmt.Println("AggLayer Certificate Proxy Kill Switch Test")
	fmt.Println("===========================================")
	fmt.Println()

	// Clean up any stale processes before starting
	fmt.Println("Cleaning up any existing processes...")
	exec.Command("pkill", "-f", "mock_receiver").Run()
	exec.Command("pkill", "-f", "proxy").Run()
	time.Sleep(500 * time.Millisecond)

	// Configuration - use 127.0.0.1 to force IPv4
	proxyAddr := "127.0.0.1:50051"
	httpAddr := "http://127.0.0.1:8080"
	mockReceiverPort := "50052"
	killKey := "test-kill-key"
	restartKey := "test-restart-key"
	dataKey := "test-data-key"
	certificateOverrideKey := "test-certificate-override-key"
	dbFile := "kill-switch-test.db"
	logFile := "kill-switch-test.log"

	// Cleanup function
	cleanup := func() {
		fmt.Println("\nCleaning up...")
		os.Remove(dbFile)
		os.Remove(logFile)
		os.Remove("mock-receiver.log")
	}
	defer cleanup()

	// Start mock receiver
	fmt.Println("Step 1: Starting mock receiver...")
	mockCmd, err := startMockReceiver(mockReceiverPort)
	if err != nil {
		log.Fatalf("Failed to start mock receiver: %v", err)
	}
	defer mockCmd.Process.Kill()

	// Wait for mock receiver to be ready
	receiverReady := false
	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)
		// Check if receiver is actually listening
		conn, err := net.Dial("tcp", "127.0.0.1:"+mockReceiverPort)
		if err == nil {
			conn.Close()
			receiverReady = true
			fmt.Println("Mock receiver is ready!")
			break
		}
		fmt.Printf("  Waiting for receiver... (%d/10) - %v\n", i+1, err)
	}

	if !receiverReady {
		// Check if process is still running
		if mockCmd.Process != nil {
			if err := mockCmd.Process.Signal(syscall.Signal(0)); err != nil {
				fmt.Printf("Mock receiver process died: %v\n", err)
			}
		}
		log.Fatalf("Mock receiver failed to start on port %s", mockReceiverPort)
	}

	// Check if proxy exists
	proxyPath := "./proxy"
	if _, err := os.Stat(proxyPath); os.IsNotExist(err) {
		log.Fatalf("Proxy binary not found at %s. Please build it first: go build -o proxy cmd/proxy/main.go", proxyPath)
	}

	// Start proxy with test configuration
	fmt.Println("Step 2: Starting certificate proxy with short delays...")
	proxyCmd := exec.Command(proxyPath,
		"--db", dbFile,
		"--http", ":8080",
		"--grpc", ":50051",
		"--kill-switch-api-key", killKey,
		"--kill-restart-api-key", restartKey,
		"--data-key", dataKey,
		"--certificate-override-key", certificateOverrideKey,
		"--delay", "5s",
		"--scheduler-interval", "2s",
		"--aggsender-addr", "127.0.0.1:"+mockReceiverPort,
		"--delayed-chains", "1,137",
	)

	proxyLogFile, err := os.Create(logFile)
	if err != nil {
		log.Fatalf("Failed to create log file: %v", err)
	}
	defer proxyLogFile.Close()

	proxyCmd.Stdout = proxyLogFile
	proxyCmd.Stderr = proxyLogFile

	if err := proxyCmd.Start(); err != nil {
		log.Fatalf("Failed to start proxy: %v", err)
	}
	defer proxyCmd.Process.Kill()

	// Wait for proxy to start and verify it's ready
	fmt.Println("Waiting for proxy to start...")
	ready := false
	grpcReady := false
	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)

		// Check HTTP endpoint
		resp, err := http.Get(httpAddr + "/")
		if err == nil {
			resp.Body.Close()
			ready = true
		}

		// Check gRPC endpoint
		conn, err := grpc.Dial(proxyAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err == nil {
			conn.Close()
			grpcReady = true
		}

		if ready && grpcReady {
			fmt.Println("Proxy is ready!")
			break
		}
		fmt.Printf("  Waiting... (%d/10) HTTP:%v gRPC:%v\n", i+1, ready, grpcReady)
	}

	if !ready || !grpcReady {
		// Check the log file for errors
		logContent, _ := os.ReadFile(logFile)
		fmt.Printf("Proxy failed to start. Log content:\n%s\n", string(logContent))
		log.Fatalf("Proxy did not start within 10 seconds (HTTP ready: %v, gRPC ready: %v)", ready, grpcReady)
	}

	// Submit test certificates
	fmt.Println("Step 3: Submitting test certificates to delayed chains...")
	if err := submitTestCertificate(proxyAddr, 1, 100, 1000); err != nil {
		fmt.Printf("  ❌ Failed to submit certificate for chain 1: %v\n", err)
		// Show proxy log for debugging
		proxyLog, _ := os.ReadFile(logFile)
		fmt.Printf("\nProxy log:\n%s\n", string(proxyLog[max(0, len(proxyLog)-2000):]))
		log.Fatalf("Test failed: Could not submit certificate")
	}
	if err := submitTestCertificate(proxyAddr, 137, 200, 1000); err != nil {
		fmt.Printf("  ❌ Failed to submit certificate for chain 137: %v\n", err)
		log.Fatalf("Test failed: Could not submit certificate")
	}
	if err := submitTestCertificate(proxyAddr, 10, 300, 1000); err != nil {
		fmt.Printf("  ❌ Failed to submit certificate for chain 10: %v\n", err)
		log.Fatalf("Test failed: Could not submit certificate")
	}

	// Wait to verify non-delayed certificate goes through
	fmt.Println("Step 4: Waiting to verify non-delayed certificate...")
	time.Sleep(5 * time.Second) // Give more time for certificate to be processed

	// Check mock receiver log - should only have chain 10
	receiverLog, err := os.ReadFile("mock-receiver.log")
	if err != nil {
		fmt.Printf("❌ ERROR: Failed to read receiver log: %v\n", err)
	} else if len(receiverLog) == 0 {
		fmt.Println("❌ ERROR: No certificates received (expected chain 10)")
		// Let's check the proxy log for clues
		proxyLog, _ := os.ReadFile(logFile)
		fmt.Printf("\nProxy log tail:\n%s\n", string(proxyLog[max(0, len(proxyLog)-1000):]))
	} else {
		fmt.Println("✅ Non-delayed certificate (chain 10) sent immediately")
		fmt.Printf("   Receiver log: %s", string(receiverLog))
	}

	// Activate kill switch
	fmt.Println("\nStep 5: Activating kill switch (3 calls required)...")
	for i := 1; i <= 3; i++ {
		fmt.Printf("  Kill switch call %d/3...", i)
		resp, err := http.Post(httpAddr+"/kill?key="+killKey, "text/plain", nil)
		if err != nil {
			fmt.Printf(" ERROR: %v\n", err)
		} else {
			fmt.Printf(" Status: %s\n", resp.Status)
			resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Check scheduler status
	fmt.Println("\nStep 6: Verifying scheduler status...")
	if status := checkSchedulerStatus(httpAddr, dataKey); status {
		fmt.Println("❌ ERROR: Scheduler still active after kill switch")
	} else {
		fmt.Println("✅ Scheduler stopped successfully")
	}

	// Wait for delay period to pass
	fmt.Println("\nStep 7: Waiting for delay period (certificates should NOT be sent)...")
	time.Sleep(6 * time.Second)

	// Clear receiver log and check
	os.WriteFile("mock-receiver.log", []byte{}, 0644)
	time.Sleep(2 * time.Second)

	receiverLog, _ = os.ReadFile("mock-receiver.log")
	if len(receiverLog) > 0 {
		fmt.Println("❌ ERROR: Certificates were sent despite kill switch!")
	} else {
		fmt.Println("✅ Kill switch prevented certificate sending")
	}

	// Restart proxy to test persistence
	fmt.Println("\nStep 8: Restarting proxy to test kill switch persistence...")
	proxyCmd.Process.Kill()
	time.Sleep(1 * time.Second)

	// Start proxy again
	proxyCmd = exec.Command(proxyPath,
		"--db", dbFile,
		"--http", ":8080",
		"--grpc", ":50051",
		"--kill-switch-api-key", killKey,
		"--kill-restart-api-key", restartKey,
		"--data-key", dataKey,
		"--certificate-override-key", certificateOverrideKey,
		"--delay", "5s",
		"--scheduler-interval", "2s",
		"--aggsender-addr", "127.0.0.1:"+mockReceiverPort,
		"--delayed-chains", "1,137",
	)
	proxyCmd.Stdout = proxyLogFile
	proxyCmd.Stderr = proxyLogFile

	if err := proxyCmd.Start(); err != nil {
		log.Fatalf("Failed to restart proxy: %v", err)
	}
	defer proxyCmd.Process.Kill()

	// Wait for proxy to restart and verify it's ready
	fmt.Println("Waiting for proxy to restart...")
	ready = false
	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)
		resp, err := http.Get(httpAddr + "/")
		if err == nil {
			resp.Body.Close()
			ready = true
			fmt.Println("Proxy restarted successfully!")
			break
		}
		fmt.Printf("  Waiting... (%d/10)\n", i+1)
	}

	if !ready {
		logContent, _ := os.ReadFile(logFile)
		fmt.Printf("Proxy failed to restart. Log content:\n%s\n", string(logContent))
		log.Fatalf("Proxy did not restart within 10 seconds")
	}

	// Submit more certificates
	fmt.Println("Step 9: Submitting more certificates after restart...")
	if err := submitTestCertificate(proxyAddr, 1, 400, 1000); err != nil {
		fmt.Printf("  ❌ Failed to submit certificate for chain 1: %v\n", err)
		log.Fatalf("Test failed: Could not submit certificate after restart")
	}
	if err := submitTestCertificate(proxyAddr, 137, 500, 1000); err != nil {
		fmt.Printf("  ❌ Failed to submit certificate for chain 137: %v\n", err)
		log.Fatalf("Test failed: Could not submit certificate after restart")
	}

	// Wait and verify they're still blocked
	fmt.Println("Step 10: Waiting to verify kill switch persisted...")
	time.Sleep(6 * time.Second)

	os.WriteFile("mock-receiver.log", []byte{}, 0644)
	time.Sleep(2 * time.Second)

	receiverLog, _ = os.ReadFile("mock-receiver.log")
	if len(receiverLog) > 0 {
		fmt.Println("❌ ERROR: Kill switch did not persist across restart!")
	} else {
		fmt.Println("✅ Kill switch persisted across restart")
	}

	// Reactivate scheduler
	fmt.Println("\nStep 11: Reactivating scheduler (3 calls required)...")
	for i := 1; i <= 3; i++ {
		fmt.Printf("  Restart call %d/3...", i)
		resp, err := http.Post(httpAddr+"/restart?key="+restartKey, "text/plain", nil)
		if err != nil {
			fmt.Printf(" ERROR: %v\n", err)
		} else {
			fmt.Printf(" Status: %s\n", resp.Status)
			resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Verify scheduler is active
	if status := checkSchedulerStatus(httpAddr, dataKey); !status {
		fmt.Println("❌ ERROR: Scheduler still stopped after restart")
	} else {
		fmt.Println("✅ Scheduler reactivated successfully")
	}

	// Wait for certificates to be sent
	fmt.Println("\nStep 12: Waiting for accumulated certificates to be sent...")
	time.Sleep(5 * time.Second)

	receiverLog, _ = os.ReadFile("mock-receiver.log")
	if len(receiverLog) == 0 {
		fmt.Println("❌ ERROR: Certificates were not sent after reactivation")
	} else {
		fmt.Println("✅ Accumulated certificates sent after reactivation")
		fmt.Printf("   Received %d bytes of certificate data\n", len(receiverLog))
	}

	fmt.Println("\n===========================================")
	fmt.Println("Kill Switch Test Complete!")
	fmt.Println("===========================================")
}

// startMockReceiver starts a mock aggsender for testing
func startMockReceiver(port string) (*exec.Cmd, error) {
	// Build the receiver if it doesn't exist
	receiverBinary := "./receiver/receiver"
	if _, err := os.Stat(receiverBinary); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", receiverBinary, "./receiver/main.go")
		buildOut, err := buildCmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("failed to build mock receiver: %v\nOutput: %s", err, string(buildOut))
		}
	}

	// Run the receiver with specified port
	cmd := exec.Command(receiverBinary, "-port", port, "-log", "mock-receiver.log")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start mock receiver: %v", err)
	}

	// Give it a moment to start and check if it's still running
	time.Sleep(100 * time.Millisecond)
	if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
		return nil, fmt.Errorf("mock receiver process died immediately after starting - likely port already in use")
	}

	return cmd, nil
}

// submitTestCertificate submits a test certificate to the proxy
func submitTestCertificate(proxyAddr string, networkID uint32, height, withdrawalValue uint64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, proxyAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := v1.NewCertificateSubmissionServiceClient(conn)

	// Create simple test certificate
	req := &v1.SubmitCertificateRequest{
		Certificate: &typesv1.Certificate{
			NetworkId:           networkID,
			Height:              height,
			PrevLocalExitRoot:   &interopv1.FixedBytes32{Value: make([]byte, 32)},
			NewLocalExitRoot:    &interopv1.FixedBytes32{Value: make([]byte, 32)},
			BridgeExits:         []*interopv1.BridgeExit{},
			ImportedBridgeExits: []*interopv1.ImportedBridgeExit{},
			Metadata:            &interopv1.FixedBytes32{Value: nil},
		},
	}

	if withdrawalValue > 0 {
		req.Certificate.BridgeExits = []*interopv1.BridgeExit{
			{
				LeafType:    interopv1.LeafType_LEAF_TYPE_TRANSFER,
				DestNetwork: 0,
				DestAddress: &interopv1.FixedBytes20{Value: []byte("0x1234567890123456789012345678901234567890")},
				TokenInfo: &interopv1.TokenInfo{
					OriginNetwork:      1,
					OriginTokenAddress: &interopv1.FixedBytes20{Value: []byte("0x1234567890123456789012345678901234567890")},
				},
				Amount:   &interopv1.FixedBytes32{Value: uint64ToBytes(withdrawalValue)},
				Metadata: &interopv1.FixedBytes32{Value: nil},
			},
		}
	}

	_, err = client.SubmitCertificate(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to submit certificate: %v", err)
	}

	fmt.Printf("  ✓ Submitted certificate for network %d (height %d)\n", networkID, height)
	return nil
}

// checkSchedulerStatus checks if the scheduler is active
func checkSchedulerStatus(httpAddr, key string) bool {
	add := fmt.Sprintf("%s?key=%s", httpAddr, key)
	req, err := http.NewRequest(http.MethodGet, add, nil)
	if err != nil {
		slog.Error("failed to create request", "err", err)
		return false
	}
	req.Header.Add("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("failed to get scheduler status", "status", resp.StatusCode)
		return false
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("failed to read response body", "err", err)
		return false
	}

	type Status struct {
		SchedulerActive bool `json:"scheduler_active"`
	}

	var status Status
	err = json.Unmarshal(body, &status)
	if err != nil {
		slog.Error("failed to unmarshal response body", "err", err)
		return false
	}

	return status.SchedulerActive
}

func uint64ToBytes(value uint64) []byte {
	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, value)
	return bytes
}
