package certificate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	nodev1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/v1"
	"google.golang.org/protobuf/proto"
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
		{
			filename:     "vectors/n15-cert_h4.json",
			expectedHash: "0x8a7c22eb1c0b17afe66c7b47283798b4b10a2853ed11d15073ae8fed1a03caf6",
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

// used as a debug tool to load up raw proto from a database file and inspect the data there
func Test_generateCertificateId_FromDatabaseFile(t *testing.T) {
	db, err := NewSqliteStore("../../certificates.db")
	if err != nil {
		// this test is only for debugging so no problem if the database is not available
		// just return and skip the test
		return
	}
	defer db.Close()

	certs, err := db.GetCertificates()
	if err != nil {
		t.Fatalf("failed to get certificates: %v", err)
	}

	var huntId int64 = 4

	for _, cert := range certs {
		if cert.ID != huntId {
			continue
		}
		certProto := &nodev1.SubmitCertificateRequest{}
		err = proto.Unmarshal(cert.RawProto, certProto)
		if err != nil {
			t.Fatalf("failed to parse certificate: %v", err)
		}
		certId := generateCertificateId(certProto.Certificate)
		fmt.Printf("cert id: %x\n", certId.Value.Value)

		certAsJson, err := json.Marshal(certProto)
		if err != nil {
			t.Fatalf("failed to marshal certificate: %v", err)
		}
		os.WriteFile("cert.json", certAsJson, 0644)
	}
}
