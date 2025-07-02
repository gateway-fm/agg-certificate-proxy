package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"

	interopv1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/interop/types/v1"
	typesv1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/types/v1"
	v1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/v1"
)

type server struct {
	v1.UnimplementedCertificateSubmissionServiceServer
	logFile *os.File
}

func (s *server) SubmitCertificate(ctx context.Context, req *v1.SubmitCertificateRequest) (*v1.SubmitCertificateResponse, error) {
	networkID := uint32(0)
	if req.Certificate != nil {
		networkID = req.Certificate.NetworkId
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf("[%s] RECEIVED CERTIFICATE for network %d\n", timestamp, networkID)

	// Log to both console and file
	log.Print(msg)

	if s.logFile != nil {
		s.logFile.WriteString(msg)
		s.logFile.Sync()
	}

	return &v1.SubmitCertificateResponse{
		CertificateId: &typesv1.CertificateId{
			Value: &interopv1.FixedBytes32{Value: []byte{1, 2, 3, 4}},
		},
	}, nil
}

func main() {
	var port string
	var logPath string

	flag.StringVar(&port, "port", "50052", "Port to listen on")
	flag.StringVar(&logPath, "log", "mock-receiver.log", "Log file path")
	flag.Parse()

	// Clear log file at startup
	os.WriteFile(logPath, []byte{}, 0644)

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal("Failed to open log file: ", err)
	}
	defer logFile.Close()

	lis, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		log.Fatal("Failed to listen: ", err)
	}

	s := grpc.NewServer()
	v1.RegisterCertificateSubmissionServiceServer(s, &server{logFile: logFile})

	log.Printf("Mock receiver listening on 127.0.0.1:%s", port)
	if err := s.Serve(lis); err != nil {
		log.Fatal("Failed to serve: ", err)
	}
}
