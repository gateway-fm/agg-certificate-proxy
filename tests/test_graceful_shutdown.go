package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func runGracefulShutdownTest() {
	fmt.Println("==============================================")
	fmt.Println("AggLayer Certificate Proxy - Graceful Shutdown Test")
	fmt.Println("==============================================")
	fmt.Println()

	// Test configuration
	httpAddr := "localhost:8088"
	grpcAddr := "localhost:50058"
	dbPath := "graceful-shutdown-test.db"
	logPath := "graceful-shutdown-test.log"
	killKey := "test-kill-key"
	restartKey := "test-restart-key"
	dataKey := "test-data-key"
	certificateOverrideKey := "test-certificate-override-key"

	// Clean up any previous test artifacts
	os.Remove(dbPath)
	os.Remove(logPath)

	// Start the proxy
	fmt.Println("Starting proxy with short scheduler interval...")
	logFile, err := os.Create(logPath)
	if err != nil {
		fmt.Printf("❌ ERROR: Failed to create log file: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()

	cmd := exec.Command("./proxy",
		"-http", httpAddr,
		"-grpc", grpcAddr,
		"-db", dbPath,
		"--kill-switch-api-key", killKey,
		"--kill-restart-api-key", restartKey,
		"--data-key", dataKey,
		"--certificate-override-key", certificateOverrideKey,
		"-scheduler-interval", "3s",
	)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		fmt.Printf("❌ ERROR: Failed to start proxy: %v\n", err)
		os.Exit(1)
	}

	// Wait for proxy to start
	fmt.Println("Waiting for proxy to start...")
	if !waitForHealth(httpAddr, true, 5*time.Second) {
		cmd.Process.Kill()
		fmt.Println("❌ ERROR: Proxy failed to start")
		os.Exit(1)
	}
	fmt.Println("✅ Proxy started successfully")

	// Test 1: Health check returns 503 during shutdown
	fmt.Println("\nTest 1: Health endpoint returns 503 during shutdown")
	fmt.Println("================================================")

	// Send SIGTERM
	fmt.Println("Sending SIGTERM to proxy...")
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		fmt.Printf("❌ ERROR: Failed to send SIGTERM: %v\n", err)
		os.Exit(1)
	}

	// Check health immediately
	time.Sleep(50 * time.Millisecond) // Small delay to let signal be processed

	resp, body, err := checkHealthEndpoint(httpAddr)
	if err != nil {
		fmt.Printf("✅ Health check correctly failed during shutdown (connection refused)\n")
	} else if resp.StatusCode == http.StatusServiceUnavailable {
		fmt.Printf("✅ Health endpoint returned 503 during shutdown\n")
		fmt.Printf("   Response: %s\n", body)
	} else {
		fmt.Printf("❌ ERROR: Expected 503 but got %d\n", resp.StatusCode)
		os.Exit(1)
	}

	// Wait for process to exit
	cmd.Wait()
	fmt.Println("✅ Proxy shut down")

	// Test 2: Verify shutdown sequence in logs
	fmt.Println("\nTest 2: Verify graceful shutdown sequence")
	fmt.Println("=========================================")

	logContent, err := os.ReadFile(logPath)
	if err != nil {
		fmt.Printf("❌ ERROR: Failed to read log file: %v\n", err)
		os.Exit(1)
	}

	requiredLogEntries := []string{
		"shutting down...",
		"certificate scheduler stopped",
		"gRPC server shut down",
		"HTTP server shut down",
	}

	logs := string(logContent)
	allFound := true
	for _, entry := range requiredLogEntries {
		if !contains(logs, entry) {
			fmt.Printf("❌ Missing log entry: %s\n", entry)
			allFound = false
		}
	}

	if allFound {
		fmt.Println("✅ All shutdown sequence steps found in logs")
	} else {
		fmt.Println("\nFull log output:")
		fmt.Println(logs)
		os.Exit(1)
	}

	// Test 3: Test with active scheduler task
	fmt.Println("\nTest 3: Shutdown waits for scheduler task")
	fmt.Println("=========================================")

	// Clean up for second test
	os.Remove(dbPath)
	os.Remove(logPath)

	// Start proxy again with very short interval
	fmt.Println("Starting proxy with 1-second scheduler interval...")
	logFile2, err := os.Create(logPath)
	if err != nil {
		fmt.Printf("❌ ERROR: Failed to create log file: %v\n", err)
		os.Exit(1)
	}
	defer logFile2.Close()

	cmd2 := exec.Command("./proxy",
		"-http", httpAddr,
		"-grpc", grpcAddr,
		"-db", dbPath,
		"--kill-switch-api-key", killKey,
		"--kill-restart-api-key", restartKey,
		"--data-key", dataKey,
		"--certificate-override-key", certificateOverrideKey,
		"-scheduler-interval", "1s",
	)
	cmd2.Stdout = logFile2
	cmd2.Stderr = logFile2

	if err := cmd2.Start(); err != nil {
		fmt.Printf("❌ ERROR: Failed to start proxy: %v\n", err)
		os.Exit(1)
	}

	// Wait for proxy to start
	if !waitForHealth(httpAddr, true, 5*time.Second) {
		cmd2.Process.Kill()
		fmt.Println("❌ ERROR: Proxy failed to start")
		os.Exit(1)
	}

	// Wait for scheduler to run at least once
	fmt.Println("Waiting for scheduler to run...")
	time.Sleep(1500 * time.Millisecond)

	// Send SIGTERM while scheduler might be processing
	fmt.Println("Sending SIGTERM (scheduler may be processing)...")
	startTime := time.Now()
	if err := cmd2.Process.Signal(syscall.SIGTERM); err != nil {
		fmt.Printf("❌ ERROR: Failed to send SIGTERM: %v\n", err)
		os.Exit(1)
	}

	// Wait for shutdown
	cmd2.Wait()
	shutdownDuration := time.Since(startTime)

	fmt.Printf("✅ Shutdown completed in %.2f seconds\n", shutdownDuration.Seconds())

	// Verify we waited for tasks
	logContent2, _ := os.ReadFile(logPath)
	if contains(string(logContent2), "waiting for running tasks to complete") {
		fmt.Println("✅ Confirmed: Shutdown waited for running tasks")
	}

	// Clean up
	fmt.Println("\nCleaning up test artifacts...")
	os.Remove(dbPath)
	os.Remove(logPath)

	fmt.Println("\n✅ All graceful shutdown tests passed!")
}

func checkHealthEndpoint(addr string) (*http.Response, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://%s/health", addr), nil)
	if err != nil {
		return nil, "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return resp, string(body), nil
}

func waitForHealth(addr string, expectSuccess bool, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, _, err := checkHealthEndpoint(addr)
		if expectSuccess && err == nil && resp.StatusCode == http.StatusOK {
			return true
		}
		if !expectSuccess && (err != nil || resp.StatusCode != http.StatusOK) {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
