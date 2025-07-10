package certificate

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"

	"log/slog"

	"github.com/ethereum/go-ethereum/common"
	interopv1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/interop/types/v1"
	typesv1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/types/v1"
	nodev1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/v1"
	v1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/v1"
)

// GRPCServer handles incoming gRPC requests for certificate submission.
type GRPCServer struct {
	nodev1.UnimplementedCertificateSubmissionServiceServer
	nodev1.UnimplementedNodeStateServiceServer
	service                  *Service
	metricsUpdater           MetricsUpdater
	upstreamAggClientAddress string
}

type MetricsUpdater interface {
	Trigger()
}

// NewGRPCServer creates a new gRPC server.
func NewGRPCServer(service *Service, metricsUpdater MetricsUpdater, upstreamAggClientAddress string) *GRPCServer {
	return &GRPCServer{
		service:                  service,
		metricsUpdater:           metricsUpdater,
		upstreamAggClientAddress: upstreamAggClientAddress,
	}
}

// SubmitCertificate handles the submission of a new certificate.
func (s *GRPCServer) SubmitCertificate(ctx context.Context, req *nodev1.SubmitCertificateRequest) (*nodev1.SubmitCertificateResponse, error) {
	defer s.metricsUpdater.Trigger()

	slog.Info("received certificate submission request")

	// Marshal the request to store it as a blob
	rawProto, err := proto.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal certificate: %w", err)
	}

	metadata := extractMetadata(req.Certificate)
	metadataJson, err := json.Marshal(metadata)
	if err != nil {
		slog.Error("failed to marshal metadata", "err", err)
	}

	// Check if this chain should be delayed
	networkID := req.Certificate.GetNetworkId()
	isDelayed, err := s.service.IsChainDelayed(networkID)
	if err != nil {
		slog.Error("error checking if chain is delayed", "chain", networkID, "err", err)
		// Default to delaying on error
		isDelayed = true
	}

	certId := generateCertificateId(req.Certificate)
	certHex := fmt.Sprintf("0x%x", certId.Value.Value)

	var resp *nodev1.SubmitCertificateResponse

	withdrawalValue := big.NewInt(0)
	for _, bridgeExit := range req.Certificate.GetBridgeExits() {
		value := bridgeExit.GetAmount().GetValue()
		asBig := big.NewInt(0).SetBytes(value)
		withdrawalValue.Add(withdrawalValue, asBig)
	}

	slog.Info("withdrawal value", "value", withdrawalValue.String())

	if !isDelayed {
		// Send immediately
		resp, err = s.sendCertificateImmediately(rawProto, string(metadataJson))
		slog.Info("successfully sent certificate for network immediately", "network", networkID)
	} else {
		if withdrawalValue.Cmp(big.NewInt(0)) == 0 {
			resp, err = s.sendCertificateImmediately(rawProto, string(metadataJson))
			slog.Info("successfully sent certificate for network immediately", "network", networkID)
		} else {
			isSuspicious, err := s.checkForSuspiciousValue(req, certHex)
			if err != nil {
				slog.Error("failed to check for suspicious value", "err", err)
				return nil, fmt.Errorf("failed to check for suspicious value: %w", err)
			}
			if isSuspicious {
				slog.Info("certificate is suspicious, locking certificate", "network", networkID)
				resp, err = s.storeCertificate(rawProto, string(metadataJson), req.Certificate, certId)
			} else {
				slog.Info("certificate doesn't appear to be suspicious, sending immediately", "network", networkID)
				resp, err = s.sendCertificateImmediately(rawProto, string(metadataJson))
			}
		}
	}

	responseIdHex := fmt.Sprintf("0x%x", resp.CertificateId.Value.Value)
	slog.Info("certificate submission response", "ourCertId", certHex, "responseId", responseIdHex)

	return resp, nil
}

func (s *GRPCServer) sendCertificateImmediately(rawProto []byte, metadataJson string) (*nodev1.SubmitCertificateResponse, error) {
	cert := Certificate{ID: 0, RawProto: rawProto, Metadata: string(metadataJson)}
	resp, err := s.service.SendToAggSender(cert)
	if err != nil {
		slog.Error("failed to send certificate immediately", "err", err)
		return nil, fmt.Errorf("failed to send certificate immediately: %w", err)
	}
	return resp, err
}

func (s *GRPCServer) storeCertificate(rawProto []byte, metadataJson string, certificate *typesv1.Certificate, certId *typesv1.CertificateId) (*nodev1.SubmitCertificateResponse, error) {
	if err := s.service.StoreCertificate(rawProto, string(metadataJson), certId.Value.Value); err != nil {
		return nil, fmt.Errorf("failed to store certificate: %w", err)
	}
	return &nodev1.SubmitCertificateResponse{
		CertificateId: certId,
	}, nil
}

// checkForSuspiciousValue will use the config passed at app startup to determine a dollar value for tokens
// in the certificate.  If it goes over the configured threshold then we will return true and lock the certificate
// likewise if we encounter a token address that is not in the config we will return true to lock the certificate
func (s *GRPCServer) checkForSuspiciousValue(req *nodev1.SubmitCertificateRequest, certHex string) (bool, error) {
	// now lets total up the values of the tokens based on the config
	suspiciousValue, err := s.service.db.GetConfigValue("suspicious_value")
	if err != nil {
		slog.Error("failed to get suspicious value", "err", err)
		return true, err
	} else {
		slog.Info("suspicious value", "value", suspiciousValue)
	}
	var susLimit *big.Int
	if suspiciousValue != "" {
		parsed, err := strconv.ParseUint(suspiciousValue, 10, 64)
		if err != nil {
			slog.Error("failed to parse suspicious value", "err", err)
			return true, err
		}
		susLimit = big.NewInt(0).SetUint64(parsed)
	}

	tokenValues, err := s.service.db.GetConfigValue("token_values")
	if err != nil {
		slog.Error("failed to get token values", "err", err)
		return true, err
	} else {
		slog.Info("token values", "value", tokenValues)
	}

	if len(suspiciousValue) == 0 && len(tokenValues) == 0 {
		slog.Info("no suspicious token config found, treating as suspicious")
		return true, nil
	}

	parsedTokenValues, err := ParseTokenValues(tokenValues)
	if err != nil {
		slog.Error("failed to parse token values", "err", err)
		return true, err
	}

	totalValue := big.NewInt(0)

	for _, bridgeExit := range req.Certificate.GetBridgeExits() {
		address := bridgeExit.GetTokenInfo().GetOriginTokenAddress()
		if address == nil {
			continue
		}
		asHex := common.BytesToAddress(address.Value).Hex()
		asHex = strings.TrimPrefix(asHex, "0x")
		asHex = strings.ToLower(asHex)
		tokenDetail, ok := parsedTokenValues[asHex]
		if !ok {
			slog.Warn("token address not found in config", "address", asHex, "cert", certHex)
			return true, nil // no error here but we need to lock the certificate
		}

		amount := bridgeExit.GetAmount().GetValue()
		asBig := big.NewInt(0).SetBytes(amount)
		asFullToken := big.NewInt(0).Div(asBig, big.NewInt(0).SetUint64(tokenDetail.Multiplier))

		slog.Info("token detail", "cert", certHex, "token", asHex, "amount-token", asFullToken.String(), "amount-wei", asBig.String())

		totalValue.Add(totalValue, asFullToken.Mul(asFullToken, big.NewInt(0).SetUint64(tokenDetail.DollarValue)))

		slog.Info("intermediate total value", "cert", certHex, "value", totalValue.String(), "limit", susLimit.String())
	}

	slog.Info("suspicious calcs", "cert", certHex, "value", totalValue, "limit", susLimit)

	return totalValue.Cmp(susLimit) == 1, nil
}

// Register registers the gRPC service.
func (s *GRPCServer) Register(grpcServer *grpc.Server) {
	nodev1.RegisterCertificateSubmissionServiceServer(grpcServer, s)
	nodev1.RegisterNodeStateServiceServer(grpcServer, s)
}

var emptyHash = [32]byte{}

func (s *GRPCServer) GetCertificateHeader(ctx context.Context, req *nodev1.GetCertificateHeaderRequest) (*nodev1.GetCertificateHeaderResponse, error) {
	requestId := req.GetCertificateId()
	fromStorage, err := s.service.db.GetCertificateById(requestId.Value.Value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Info("requested certificate not found in storage, checking upstream client", "id", requestId.Value.Value)
			return dialAndGetCertificateHeader(ctx, s.upstreamAggClientAddress, req)
		}

		return nil, fmt.Errorf("failed to get certificate by id: %w", err)
	}

	var resp *nodev1.GetCertificateHeaderResponse

	if fromStorage.ProcessedAt.Valid {
		// we have processed this certificate so pass it through to the base grpc server
		slog.Info("certificate has been processed, passing through to base grpc server", "id", requestId.Value.Value, "upstream", s.upstreamAggClientAddress)
		resp, err = dialAndGetCertificateHeader(ctx, s.upstreamAggClientAddress, req)
		if err != nil {
			return nil, fmt.Errorf("failed to get certificate header from base grpc server: %w", err)
		}
		slog.Info("successfully got certificate header from base grpc server", "requestId", requestId.Value.Value, "receivedId", resp.CertificateHeader.CertificateId.Value.Value, "upstream", s.upstreamAggClientAddress)
	} else {
		slog.Info("certificate has not been processed yet, returning pending state", "requestId", requestId.Value.Value)
		// this certificate has not been processed yet so return a pending state
		resp = &nodev1.GetCertificateHeaderResponse{
			CertificateHeader: &typesv1.CertificateHeader{
				Height:            1,
				CertificateId:     requestId,
				Status:            typesv1.CertificateStatus_CERTIFICATE_STATUS_PENDING,
				PrevLocalExitRoot: &interopv1.FixedBytes32{Value: emptyHash[:]},
				NewLocalExitRoot:  &interopv1.FixedBytes32{Value: emptyHash[:]},
				Metadata:          &interopv1.FixedBytes32{Value: emptyHash[:]},
			},
		}
	}

	return resp, nil
}

func (s *GRPCServer) GetLatestCertificateHeader(ctx context.Context, req *nodev1.GetLatestCertificateHeaderRequest) (*nodev1.GetLatestCertificateHeaderResponse, error) {
	// forward this straight on to the upstream client
	conn, err := grpc.NewClient(s.upstreamAggClientAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to upstream client at %s: %v", s.upstreamAggClientAddress, err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			slog.Error("failed to close gRPC connection to upstream client", "err", err)
		}
	}()

	client := v1.NewNodeStateServiceClient(conn)

	return client.GetLatestCertificateHeader(ctx, req)
}

func dialAndGetCertificateHeader(ctx context.Context, address string, req *nodev1.GetCertificateHeaderRequest) (*nodev1.GetCertificateHeaderResponse, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to upstream client at %s: %v", address, err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			slog.Error("failed to close gRPC connection to upstream client", "err", err)
		}
	}()

	client := v1.NewNodeStateServiceClient(conn)

	return client.GetCertificateHeader(ctx, req)
}

// bytesToUint64 converts big-endian bytes to uint64
func bytesToUint64(bytes []byte) uint64 {
	if len(bytes) == 0 {
		return 0
	}

	var result uint64
	// Start from the end of the byte array (for big-endian)
	startIdx := 0
	if len(bytes) > 8 {
		startIdx = len(bytes) - 8
	}

	for i := startIdx; i < len(bytes); i++ {
		result = (result << 8) | uint64(bytes[i])
	}

	return result
}

// extractMetadata extracts metadata from a certificate.
func extractMetadata(cert *typesv1.Certificate) map[string]interface{} {
	meta := make(map[string]interface{})
	if cert != nil {
		// Basic fields
		meta["height"] = cert.GetHeight()
		meta["network_id"] = cert.GetNetworkId()

		// Exit roots (convert to hex strings)
		if cert.GetPrevLocalExitRoot() != nil && cert.GetPrevLocalExitRoot().Value != nil {
			meta["prev_local_exit_root"] = fmt.Sprintf("0x%x", cert.GetPrevLocalExitRoot().Value)
		}
		if cert.GetNewLocalExitRoot() != nil && cert.GetNewLocalExitRoot().Value != nil {
			meta["new_local_exit_root"] = fmt.Sprintf("0x%x", cert.GetNewLocalExitRoot().Value)
		}

		// Bridge exits
		meta["bridge_exits_count"] = len(cert.GetBridgeExits())
		if len(cert.GetBridgeExits()) > 0 {
			bridgeExits := make([]map[string]interface{}, 0, len(cert.GetBridgeExits()))
			for _, be := range cert.GetBridgeExits() {
				if be != nil {
					beMap := map[string]interface{}{
						"dest_network": be.GetDestNetwork(),
					}
					if be.GetAmount() != nil && be.GetAmount().Value != nil {
						// Convert amount bytes to string
						amountUint := bytesToUint64(be.GetAmount().Value)
						beMap["amount"] = fmt.Sprintf("%d", amountUint)
					}
					if be.GetDestAddress() != nil && be.GetDestAddress().Value != nil {
						beMap["dest_address"] = fmt.Sprintf("0x%x", be.GetDestAddress().Value)
					}
					token := be.GetTokenInfo()
					if token != nil && token.OriginTokenAddress != nil && token.OriginTokenAddress.Value != nil {
						beMap["token_address"] = fmt.Sprintf("0x%x", token.OriginTokenAddress.Value)
					}
					bridgeExits = append(bridgeExits, beMap)
				}
			}
			meta["bridge_exits"] = bridgeExits
		}

		// Imported bridge exits
		meta["imported_bridge_exits_count"] = len(cert.GetImportedBridgeExits())
		if len(cert.GetImportedBridgeExits()) > 0 {
			importedExits := make([]map[string]interface{}, 0, len(cert.GetImportedBridgeExits()))
			for _, ibe := range cert.GetImportedBridgeExits() {
				if ibe != nil && ibe.GetBridgeExit() != nil {
					ibeMap := map[string]interface{}{
						"dest_network": ibe.GetBridgeExit().GetDestNetwork(),
					}
					if ibe.GetBridgeExit().GetAmount() != nil && ibe.GetBridgeExit().GetAmount().Value != nil {
						// Convert amount bytes to string
						amountUint := bytesToUint64(ibe.GetBridgeExit().GetAmount().Value)
						ibeMap["amount"] = fmt.Sprintf("%d", amountUint)
					}
					if ibe.GetBridgeExit().GetDestAddress() != nil && ibe.GetBridgeExit().GetDestAddress().Value != nil {
						ibeMap["dest_address"] = fmt.Sprintf("0x%x", ibe.GetBridgeExit().GetDestAddress().Value)
					}
					token := ibe.GetBridgeExit().GetTokenInfo()
					if token != nil && token.OriginTokenAddress != nil && token.OriginTokenAddress.Value != nil {
						ibeMap["token_address"] = fmt.Sprintf("0x%x", token.OriginTokenAddress.Value)
					}
					if ibe.GetGlobalIndex() != nil && ibe.GetGlobalIndex().Value != nil {
						ibeMap["global_index"] = fmt.Sprintf("0x%x", ibe.GetGlobalIndex().Value)
					}
					importedExits = append(importedExits, ibeMap)
				}
			}
			meta["imported_bridge_exits"] = importedExits
		}

		// Metadata field (if present)
		if cert.GetMetadata() != nil && cert.GetMetadata().Value != nil {
			meta["metadata"] = fmt.Sprintf("0x%x", cert.GetMetadata().Value)
		}

		// Custom chain data
		if cert.GetCustomChainData() != nil {
			meta["custom_chain_data"] = fmt.Sprintf("0x%x", cert.GetCustomChainData())
		}

		// L1 info tree leaf count (optional field)
		if cert.GetL1InfoTreeLeafCount() != 0 {
			meta["l1_info_tree_leaf_count"] = cert.GetL1InfoTreeLeafCount()
		}

		// Aggchain data
		if cert.GetAggchainData() != nil {
			aggchainMeta := make(map[string]interface{})

			// Handle the oneof field - either Signature or Generic (AggchainProof)
			if sig := cert.GetAggchainData().GetSignature(); sig != nil && sig.Value != nil {
				aggchainMeta["signature"] = fmt.Sprintf("0x%x", sig.Value)
			}

			if generic := cert.GetAggchainData().GetGeneric(); generic != nil {
				genericMeta := make(map[string]interface{})

				if generic.GetAggchainParams() != nil && generic.GetAggchainParams().Value != nil {
					genericMeta["aggchain_params"] = fmt.Sprintf("0x%x", generic.GetAggchainParams().Value)
				}

				if generic.GetSignature() != nil && generic.GetSignature().Value != nil {
					genericMeta["signature"] = fmt.Sprintf("0x%x", generic.GetSignature().Value)
				}

				if generic.GetContext() != nil {
					genericMeta["context_size"] = len(generic.GetContext())
				}

				// Handle SP1 stark proof if present
				if sp1 := generic.GetSp1Stark(); sp1 != nil {
					sp1Meta := make(map[string]interface{})
					sp1Meta["version"] = sp1.GetVersion()
					if sp1.GetProof() != nil {
						sp1Meta["proof"] = fmt.Sprintf("0x%x", sp1.GetProof())
					}
					if sp1.GetVkey() != nil {
						sp1Meta["vkey"] = fmt.Sprintf("0x%x", sp1.GetVkey())
					}
					genericMeta["sp1_stark"] = sp1Meta
				}

				aggchainMeta["generic"] = genericMeta
			}

			meta["aggchain_data"] = aggchainMeta
		}
	}
	return meta
}
