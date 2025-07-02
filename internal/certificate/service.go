package certificate

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"

	"log/slog"

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
			slog.Warn("invalid chain ID in configuration", "chain", part)
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
	slog.Info("checking for processable certificates...")
	certs, err := s.db.GetProcessableCertificates()
	if err != nil {
		slog.Error("error getting processable certificates", "err", err)
		return
	}

	if len(certs) == 0 {
		slog.Info("no processable certificates found.")
		return
	}

	slog.Info("found processable certificates.", "count", len(certs))

	for _, cert := range certs {
		if err := s.SendToAggSender(cert); err != nil {
			slog.Error("error sending certificate to agg sender", "certificate", cert.ID, "err", err)
			continue
		}

		if err := s.db.MarkCertificateProcessed(cert.ID); err != nil {
			slog.Error("error marking certificate as processed", "certificate", cert.ID, "err", err)
		}
	}
}

// SendToAggSender sends a certificate to the agg sender.
func (s *Service) SendToAggSender(cert Certificate) error {
	if cert.ID == 0 {
		slog.Info("sending immediate certificate to aggsender...")
	} else {
		slog.Info("sending certificate to aggsender...", "certificate", cert.ID)
	}

	// Check if we have an aggsender_address in config
	aggSenderAddr, err := s.db.GetConfigValue("aggsender_address")
	if err != nil || aggSenderAddr == "" {
		slog.Warn("no aggsender_address configured, using default: localhost:50052")
		aggSenderAddr = "localhost:50052" // Default to our mock receiver
	}

	// Create gRPC connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(aggSenderAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to aggsender at %s: %v", aggSenderAddr, err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			slog.Error("failed to close gRPC connection to aggsender", "err", err)
		}
	}()

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
		slog.Info("forwarding immediate certificate to aggsender", "network", certProto.GetNetworkId(), "address", aggSenderAddr)
	} else {
		slog.Info("forwarding certificate to aggsender", "certificate", cert.ID, "network", certProto.GetNetworkId(), "address", aggSenderAddr)
	}

	resp, err := client.SubmitCertificate(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to submit certificate to aggsender: %v", err)
	}

	if resp.CertificateId != nil && resp.CertificateId.Value != nil {
		if cert.ID == 0 {
			slog.Info("immediate certificate forwarded successfully", "received-id", resp.CertificateId.Value.Value)
		} else {
			slog.Info("certificate forwarded successfully", "certificate", cert.ID, "received-id", resp.CertificateId.Value.Value)
		}
	} else {
		if cert.ID == 0 {
			slog.Info("immediate certificate forwarded successfully")
		} else {
			slog.Info("certificate forwarded successfully", "certificate", cert.ID)
		}
	}

	return nil
}
