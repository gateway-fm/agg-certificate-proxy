package certificate

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	typesv1 "github.com/gateway-fm/agg-certificate-proxy/pkg/proto/agglayer/node/types/v1"
)

func Test_generateCertificateId_h0(t *testing.T) {
	contents, err := os.ReadFile("vectors/n15-cert_h0.json")
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	var cert typesv1.Certificate
	err = json.Unmarshal(contents, &cert)
	if err != nil {
		t.Fatalf("failed to unmarshal file: %v", err)
	}

	certId := generateCertificateId(&cert)

	expectedCertId := common.HexToHash("0x30b52dba403e8799952b0ed038183149c289fea13efa5e38787c263567346eca").Bytes()

	if !bytes.Equal(certId.Value.Value, expectedCertId) {
		t.Fatalf("expected cert id to be %s, got %s", expectedCertId, certId)
	}
}
