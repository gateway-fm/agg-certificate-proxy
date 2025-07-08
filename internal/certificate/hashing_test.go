package certificate

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func Test_generateCertificateId_h0(t *testing.T) {
	tests := []struct {
		filename     string
		expectedHash string
	}{
		{
			filename:     "vectors/n15-cert_h0.json",
			expectedHash: "0x30b52dba403e8799952b0ed038183149c289fea13efa5e38787c263567346eca",
		},
		{
			filename:     "vectors/n15-cert_h1.json",
			expectedHash: "0x3b1aa52ca4ffec8d1f05cdafa4be5156cb4485cc3f84fac08b83c88db0eca7db",
		},
		{
			filename:     "vectors/n15-cert_h2.json",
			expectedHash: "0x420d1169f49b212e558409ed170d852250a2c5f32195cbcaff2ca4fd5da837ef",
		},
		{
			filename:     "vectors/n15-cert_h3.json",
			expectedHash: "0x17802d6545ef138a88e99ac7da27bebadbeab67ccccedde5114e01cd0f970d5f",
		},
	}

	for _, test := range tests {
		t.Run(test.filename, func(t *testing.T) {
			cert, err := LoadCertificateFromJSONFile(test.filename)
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}
			expectedCertId := common.HexToHash(test.expectedHash).Bytes()
			certId := generateCertificateId(cert)
			if !bytes.Equal(certId.Value.Value, expectedCertId) {
				t.Fatalf("expected cert id to be %x, got %x", expectedCertId, certId.Value.Value)
			}
		})
	}
}
