package utility

import (
	"bytes"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"testing"
)

// Smoke test
func TestNewOrLoadSalt(t *testing.T) {
	kv := ekv.MakeMemstore()
	vkv := versioned.NewKV(kv)

	rng := csprng.NewSystemRNG()

	_, err := NewOrLoadSalt(vkv, rng)
	if err != nil {
		t.Fatalf("NewOrLoadSalt error: %+v", err)
	}

}

// Test that calling NewOrLoadSalt twice returns the same
// salt that exists in storage.
func TestLoadSalt(t *testing.T) {

	kv := ekv.MakeMemstore()
	vkv := versioned.NewKV(kv)

	rng := csprng.NewSystemRNG()

	original, err := NewOrLoadSalt(vkv, rng)
	if err != nil {
		t.Fatalf("NewOrLoadSalt error: %+v", err)
	}

	loaded, err := NewOrLoadSalt(vkv, rng)
	if err != nil {
		t.Fatalf("NewOrLoadSalt error: %+v", err)
	}

	// Test that loaded matches the original (ie a new one was not generated)
	if !bytes.Equal(original, loaded) {
		t.Fatalf("Failed to load salt.")
	}

}
