package main

// NOTE: Before running this test, you need to generate the proto files:
//   cd .. && make proto
// This will generate the NodeStateService and ConfigurationService types

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/gateway-fm/agg-certificate-proxy/internal/certificate"
	interopv1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/interop/types/v1"
	typesv1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/types/v1"
	v1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/v1"
)

func runOverridesTest() {
	fmt.Println("=================================================")
	fmt.Println("AggLayer Certificate Proxy - Certificate override test")
	fmt.Println("Testing that an overriden certificate is sent in the next service iteration")
	fmt.Println("=================================================")
	fmt.Println()

	// Test configuration
	proxyAddr := "127.0.0.1:50071"
	backendAddr := "127.0.0.1:50072"
	dbFile := "overrides-test.db"
	logFile := "overrides-test.log"

	// Clean up any stale processes before starting
	fmt.Println("Cleaning up any existing processes...")
	exec.Command("pkill", "-f", "mock_receiver").Run()
	exec.Command("pkill", "-f", "proxy").Run()
	time.Sleep(500 * time.Millisecond)

	// Cleanup
	defer func() {
		fmt.Println("Cleaning up any existing processes...")
		exec.Command("pkill", "-f", "mock_receiver").Run()
		exec.Command("pkill", "-f", "proxy").Run()
		os.Remove(dbFile)
		os.Remove(logFile)
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
		"--delay", "48h", // really long time :)
		"--scheduler-interval", "500ms",
		"--kill-switch-api-key", "test-key",
		"--kill-restart-api-key", "test-key",
		"--data-key", "test-data-key",
		"--certificate-override-key", "test-certificate-override-key",
	)
	proxyCmd.Stdout = proxyLogFileHandle
	proxyCmd.Stderr = proxyLogFileHandle

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
	nodeStateClient := v1.NewNodeStateServiceClient(conn)

	ctx := context.Background()

	// Test 1: Certificate submission should be intercepted
	fmt.Println("\n==== Test 1: Certificate Submission with withdrawal (intercepted) ====")
	certResponse, err := certClient.SubmitCertificate(ctx, &v1.SubmitCertificateRequest{
		Certificate: &typesv1.Certificate{
			NetworkId: 1, // Delayed chain
			Height:    100,
			BridgeExits: []*interopv1.BridgeExit{
				{
					TokenInfo: &interopv1.TokenInfo{
						OriginNetwork:      1,
						OriginTokenAddress: &interopv1.FixedBytes20{Value: []byte("0x1234567890123456789012345678901234567890")},
					},
					DestNetwork: 1,
					DestAddress: &interopv1.FixedBytes20{Value: []byte("0x1234567890123456789012345678901234567890")},
					Amount:      &interopv1.FixedBytes32{Value: []byte("1000000000000000000")},
					Metadata:    &interopv1.FixedBytes32{Value: []byte("metadata")},
				},
			},
			ImportedBridgeExits: []*interopv1.ImportedBridgeExit{},
			PrevLocalExitRoot:   &interopv1.FixedBytes32{Value: []byte("prev")},
			NewLocalExitRoot:    &interopv1.FixedBytes32{Value: []byte("new")},
			Metadata:            &interopv1.FixedBytes32{Value: nil},
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

	// now quickly request the certificate header to ensure it is returned by our service intercepting
	headerResp, err := nodeStateClient.GetCertificateHeader(ctx, &v1.GetCertificateHeaderRequest{
		CertificateId: &typesv1.CertificateId{
			Value: certResponse.CertificateId.Value,
		},
	})
	if err != nil {
		fmt.Printf("❌ failed to get certificate header: %v\n", err)
	}
	if headerResp.CertificateHeader.Status != typesv1.CertificateStatus_CERTIFICATE_STATUS_PENDING {
		fmt.Println("❌ certificate header status is not pending")
	} else {
		fmt.Println("✅ certificate header status is pending")
	}

	// ensure the json response from the server also shows this has not been
	// processed yet
	req, _ := http.NewRequest(http.MethodGet, "http://localhost:8095?key=test-data-key", nil)
	req.Header.Add("Accept", "application/json")
	allData, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("❌ failed to get all data: %v\n", err)
	}
	type Data struct {
		Certificates []certificate.CertificateView `json:"certificates"`
	}
	var data Data
	if err := json.NewDecoder(allData.Body).Decode(&data); err != nil {
		fmt.Printf("❌ failed to decode all data: %v\n", err)
	}

	if len(data.Certificates) != 1 {
		fmt.Println("❌ expected 1 certificate, got", len(data.Certificates))
	} else {
		fmt.Println("✅ got 1 certificate as expected")
	}

	if data.Certificates[0].ProcessedAt.Valid {
		fmt.Println("❌ certificate should not be processed yet as it has not been overridden")
	} else {
		fmt.Println("✅ certificate is not processed yet as it has not been overridden")
	}

	// attempt to override without a key
	fmt.Println("\nAttempting to override without a key...")
	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:8095/override?cert_id=%d", data.Certificates[0].ID), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("❌ failed to override certificate: %v\n", err)
	} else {
		if resp.StatusCode != http.StatusUnauthorized {
			fmt.Println("❌ expected 401 status code, got", resp.StatusCode)
		} else {
			fmt.Println("✅ got 401 status code as expected")
		}
	}

	// attempt to override without a cert id
	fmt.Println("\nAttempting to override without a cert id...")
	req, _ = http.NewRequest(http.MethodPost, "http://localhost:8095/override?key=test-certificate-override-key", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("❌ failed to override certificate: %v\n", err)
	} else {
		if resp.StatusCode != http.StatusBadRequest {
			body, _ := io.ReadAll(resp.Body)
			fmt.Println("❌ expected 400 status code, got", resp.StatusCode, ":", string(body))
		} else {
			fmt.Println("✅ got 400 status code as expected")
		}
	}

	// now override the certificate properly
	fmt.Println("\nOverriding certificate with valid key and cert id...")
	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:8095/override?key=test-certificate-override-key&cert_id=%d", data.Certificates[0].ID), nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("❌ failed to override certificate: %v\n", err)
	} else {
		if resp.StatusCode == http.StatusOK {
			fmt.Println("✅ overridden certificate")
		} else {
			body, _ := io.ReadAll(resp.Body)
			fmt.Println("❌ expected 200 status code, got", resp.StatusCode, ":", string(body))
		}
	}

	// wait a little moment to ensure the scheduler has time to fire
	time.Sleep(1 * time.Second)

	// now check the certificate is processed
	req, _ = http.NewRequest(http.MethodGet, "http://localhost:8095?key=test-data-key", nil)
	req.Header.Add("Accept", "application/json")
	allData, err = http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("❌ failed to get all data: %v\n", err)
	}
	if err := json.NewDecoder(allData.Body).Decode(&data); err != nil {
		fmt.Printf("❌ failed to decode all data: %v\n", err)
	}
	if len(data.Certificates) != 1 {
		fmt.Println("❌ expected 1 certificate, got", len(data.Certificates))
	} else {
		fmt.Println("✅ got 1 certificate as expected")
	}

	if data.Certificates[0].ProcessedAt.Valid {
		fmt.Println("✅ certificate is processed as expected")
	} else {
		fmt.Println("❌ certificate is not processed")
	}

	// ensure the header request also now goes through to the upstream service and
	// isn't handled by the proxy
	headerResp, err = nodeStateClient.GetCertificateHeader(ctx, &v1.GetCertificateHeaderRequest{
		CertificateId: &typesv1.CertificateId{
			Value: certResponse.CertificateId.Value,
		},
	})
	if err != nil {
		fmt.Printf("❌ failed to get certificate header: %v\n", err)
	} else {
		fmt.Println("✅ got certificate header")
	}

	if headerResp.CertificateHeader.Status != typesv1.CertificateStatus_CERTIFICATE_STATUS_SETTLED {
		fmt.Println("❌ certificate header status is not settled")
	} else {
		fmt.Println("✅ certificate header status is settled")
	}
}
