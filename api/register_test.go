package api

import (
	"bytes"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"testing"
)

type CountingReader struct {
	count uint8
}

// Read just counts until 254 then starts over again
func (c *CountingReader) Read(b []byte) (int, error) {
	for i := 0; i < len(b); i++ {
		c.count = (c.count + 1) % 255
		b[i] = c.count
	}
	return len(b), nil
}

//
func TestGenerateKeys_NilPrivateKey(t *testing.T) {
	privKey, pubKey, err := generateRsaKeys(nil)
	if privKey == nil {
		t.Errorf("Failed to generate private key when generateRsaKeys() is passed nil")
	}
	if pubKey == nil {
		t.Errorf("Failed to pull public key from private key")
	}

	if err != nil {
		t.Errorf("%+v", err)
	}
}

//
func TestGenerateKeys(t *testing.T) {
	notRand := &CountingReader{count: uint8(0)}

	privKey, err := rsa.GenerateKey(notRand, 1024)
	if err != nil {
		t.Errorf("%+v", err)
	}
	expected_N := privKey.N.Bytes()
	privKey, pubKey, err := generateRsaKeys(privKey)
	if err != nil {
		t.Errorf("Failecd to generate keys: %+v", err)
	}
	if bytes.Compare(expected_N, privKey.N.Bytes()) != 0 {
		t.Errorf("Private key overwritten in generateKeys() despite privateKey not being nil")
	}

	if bytes.Compare(pubKey.GetN().Bytes(), expected_N) != 0 {
		t.Logf("N: %v", pubKey.GetN().Bytes())
		t.Errorf("Bad N-val, expected: %v", expected_N)
	}
	//TODO: Add more checks here
}

func TestGenerateCmixKeys(t *testing.T) {
	cmixGrp, _ := generateGroups(def)

	cmixPrivKey, _, err := generateCmixKeys(cmixGrp)
	if err != nil {
		t.Errorf("%+v", err)
	}

	if !csprng.InGroup(cmixPrivKey.Bytes(), cmixGrp.GetPBytes()) {
		t.Errorf("Generated cmix private key is not in the cmix group!")
	}
}
