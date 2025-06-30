package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

// runPassthroughTest runs a simple test to verify passthrough works
func runPassthroughTest() {
	fmt.Println("=====================================")
	fmt.Println("Certificate Passthrough Test")
	fmt.Println("=====================================")
	fmt.Println()

	// Clean up first
	os.Remove("passthrough-test.db")
	os.Remove("passthrough-test.log")
	os.Remove("passthrough-receiver.log")

	// Start a simple receiver
	fmt.Println("1. Starting mock receiver on :50052...")
	receiverCmd := startSimpleReceiver("50052")
	defer receiverCmd.Process.Kill()
	time.Sleep(2 * time.Second)

	// Start proxy
	fmt.Println("2. Starting proxy...")
	proxyCmd := exec.Command("./proxy",
		"--db", "passthrough-test.db",
		"--http", ":8080",
		"--grpc", ":50051",
		"--aggsender-addr", "127.0.0.1:50052",
		"--delayed-chains", "1,137", // Only delay chains 1 and 137
	)

	logFile, _ := os.Create("passthrough-test.log")
	defer logFile.Close()
	proxyCmd.Stdout = logFile
	proxyCmd.Stderr = logFile

	if err := proxyCmd.Start(); err != nil {
		log.Fatalf("Failed to start proxy: %v", err)
	}
	defer proxyCmd.Process.Kill()

	// Wait for proxy to be ready
	fmt.Println("3. Waiting for proxy to start...")
	time.Sleep(3 * time.Second)

	// Submit a non-delayed certificate
	fmt.Println("4. Submitting certificate for chain 10 (should pass through)...")
	if err := submitTestCertificate("127.0.0.1:50051", 10, 100); err != nil {
		fmt.Printf("   ❌ Failed: %v\n", err)
	}

	// Wait and check
	fmt.Println("5. Checking if certificate was received...")
	time.Sleep(2 * time.Second)

	receiverLog, _ := os.ReadFile("passthrough-receiver.log")
	if strings.Contains(string(receiverLog), "RECEIVED network 10") {
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
func startSimpleReceiver(port string) *exec.Cmd {
	receiverCode := `
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	
	"google.golang.org/grpc"
	
	interopv1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/interop/types/v1"
	typesv1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/types/v1"
	v1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/v1"
)

type server struct {
	v1.UnimplementedCertificateSubmissionServiceServer
}

func (s *server) SubmitCertificate(ctx context.Context, req *v1.SubmitCertificateRequest) (*v1.SubmitCertificateResponse, error) {
	networkID := uint32(0)
	if req.Certificate != nil {
		networkID = req.Certificate.NetworkId
	}
	
	msg := fmt.Sprintf("RECEIVED network %%d\n", networkID)
	log.Print(msg)
	
	// Write to file
	f, _ := os.OpenFile("passthrough-receiver.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	f.WriteString(msg)
	f.Close()
	
	return &v1.SubmitCertificateResponse{
		CertificateId: &typesv1.CertificateId{
			Value: &interopv1.FixedBytes32{Value: []byte{1, 2, 3}},
		},
	}, nil
}

func main() {
	lis, err := net.Listen("tcp", "127.0.0.1:%s")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	
	s := grpc.NewServer()
	v1.RegisterCertificateSubmissionServiceServer(s, &server{})
	
	log.Println("Receiver listening on 127.0.0.1:%s")
	s.Serve(lis)
}
`
	// Format the code with the port
	receiverCode = fmt.Sprintf(receiverCode, port, port)

	// Write to temp file
	tmpFile := "simple_receiver_temp.go"
	os.WriteFile(tmpFile, []byte(receiverCode), 0644)

	// Build
	buildCmd := exec.Command("go", "build", "-o", "simple_receiver", tmpFile)
	if err := buildCmd.Run(); err != nil {
		log.Fatalf("Failed to build receiver: %v", err)
	}
	os.Remove(tmpFile)

	// Run
	cmd := exec.Command("./simple_receiver")
	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start receiver: %v", err)
	}

	// Cleanup after 30 seconds
	go func() {
		time.Sleep(30 * time.Second)
		os.Remove("simple_receiver")
	}()

	return cmd
}
