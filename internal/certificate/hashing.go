package certificate

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	interopv1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/interop/types/v1"
	typesv1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/types/v1"
)

func generateCertificateId(cert *typesv1.Certificate) *typesv1.CertificateId {
	var combinedExits []byte
	for _, bridgeExit := range cert.GetBridgeExits() {
		combinedExits = append(combinedExits, hashBridgeExit(bridgeExit)...)
	}
	hashedCombinedExits := crypto.Keccak256Hash(combinedExits).Bytes()

	var importedBridgeExits []byte
	for _, importedBridgeExit := range cert.GetImportedBridgeExits() {
		importedBridgeExits = append(importedBridgeExits, hashImportedBridgeExit(importedBridgeExit)...)
	}
	hashedImportedBridgeExits := crypto.Keccak256Hash(importedBridgeExits).Bytes()

	networkIdBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(networkIdBytes, cert.NetworkId)

	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, cert.Height)

	var finalHash []byte
	finalHash = append(finalHash, networkIdBytes...)
	finalHash = append(finalHash, heightBytes...)
	finalHash = append(finalHash, cert.PrevLocalExitRoot.Value...)
	finalHash = append(finalHash, cert.NewLocalExitRoot.Value...)
	finalHash = append(finalHash, hashedCombinedExits...)
	finalHash = append(finalHash, hashedImportedBridgeExits...)
	finalHash = append(finalHash, cert.Metadata.Value...)

	return &typesv1.CertificateId{
		Value: &interopv1.FixedBytes32{
			Value: crypto.Keccak256Hash(finalHash).Bytes(),
		},
	}
}

var EMPTY_METADATA_HASH = common.HexToHash("c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470")

func hashBridgeExit(b *interopv1.BridgeExit) []byte {
	// Create a buffer to hold all the data to be hashed
	var data []byte

	// 1. leaf_type as single byte
	data = append(data, byte(b.LeafType))

	// 2. origin_network as 4 bytes big-endian
	originNetworkBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(originNetworkBytes, uint32(b.TokenInfo.OriginNetwork))
	data = append(data, originNetworkBytes...)

	// 3. origin_token_address as 20 bytes
	data = append(data, b.TokenInfo.OriginTokenAddress.Value...)

	// 4. dest_network as 4 bytes big-endian
	destNetworkBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(destNetworkBytes, uint32(b.DestNetwork))
	data = append(data, destNetworkBytes...)

	// 5. dest_address as 20 bytes
	data = append(data, b.DestAddress.Value...)

	// 6. amount as 32 bytes big-endian
	data = append(data, b.Amount.Value...)

	// 7. metadata hash as 32 bytes (or EMPTY_METADATA_HASH if nil)
	if b.Metadata != nil {
		data = append(data, b.Metadata.Value...)
	} else {
		data = append(data, EMPTY_METADATA_HASH.Bytes()...)
	}

	// Hash the concatenated data using Keccak256
	return crypto.Keccak256Hash(data).Bytes()
}

func hashImportedBridgeExit(b *interopv1.ImportedBridgeExit) []byte {
	var data []byte
	data = append(data, hashBridgeExit(b.BridgeExit)...)

	var claimToHash []byte
	if b.GetMainnet() != nil {
		m := b.GetMainnet()

		var leafMer []byte
		leafMer = append(leafMer, m.ProofLeafMer.Root.Value...)
		for _, leaf := range m.ProofLeafMer.Siblings {
			leafMer = append(leafMer, leaf.Value...)
		}
		leafMerHash := crypto.Keccak256Hash(leafMer).Bytes()

		var proofGerL1Root []byte
		proofGerL1Root = append(proofGerL1Root, m.ProofGerL1Root.Root.Value...)
		for _, leaf := range m.ProofGerL1Root.Siblings {
			proofGerL1Root = append(proofGerL1Root, leaf.Value...)
		}
		proofGerL1RootHash := crypto.Keccak256Hash(proofGerL1Root).Bytes()

		var l1Leaf []byte
		gerBytes := append(m.L1Leaf.Mer.Value, m.L1Leaf.Rer.Value...)
		gerHash := crypto.Keccak256Hash(gerBytes).Bytes()
		l1Leaf = append(l1Leaf, gerHash...)
		l1Leaf = append(l1Leaf, m.L1Leaf.Inner.BlockHash.Value...)
		l1Leaf = append(l1Leaf, uint64ToBytes(m.L1Leaf.Inner.Timestamp)...)

		mainnetHash := crypto.Keccak256Hash(leafMerHash, proofGerL1RootHash, l1Leaf).Bytes()
		claimToHash = append(claimToHash, mainnetHash...)
	} else if b.GetRollup() != nil {
		r := b.GetRollup()

		var leafLer []byte
		leafLer = append(leafLer, r.ProofLeafLer.Root.Value...)
		for _, leaf := range r.ProofLeafLer.Siblings {
			leafLer = append(leafLer, leaf.Value...)
		}
		leafLerHash := crypto.Keccak256Hash(leafLer).Bytes()

		var proofLerRer []byte
		proofLerRer = append(proofLerRer, r.ProofLerRer.Root.Value...)
		for _, leaf := range r.ProofLerRer.Siblings {
			proofLerRer = append(proofLerRer, leaf.Value...)
		}
		proofLerRerHash := crypto.Keccak256Hash(proofLerRer).Bytes()

		var proofGer []byte
		proofGer = append(proofGer, r.ProofGerL1Root.Root.Value...)
		for _, leaf := range r.ProofGerL1Root.Siblings {
			proofGer = append(proofGer, leaf.Value...)
		}
		proofGerHash := crypto.Keccak256Hash(proofGer).Bytes()

		var l1Leaf []byte
		gerBytes := append(r.L1Leaf.Mer.Value, r.L1Leaf.Rer.Value...)
		gerHash := crypto.Keccak256Hash(gerBytes).Bytes()
		l1Leaf = append(l1Leaf, gerHash...)
		l1Leaf = append(l1Leaf, r.L1Leaf.Inner.BlockHash.Value...)
		l1Leaf = append(l1Leaf, uint64ToBytes(r.L1Leaf.Inner.Timestamp)...)

		rollupHash := crypto.Keccak256Hash(leafLerHash, proofLerRerHash, proofGerHash, l1Leaf).Bytes()
		claimToHash = append(claimToHash, rollupHash...)
	}

	data = append(data, claimToHash...)

	var globalIndex []byte
	globalIndex = append(globalIndex, b.GlobalIndex.Value...)
	globalIndexHash := crypto.Keccak256Hash(globalIndex).Bytes()
	data = append(data, globalIndexHash...)

	return crypto.Keccak256Hash(data).Bytes()
}

func uint64ToBytes(value uint64) []byte {
	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, value)
	return bytes
}
