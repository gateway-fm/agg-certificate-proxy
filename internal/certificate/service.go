package certificate

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"

	typesv1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/types/v1"
	v1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/v1"
)

// Db defines the interface for database operations.
type Db interface {
	Init() error
	Close() error
	StoreCertificate(rawProto []byte, metadata string) error
	GetProcessableCertificates() ([]Certificate, error)
	MarkCertificateProcessed(id int64) error
	GetCertificates() ([]Certificate, error)
	GetConfigValue(key string) (string, error)
	SetConfigValue(key, value string) error
	GetCredential(key string) (string, error)
	SetCredential(key, value string) error
	GetSchedulerStatus() (bool, error)
	SetSchedulerStatus(isActive bool) error
	RecordKillSwitchAttempt(attemptType string) error
	GetRecentKillSwitchAttempts(attemptType string, duration time.Duration) (int, error)
	CleanupOldKillSwitchAttempts(olderThan time.Duration) error
}

// Service handles the business logic for certificates.
type Service struct {
	db Db
}

// NewService creates a new certificate service.
func NewService(db Db) *Service {
	return &Service{db: db}
}

// StoreCertificate stores a certificate for delayed processing.
func (s *Service) StoreCertificate(rawProto []byte, metadata string) error {
	return s.db.StoreCertificate(rawProto, metadata)
}

// GetCertificates retrieves all certificates.
func (s *Service) GetCertificates() ([]Certificate, error) {
	return s.db.GetCertificates()
}

// GetConfigValue retrieves a configuration value.
func (s *Service) GetConfigValue(key string) (string, error) {
	return s.db.GetConfigValue(key)
}

// GetDelayedChains retrieves the list of chain IDs that should be delayed.
func (s *Service) GetDelayedChains() ([]uint32, error) {
	value, err := s.db.GetConfigValue("delayed_chains")
	if err != nil {
		return nil, err
	}

	if value == "" {
		return []uint32{}, nil
	}

	// Parse comma-separated chain IDs
	parts := strings.Split(value, ",")
	chains := make([]uint32, 0, len(parts))
	for _, part := range parts {
		chainID, err := strconv.ParseUint(strings.TrimSpace(part), 10, 32)
		if err != nil {
			log.Printf("warning: invalid chain ID in configuration: %s", part)
			continue
		}
		chains = append(chains, uint32(chainID))
	}

	return chains, nil
}

// IsChainDelayed checks if a specific chain ID should be delayed.
func (s *Service) IsChainDelayed(chainID uint32) (bool, error) {
	chains, err := s.GetDelayedChains()
	if err != nil {
		return false, err
	}

	for _, id := range chains {
		if id == chainID {
			return true, nil
		}
	}

	return false, nil
}

// SetDelayedChains updates the list of chain IDs that should be delayed.
func (s *Service) SetDelayedChains(chains []uint32) error {
	// Convert to comma-separated string
	parts := make([]string, len(chains))
	for i, chain := range chains {
		parts[i] = strconv.FormatUint(uint64(chain), 10)
	}
	value := strings.Join(parts, ",")

	return s.db.SetConfigValue("delayed_chains", value)
}

// ProcessPendingCertificates processes certificates that are ready.
func (s *Service) ProcessPendingCertificates() {
	log.Println("Checking for processable certificates...")
	certs, err := s.db.GetProcessableCertificates()
	if err != nil {
		log.Printf("error getting processable certificates: %v", err)
		return
	}

	if len(certs) == 0 {
		log.Println("No processable certificates found.")
		return
	}

	log.Printf("Found %d processable certificates.", len(certs))

	for _, cert := range certs {
		if err := s.SendToAggSender(cert); err != nil {
			log.Printf("error sending certificate %d to agg sender: %v", cert.ID, err)
			continue
		}

		if err := s.db.MarkCertificateProcessed(cert.ID); err != nil {
			log.Printf("error marking certificate %d as processed: %v", cert.ID, err)
		}
	}
}

// SendToAggSender sends a certificate to the agg sender.
func (s *Service) SendToAggSender(cert Certificate) error {
	if cert.ID == 0 {
		log.Printf("Sending immediate certificate to aggsender...")
	} else {
		log.Printf("Sending certificate %d to aggsender...", cert.ID)
	}

	// Check if we have an aggsender_address in config
	aggSenderAddr, err := s.db.GetConfigValue("aggsender_address")
	if err != nil || aggSenderAddr == "" {
		log.Printf("No aggsender_address configured, using default: localhost:50052")
		aggSenderAddr = "localhost:50052" // Default to our mock receiver
	}

	// Create gRPC connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, aggSenderAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to aggsender at %s: %v", aggSenderAddr, err)
	}
	defer conn.Close()

	// Create client
	client := v1.NewCertificateSubmissionServiceClient(conn)

	// Unmarshal the certificate
	var certProto typesv1.Certificate
	if err := proto.Unmarshal(cert.RawProto, &certProto); err != nil {
		return fmt.Errorf("failed to unmarshal certificate: %v", err)
	}

	// Create request
	req := &v1.SubmitCertificateRequest{
		Certificate: &certProto,
	}

	// Send the certificate
	if cert.ID == 0 {
		log.Printf("Forwarding immediate certificate (network %d) to aggsender at %s", certProto.GetNetworkId(), aggSenderAddr)
	} else {
		log.Printf("Forwarding certificate %d (network %d) to aggsender at %s", cert.ID, certProto.GetNetworkId(), aggSenderAddr)
	}

	resp, err := client.SubmitCertificate(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to submit certificate to aggsender: %v", err)
	}

	if resp.CertificateId != nil && resp.CertificateId.Value != nil {
		if cert.ID == 0 {
			log.Printf("Immediate certificate forwarded successfully, received ID: %x", resp.CertificateId.Value.Value)
		} else {
			log.Printf("Certificate %d forwarded successfully, received ID: %x", cert.ID, resp.CertificateId.Value.Value)
		}
	} else {
		if cert.ID == 0 {
			log.Printf("Immediate certificate forwarded successfully")
		} else {
			log.Printf("Certificate %d forwarded successfully", cert.ID)
		}
	}

	return nil
}
