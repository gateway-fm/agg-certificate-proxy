package certificate

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	interopv1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/interop/types/v1"
	typesv1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/types/v1"
)

func generateCertificateId(cert *typesv1.Certificate) *typesv1.CertificateId {
	var combinedExits [][]byte
	for _, bridgeExit := range cert.GetBridgeExits() {
		hash := hashBridgeExit(bridgeExit)
		combinedExits = append(combinedExits, hash)
	}
	hashedCombinedExits := crypto.Keccak256Hash(combinedExits...).Bytes()

	var importedBridgeExits [][]byte
	for _, importedBridgeExit := range cert.GetImportedBridgeExits() {
		hash := hashImportedBridgeExit(importedBridgeExit)
		importedBridgeExits = append(importedBridgeExits, hash)
	}
	hashedImportedBridgeExits := crypto.Keccak256Hash(importedBridgeExits...).Bytes()

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
	if b.LeafType == interopv1.LeafType_LEAF_TYPE_TRANSFER {
		data = append(data, byte(0))
	} else {
		data = append(data, byte(1))
	}

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
	amountBytes := make([]byte, 32)
	// Copy the big endian bytes to the right side of the 32-byte array (left-padded with zeros)
	if len(b.Amount.Value) <= 32 {
		copy(amountBytes[32-len(b.Amount.Value):], b.Amount.Value)
	} else {
		// If longer than 32 bytes, take the rightmost 32 bytes
		copy(amountBytes, b.Amount.Value[len(b.Amount.Value)-32:])
	}
	data = append(data, amountBytes...)

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

		combinedSiblings := make([]byte, 0, len(m.ProofLeafMer.Siblings)*32)
		for _, leaf := range m.ProofLeafMer.Siblings {
			combinedSiblings = append(combinedSiblings, leaf.Value...)
		}
		leafMerHash := crypto.Keccak256Hash(m.ProofLeafMer.Root.Value, combinedSiblings).Bytes()

		combinedSiblings = make([]byte, 0, len(m.ProofGerL1Root.Siblings)*32)
		for _, leaf := range m.ProofGerL1Root.Siblings {
			combinedSiblings = append(combinedSiblings, leaf.Value...)
		}
		proofGerL1RootHash := crypto.Keccak256Hash(m.ProofGerL1Root.Root.Value, combinedSiblings).Bytes()

		gerHash := crypto.Keccak256Hash(m.L1Leaf.Mer.Value, m.L1Leaf.Rer.Value).Bytes()
		l1LeafHash := crypto.Keccak256Hash(
			gerHash,
			m.L1Leaf.Inner.BlockHash.Value,
			uint64ToBytes(m.L1Leaf.Inner.Timestamp),
		).Bytes()

		mainnetHash := crypto.Keccak256Hash(leafMerHash, proofGerL1RootHash, l1LeafHash).Bytes()
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

	data = append(data, b.GlobalIndex.Value...)

	return crypto.Keccak256Hash(data).Bytes()
}

func uint64ToBytes(value uint64) []byte {
	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, value)
	return bytes
}
