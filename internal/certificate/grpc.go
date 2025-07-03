package certificate

import (
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	interopv1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/interop/types/v1"
	typesv1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/types/v1"
	nodev1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/v1"
	"log/slog"
)

// GRPCServer handles incoming gRPC requests for certificate submission.
type GRPCServer struct {
	nodev1.UnimplementedCertificateSubmissionServiceServer
	service *Service
}

// NewGRPCServer creates a new gRPC server.
func NewGRPCServer(service *Service) *GRPCServer {
	return &GRPCServer{service: service}
}

// SubmitCertificate handles the submission of a new certificate.
func (s *GRPCServer) SubmitCertificate(ctx context.Context, req *nodev1.SubmitCertificateRequest) (*nodev1.SubmitCertificateResponse, error) {
	slog.Info("received certificate submission request")

	// Marshal the request to store it as a blob
	rawProto, err := proto.Marshal(req.Certificate)
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

	if isDelayed {
		slog.Info("network is on the delay list. Storing certificate for delayed processing.", "network", networkID)
		if err := s.service.StoreCertificate(rawProto, string(metadataJson)); err != nil {
			return nil, fmt.Errorf("failed to store certificate: %w", err)
		}
	} else {
		slog.Info("network is not on the delay list. Sending certificate straight through.", "network", networkID)
		// Send immediately
		cert := Certificate{ID: 0, RawProto: rawProto, Metadata: string(metadataJson)}
		if err := s.service.SendToAggSender(cert); err != nil {
			slog.Error("failed to send certificate immediately", "err", err)
			return nil, fmt.Errorf("failed to send certificate immediately: %w", err)
		}
		slog.Info("successfully sent certificate for network immediately", "network", networkID)
	}

	return &nodev1.SubmitCertificateResponse{
		CertificateId: &typesv1.CertificateId{
			Value: &interopv1.FixedBytes32{
				Value: []byte("certificate-processed-id"), // Dummy ID
			},
		},
	}, nil
}

// Register registers the gRPC service.
func (s *GRPCServer) Register(grpcServer *grpc.Server) {
	nodev1.RegisterCertificateSubmissionServiceServer(grpcServer, s)
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
