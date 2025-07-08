package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	interopv1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/interop/types/v1"
	typesv1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/types/v1"
	v1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func runMetricsIntegrationTest() {
	fmt.Println("=====================================")
	fmt.Println("Metrics Integration Test")
	fmt.Println("=====================================")
	fmt.Println()

	// Clean up any stale processes before starting
	fmt.Println("Cleaning up any existing processes...")
	exec.Command("pkill", "-f", "mock_receiver").Run()
	exec.Command("pkill", "-f", "proxy").Run()
	time.Sleep(500 * time.Millisecond)

	// Clean up first
	os.Remove("metrics-test.db")
	os.Remove("metrics-test.log")

	killKey := "test-kill-key"
	restartKey := "test-restart-key"
	dataKey := "test-data-key"
	certificateOverrideKey := "test-certificate-override-key"

	// Start proxy
	fmt.Println("1. Starting proxy...")
	proxyCmd := exec.Command("./proxy",
		"--db", "metrics-test.db",
		"--http", ":8080",
		"--grpc", ":50051",
		"--kill-switch-api-key", killKey,
		"--kill-restart-api-key", restartKey,
		"--data-key", dataKey,
		"--certificate-override-key", certificateOverrideKey,
		"--aggsender-addr", "127.0.0.1:50052",
		"--delayed-chains", "1,2", // Only delay chains 1 and 2
	)

	logFile, _ := os.Create("metrics-test.log")
	defer logFile.Close()
	proxyCmd.Stdout = logFile
	proxyCmd.Stderr = logFile

	if err := proxyCmd.Start(); err != nil {
		log.Fatal("Failed to start proxy: ", err)
	}
	defer proxyCmd.Process.Kill()

	// Wait for proxy to be ready
	fmt.Println("2. Waiting for proxy to start...")
	time.Sleep(3 * time.Second)

	// now send two certifcates for both networks
	fmt.Println("3. Submitting certificate for network 1...")
	if err := submitMetricsTestCertificate("127.0.0.1:50051", 1, 1100000000000000000); err != nil {
		log.Fatal("Failed to submit certificate: ", err)
	}
	fmt.Println("4. Submitting certificate for network 2...")
	if err := submitMetricsTestCertificate("127.0.0.1:50051", 2, 2100000000000000000); err != nil {
		log.Fatal("Failed to submit certificate: ", err)
	}

	// now poll the proxy json endpoint until the certificates have been processed
	type QuickCheck struct {
		Certificates []interface{} `json:"certificates"`
	}

	killSwitch := 0
	check := func() bool {
		req, err := http.NewRequest("GET", "http://127.0.0.1:8080?key="+dataKey, nil)
		if err != nil {
			log.Fatal("Failed to create request: ", err)
		}
		req.Header.Set("Accept", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatal("Failed to get proxy json: ", err)
		}
		defer resp.Body.Close()

		var quickCheck QuickCheck
		if err := json.NewDecoder(resp.Body).Decode(&quickCheck); err != nil {
			log.Fatal("Failed to unmarshal proxy json: ", err)
		}

		if len(quickCheck.Certificates) == 2 {
			return true
		}
		return false
	}
	for {
		if killSwitch > 10 {
			log.Fatal("Failed to process certificates")
		}
		if check() {
			break
		}
		time.Sleep(500 * time.Millisecond)
		killSwitch++
	}

	// wait a little time for the metrics to be updated
	time.Sleep(2 * time.Second)

	// now lets check the metrics
	fmt.Println("5. Fetching metrics...")
	resp, err := http.Get("http://127.0.0.1:8080/metrics")
	if err != nil {
		log.Fatal("Failed to get metrics: ", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Failed to read metrics: ", err)
	}

	// check the first metric
	expectedMetrics := []string{
		"certificate_total_count 2",
		"certificate_total_eth 3.2",
		"network_1_total_eth 1.1",
		"network_2_total_eth 2.1",
	}

	ok := true

	for _, expectedMetric := range expectedMetrics {
		if !strings.Contains(string(body), expectedMetric) {
			log.Printf("  ❌ Expected metric not found: %s", expectedMetric)
			ok = false
		} else {
			log.Printf("  ✅ Expected metric found: %s", expectedMetric)
		}
	}

	if ok {
		fmt.Println("6. ✅ Metrics check complete")
	} else {
		fmt.Println("6. ❌ Metrics check failed")
	}

	// now lets send a third certificate for network 1 to ensure the metrics are updated
	fmt.Println("7. Submitting certificate for network 1...")
	if err := submitMetricsTestCertificate("127.0.0.1:50051", 1, 3100000000000000000); err != nil {
		log.Fatal("Failed to submit certificate: ", err)
	}

	// wait a little time for the metrics to be updated
	time.Sleep(2 * time.Second)

	// now lets check the metrics
	fmt.Println("8. Fetching metrics...")
	resp, err = http.Get("http://127.0.0.1:8080/metrics")
	if err != nil {
		log.Fatal("Failed to get metrics: ", err)
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Failed to read metrics: ", err)
	}

	expectedMetrics = []string{
		"certificate_total_count 3",
		"certificate_total_eth 6.3",
		"network_1_total_eth 4.2",
		"network_2_total_eth 2.1",
	}

	ok = true
	for _, expectedMetric := range expectedMetrics {
		if !strings.Contains(string(body), expectedMetric) {
			log.Printf("  ❌ Expected metric not found: %s", expectedMetric)
			ok = false
		} else {
			log.Printf("  ✅ Expected metric found: %s", expectedMetric)
		}
	}

	if ok {
		fmt.Println("9. ✅ Metrics check complete")
	} else {
		fmt.Println("9. ❌ Metrics check failed")
	}

	fmt.Println()
	fmt.Println("=====================================")
	fmt.Println("Metrics Integration Test Complete")
	fmt.Println("=====================================")
	fmt.Println()
}

func submitMetricsTestCertificate(proxyAddr string, networkID uint32, amount uint64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, proxyAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := v1.NewCertificateSubmissionServiceClient(conn)

	amountBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(amountBytes, amount)

	req := &v1.SubmitCertificateRequest{
		Certificate: &typesv1.Certificate{
			NetworkId: networkID,
			Height:    1,
			BridgeExits: []*interopv1.BridgeExit{
				{
					LeafType: 321,
					TokenInfo: &interopv1.TokenInfo{
						OriginNetwork:      123,
						OriginTokenAddress: &interopv1.FixedBytes20{Value: []byte(randomAddress())},
					},
					DestNetwork: 123,
					DestAddress: &interopv1.FixedBytes20{Value: []byte(randomAddress())},
					Amount:      &interopv1.FixedBytes32{Value: amountBytes},
				},
			},
			ImportedBridgeExits: []*interopv1.ImportedBridgeExit{},
			PrevLocalExitRoot:   &interopv1.FixedBytes32{Value: make([]byte, 32)},
			NewLocalExitRoot:    &interopv1.FixedBytes32{Value: make([]byte, 32)},
			Metadata:            &interopv1.FixedBytes32{Value: nil},
		},
	}

	_, err = client.SubmitCertificate(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to submit certificate: %v", err)
	}

	return nil
}
