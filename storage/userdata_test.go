package storage

import (
	"bytes"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"reflect"
	"testing"
)

// Test committing/retrieving userdata struct
func TestSession_CommitUserData(t *testing.T) {
	rsaPrivateKey, err := rsa.GenerateKey(rand.New(rand.NewSource(0)), 64)
	if err != nil {
		t.Fatal(err)
	}
	// These don't have to represent actual data because they're just stored and retrieved
	cmixGrp := cyclic.NewGroup(large.NewInt(53), large.NewInt(2))
	e2eGrp := cyclic.NewGroup(large.NewInt(53), large.NewInt(2))
	expectedData := &UserData{
		ThisUser: &User{
			User:     id.NewIdFromUInt(5, id.User, t),
			Username: "ted",
			Precan:   true,
		},
		RSAPrivateKey:    rsaPrivateKey,
		RSAPublicKey:     rsaPrivateKey.GetPublic(),
		CMIXDHPrivateKey: cmixGrp.NewInt(3),
		CMIXDHPublicKey:  cmixGrp.NewInt(4),
		E2EDHPrivateKey:  e2eGrp.NewInt(5),
		E2EDHPublicKey:   e2eGrp.NewInt(6),
		CmixGrp:          cmixGrp,
		E2EGrp:           e2eGrp,
		Salt:             []byte("potassium permanganate"),
	}

	// Create a session backed by memory
	store := make(ekv.Memstore)
	vkv := NewVersionedKV(store)
	session := Session{kv: vkv}
	err = session.CommitUserData(expectedData)
	if err != nil {
		t.Fatal(err)
	}
	retrievedData, err := session.GetUserData()
	if err != nil {
		t.Fatal(err)
	}

	// Field by field comparison
	if !retrievedData.ThisUser.User.Cmp(expectedData.ThisUser.User) {
		t.Error("User IDs didn't match")
	}
	if retrievedData.ThisUser.Precan != expectedData.ThisUser.Precan {
		t.Error("User precan didn't match")
	}
	if retrievedData.ThisUser.Username != expectedData.ThisUser.Username {
		t.Error("User names didn't match")
	}
	if retrievedData.CMIXDHPublicKey.Cmp(expectedData.CMIXDHPublicKey) != 0 {
		t.Error("cmix DH public key didn't match")
	}
	if retrievedData.CMIXDHPrivateKey.Cmp(expectedData.CMIXDHPrivateKey) != 0 {
		t.Error("cmix DH private key didn't match")
	}
	if retrievedData.E2EDHPrivateKey.Cmp(expectedData.E2EDHPrivateKey) != 0 {
		t.Error("e2e DH private key didn't match")
	}
	if retrievedData.E2EDHPublicKey.Cmp(expectedData.E2EDHPublicKey) != 0 {
		t.Error("e2e DH public key didn't match")
	}
	if !reflect.DeepEqual(retrievedData.CmixGrp, expectedData.CmixGrp) {
		t.Error("cmix groups didn't match")
	}
	if !reflect.DeepEqual(retrievedData.E2EGrp, expectedData.E2EGrp) {
		t.Error("e2e groups didn't match")
	}
	if retrievedData.RSAPrivateKey.D.Cmp(expectedData.RSAPrivateKey.D) != 0 {
		t.Error("rsa D doesn't match")
	}
	if !bytes.Equal(retrievedData.Salt, expectedData.Salt) {
		t.Error("salts don't match")
	}
}
