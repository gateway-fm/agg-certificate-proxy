package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// runPassthroughTest runs a simple test to verify passthrough works
func runPassthroughTest() {
	fmt.Println("=====================================")
	fmt.Println("Certificate Passthrough Test")
	fmt.Println("=====================================")
	fmt.Println()

	// Clean up any stale processes before starting
	fmt.Println("Cleaning up any existing processes...")
	exec.Command("pkill", "-f", "mock_receiver").Run()
	exec.Command("pkill", "-f", "proxy").Run()
	time.Sleep(500 * time.Millisecond)

	// Clean up first
	os.Remove("passthrough-test.db")
	os.Remove("passthrough-test.log")
	os.Remove("passthrough-receiver.log")

	// Start a simple receiver
	fmt.Println("1. Starting mock receiver on :50052...")
	receiverCmd, err := startSimpleReceiver("50052")
	if err != nil {
		log.Fatalf("Failed to start receiver: %v", err)
	}
	defer receiverCmd.Process.Kill()
	time.Sleep(2 * time.Second)

	killKey := "test-kill-key"
	restartKey := "test-restart-key"
	dataKey := "test-data-key"
	certificateOverrideKey := "test-certificate-override-key"

	// Start proxy with no delayed chains (all pass through)
	fmt.Println("Step 2: Starting certificate proxy (all passthrough mode)...")
	proxyCmd := exec.Command("./proxy",
		"--db", "passthrough-test.db",
		"--http", ":8081",
		"--grpc", ":50054",
		"--kill-switch-api-key", killKey,
		"--kill-restart-api-key", restartKey,
		"--data-key", dataKey,
		"--certificate-override-key", certificateOverrideKey,
		"--delay", "5s",
		"--scheduler-interval", "1s",
		"--aggsender-addr", "127.0.0.1:50052",
		"--delayed-chains", "", // Empty = no delayed chains
	)

	logFile, _ := os.Create("passthrough-test.log")
	defer logFile.Close()
	proxyCmd.Stdout = logFile
	proxyCmd.Stderr = logFile

	if err := proxyCmd.Start(); err != nil {
		log.Fatal("Failed to start proxy: ", err)
	}
	defer proxyCmd.Process.Kill()

	// Wait for proxy to be ready
	fmt.Println("3. Waiting for proxy to start...")
	time.Sleep(3 * time.Second)

	// Submit a non-delayed certificate
	fmt.Println("4. Submitting certificate for chain 10 (should pass through)...")
	if err := submitTestCertificate("127.0.0.1:50054", 10, 100, 1000); err != nil {
		fmt.Printf("   ❌ Failed: %v\n", err)
	}

	// Wait and check
	fmt.Println("5. Checking if certificate was received...")
	time.Sleep(2 * time.Second)

	receiverLog, _ := os.ReadFile("passthrough-receiver.log")
	if strings.Contains(string(receiverLog), "RECEIVED CERTIFICATE for network 10") {
		fmt.Println("   ✅ SUCCESS: Certificate passed through!")
	} else {
		fmt.Println("   ❌ FAILED: Certificate was not received")
		fmt.Println("\nProxy log:")
		proxyLog, _ := os.ReadFile("passthrough-test.log")
		fmt.Println(string(proxyLog))
		fmt.Println("\nReceiver log:")
		fmt.Println(string(receiverLog))
	}

	// Cleanup
	os.Remove("passthrough-test.db")
	os.Remove("passthrough-test.log")
	os.Remove("passthrough-receiver.log")
}

// startSimpleReceiver starts a very simple mock receiver
func startSimpleReceiver(port string) (*exec.Cmd, error) {
	// Build the receiver if it doesn't exist
	receiverBinary := "./receiver/receiver"
	if _, err := os.Stat(receiverBinary); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", receiverBinary, "./receiver/main.go")
		buildOut, err := buildCmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("failed to build receiver: %v\nOutput: %s", err, string(buildOut))
		}
	}

	// Run the receiver with specified port
	cmd := exec.Command(receiverBinary, "-port", port, "-log", "passthrough-receiver.log")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start receiver: %v", err)
	}

	// Give it a moment to start and check if it's still running
	time.Sleep(100 * time.Millisecond)
	if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
		return nil, fmt.Errorf("receiver process died immediately after starting - likely port already in use")
	}

	return cmd, nil
}
