package certificate

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	interopv1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/interop/types/v1"
	typesv1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/types/v1"
)

// LoadCertificateFromJSONFile reads a JSON file and converts it to a protobuf Certificate
func LoadCertificateFromJSONFile(filename string) (*typesv1.Certificate, error) {
	contents, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return CertificateFromJSON(contents)
}

// CertificateFromJSON unmarshals a JSON certificate into a protobuf Certificate type
func CertificateFromJSON(jsonData []byte) (*typesv1.Certificate, error) {
	// First unmarshal into a map to handle dynamic conversion
	var raw map[string]interface{}
	if err := json.Unmarshal(jsonData, &raw); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	cert := &typesv1.Certificate{}

	// Basic fields
	if networkID, ok := raw["network_id"].(float64); ok {
		cert.NetworkId = uint32(networkID)
	}

	if height, ok := raw["height"].(float64); ok {
		cert.Height = uint64(height)
	}

	// Convert hex string fields to FixedBytes32
	if prevRoot, ok := raw["prev_local_exit_root"].(string); ok {
		if bytes, err := hexStringToBytes(prevRoot); err == nil {
			cert.PrevLocalExitRoot = &interopv1.FixedBytes32{Value: bytes}
		}
	}

	if newRoot, ok := raw["new_local_exit_root"].(string); ok {
		if bytes, err := hexStringToBytes(newRoot); err == nil {
			cert.NewLocalExitRoot = &interopv1.FixedBytes32{Value: bytes}
		}
	}

	// Handle metadata if present
	if metadata, ok := raw["metadata"].(string); ok && metadata != "" && metadata != "null" {
		if bytes, err := hexStringToBytes(metadata); err == nil {
			cert.Metadata = &interopv1.FixedBytes32{Value: bytes}
		}
	}

	// Handle l1_info_tree_leaf_count if present
	if l1Count, ok := raw["l1_info_tree_leaf_count"].(float64); ok {
		count := uint32(l1Count)
		cert.L1InfoTreeLeafCount = &count
	}

	// Handle bridge_exits
	if bridgeExits, ok := raw["bridge_exits"].([]interface{}); ok {
		for _, exitRaw := range bridgeExits {
			if exitMap, ok := exitRaw.(map[string]interface{}); ok {
				bridgeExit, err := mapToBridgeExit(exitMap)
				if err != nil {
					return nil, fmt.Errorf("failed to convert bridge exit: %w", err)
				}
				cert.BridgeExits = append(cert.BridgeExits, bridgeExit)
			}
		}
	}

	// Handle imported_bridge_exits
	if importedExits, ok := raw["imported_bridge_exits"].([]interface{}); ok {
		for _, exitRaw := range importedExits {
			if exitMap, ok := exitRaw.(map[string]interface{}); ok {
				importedExit, err := mapToImportedBridgeExit(exitMap)
				if err != nil {
					return nil, fmt.Errorf("failed to convert imported bridge exit: %w", err)
				}
				cert.ImportedBridgeExits = append(cert.ImportedBridgeExits, importedExit)
			}
		}
	}

	return cert, nil
}

// hexStringToBytes converts a hex string (with or without 0x prefix) to bytes
func hexStringToBytes(hexStr string) ([]byte, error) {
	if hexStr == "" || hexStr == "null" {
		return nil, fmt.Errorf("empty hex string")
	}

	// Remove 0x prefix if present
	hexStr = strings.TrimPrefix(hexStr, "0x")

	// prefix a 0 for odd length
	if len(hexStr)%2 == 1 {
		hexStr = "0" + hexStr
	}

	return hex.DecodeString(hexStr)
}

// mapToBridgeExit converts a map to a BridgeExit protobuf
func mapToBridgeExit(m map[string]interface{}) (*interopv1.BridgeExit, error) {
	exit := &interopv1.BridgeExit{}

	// leaf_type
	if leafType, ok := m["leaf_type"].(string); ok {
		switch leafType {
		case "Transfer":
			exit.LeafType = interopv1.LeafType_LEAF_TYPE_TRANSFER
		case "Message":
			exit.LeafType = interopv1.LeafType_LEAF_TYPE_MESSAGE
		default:
			exit.LeafType = interopv1.LeafType_LEAF_TYPE_UNSPECIFIED
		}
	}

	// token_info
	if tokenInfo, ok := m["token_info"].(map[string]interface{}); ok {
		exit.TokenInfo = &interopv1.TokenInfo{}

		if originNetwork, ok := tokenInfo["origin_network"].(float64); ok {
			exit.TokenInfo.OriginNetwork = uint32(originNetwork)
		}

		if originTokenAddr, ok := tokenInfo["origin_token_address"].(string); ok {
			if bytes, err := hexStringToBytes(originTokenAddr); err == nil {
				exit.TokenInfo.OriginTokenAddress = &interopv1.FixedBytes20{Value: bytes}
			}
		}
	}

	// dest_network
	if destNetwork, ok := m["dest_network"].(float64); ok {
		exit.DestNetwork = uint32(destNetwork)
	}

	// dest_address
	if destAddr, ok := m["dest_address"].(string); ok {
		if bytes, err := hexStringToBytes(destAddr); err == nil {
			exit.DestAddress = &interopv1.FixedBytes20{Value: bytes}
		}
	}

	// amount
	if amount, ok := m["amount"].(string); ok {
		if bytes, err := hexStringToBytes(amount); err == nil {
			exit.Amount = &interopv1.FixedBytes32{Value: bytes}
		}
	}

	// metadata (optional)
	if metadata, ok := m["metadata"].(string); ok {
		if metadata != "" && metadata != "null" {
			if bytes, err := hexStringToBytes(metadata); err == nil {
				exit.Metadata = &interopv1.FixedBytes32{Value: bytes}
			}
		}
	}

	return exit, nil
}

// mapToImportedBridgeExit converts a map to an ImportedBridgeExit protobuf
func mapToImportedBridgeExit(m map[string]interface{}) (*interopv1.ImportedBridgeExit, error) {
	imported := &interopv1.ImportedBridgeExit{}

	// bridge_exit
	if bridgeExit, ok := m["bridge_exit"].(map[string]interface{}); ok {
		exit, err := mapToBridgeExit(bridgeExit)
		if err != nil {
			return nil, fmt.Errorf("failed to convert bridge exit: %w", err)
		}
		imported.BridgeExit = exit
	}

	// global_index
	if globalIndex, ok := m["global_index"].(map[string]interface{}); ok {
		// Convert the global_index structure
		if mainnetFlag, ok := globalIndex["mainnet_flag"].(bool); ok {
			if rollupIndex, ok := globalIndex["rollup_index"].(float64); ok {
				if leafIndex, ok := globalIndex["leaf_index"].(float64); ok {
					// Create a 32-byte array initialized to zeros
					bytes := make([]byte, 32)

					// Convert leaf_index to little-endian bytes and copy to bytes[0:4]
					binary.LittleEndian.PutUint32(bytes[0:4], uint32(leafIndex))

					// Convert rollup_index to little-endian bytes and copy to bytes[4:8]
					binary.LittleEndian.PutUint32(bytes[4:8], uint32(rollupIndex))

					// If mainnet_flag is true, set the least significant bit of byte 8
					if mainnetFlag {
						bytes[8] |= 0x01
					}

					hashed := crypto.Keccak256Hash(bytes).Bytes()

					imported.GlobalIndex = &interopv1.FixedBytes32{Value: hashed}
				}
			}
		}
	}

	// claim_data
	if claimData, ok := m["claim_data"].(map[string]interface{}); ok {
		// Handle Mainnet claim
		if mainnetClaim, ok := claimData["Mainnet"].(map[string]interface{}); ok {
			claim, err := mapToClaimFromMainnet(mainnetClaim)
			if err != nil {
				return nil, fmt.Errorf("failed to convert mainnet claim: %w", err)
			}
			imported.Claim = &interopv1.ImportedBridgeExit_Mainnet{Mainnet: claim}
		}
		// Handle Rollup claim if needed
		if rollupClaim, ok := claimData["Rollup"].(map[string]interface{}); ok {
			claim, err := mapToClaimFromRollup(rollupClaim)
			if err != nil {
				return nil, fmt.Errorf("failed to convert rollup claim: %w", err)
			}
			imported.Claim = &interopv1.ImportedBridgeExit_Rollup{Rollup: claim}
		}
	}

	return imported, nil
}

// mapToClaimFromMainnet converts a map to a ClaimFromMainnet protobuf
func mapToClaimFromMainnet(m map[string]interface{}) (*interopv1.ClaimFromMainnet, error) {
	claim := &interopv1.ClaimFromMainnet{}

	// proof_leaf_mer
	if proofLeafMer, ok := m["proof_leaf_mer"].(map[string]interface{}); ok {
		proof, err := mapToMerkleProof(proofLeafMer)
		if err != nil {
			return nil, fmt.Errorf("failed to convert proof_leaf_mer: %w", err)
		}
		claim.ProofLeafMer = proof
	}

	// proof_ger_l1root
	if proofGerL1Root, ok := m["proof_ger_l1root"].(map[string]interface{}); ok {
		proof, err := mapToMerkleProof(proofGerL1Root)
		if err != nil {
			return nil, fmt.Errorf("failed to convert proof_ger_l1root: %w", err)
		}
		claim.ProofGerL1Root = proof
	}

	// l1_leaf
	if l1Leaf, ok := m["l1_leaf"].(map[string]interface{}); ok {
		leaf, err := mapToL1InfoTreeLeafWithContext(l1Leaf)
		if err != nil {
			return nil, fmt.Errorf("failed to convert l1_leaf: %w", err)
		}
		claim.L1Leaf = leaf
	}

	return claim, nil
}

// mapToClaimFromRollup converts a map to a ClaimFromRollup protobuf
func mapToClaimFromRollup(m map[string]interface{}) (*interopv1.ClaimFromRollup, error) {
	claim := &interopv1.ClaimFromRollup{}

	// proof_leaf_ler
	if proofLeafLer, ok := m["proof_leaf_ler"].(map[string]interface{}); ok {
		proof, err := mapToMerkleProof(proofLeafLer)
		if err != nil {
			return nil, fmt.Errorf("failed to convert proof_leaf_ler: %w", err)
		}
		claim.ProofLeafLer = proof
	}

	// proof_ler_rer
	if proofLerRer, ok := m["proof_ler_rer"].(map[string]interface{}); ok {
		proof, err := mapToMerkleProof(proofLerRer)
		if err != nil {
			return nil, fmt.Errorf("failed to convert proof_ler_rer: %w", err)
		}
		claim.ProofLerRer = proof
	}

	// proof_ger_l1root
	if proofGerL1Root, ok := m["proof_ger_l1root"].(map[string]interface{}); ok {
		proof, err := mapToMerkleProof(proofGerL1Root)
		if err != nil {
			return nil, fmt.Errorf("failed to convert proof_ger_l1root: %w", err)
		}
		claim.ProofGerL1Root = proof
	}

	// l1_leaf
	if l1Leaf, ok := m["l1_leaf"].(map[string]interface{}); ok {
		leaf, err := mapToL1InfoTreeLeafWithContext(l1Leaf)
		if err != nil {
			return nil, fmt.Errorf("failed to convert l1_leaf: %w", err)
		}
		claim.L1Leaf = leaf
	}

	return claim, nil
}

// mapToMerkleProof converts a map to a MerkleProof protobuf
func mapToMerkleProof(m map[string]interface{}) (*interopv1.MerkleProof, error) {
	proof := &interopv1.MerkleProof{}

	// root is at the top level
	if root, ok := m["root"].(string); ok {
		if bytes, err := hexStringToBytes(root); err == nil {
			proof.Root = &interopv1.FixedBytes32{Value: bytes}
		}
	}

	// proof map contains siblings
	if proofData, ok := m["proof"].(map[string]interface{}); ok {
		// siblings
		if siblings, ok := proofData["siblings"].([]interface{}); ok {
			for _, siblingRaw := range siblings {
				if sibling, ok := siblingRaw.(string); ok {
					if bytes, err := hexStringToBytes(sibling); err == nil {
						proof.Siblings = append(proof.Siblings, &interopv1.FixedBytes32{Value: bytes})
					}
				}
			}
		}
	}

	return proof, nil
}

// mapToL1InfoTreeLeafWithContext converts a map to a L1InfoTreeLeafWithContext protobuf
func mapToL1InfoTreeLeafWithContext(m map[string]interface{}) (*interopv1.L1InfoTreeLeafWithContext, error) {
	leaf := &interopv1.L1InfoTreeLeafWithContext{}

	// l1_info_tree_index
	if index, ok := m["l1_info_tree_index"].(float64); ok {
		leaf.L1InfoTreeIndex = uint32(index)
	}

	// rer
	if rer, ok := m["rer"].(string); ok {
		if bytes, err := hexStringToBytes(rer); err == nil {
			leaf.Rer = &interopv1.FixedBytes32{Value: bytes}
		}
	}

	// mer
	if mer, ok := m["mer"].(string); ok {
		if bytes, err := hexStringToBytes(mer); err == nil {
			leaf.Mer = &interopv1.FixedBytes32{Value: bytes}
		}
	}

	// inner
	if inner, ok := m["inner"].(map[string]interface{}); ok {
		innerLeaf := &interopv1.L1InfoTreeLeaf{}

		if globalExitRoot, ok := inner["global_exit_root"].(string); ok {
			if bytes, err := hexStringToBytes(globalExitRoot); err == nil {
				innerLeaf.GlobalExitRoot = &interopv1.FixedBytes32{Value: bytes}
			}
		}

		if blockHash, ok := inner["block_hash"].(string); ok {
			if bytes, err := hexStringToBytes(blockHash); err == nil {
				innerLeaf.BlockHash = &interopv1.FixedBytes32{Value: bytes}
			}
		}

		if timestamp, ok := inner["timestamp"].(float64); ok {
			innerLeaf.Timestamp = uint64(timestamp)
		}

		leaf.Inner = innerLeaf
	}

	return leaf, nil
}
