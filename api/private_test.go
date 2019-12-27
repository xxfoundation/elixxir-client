package api

import (
	"bytes"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"reflect"
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

//Happy path: test it generates key when passed nil
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

//Tests it generates keys based on an existing privateKey
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
}

//Tests GenerateCmixKeys cases
func TestGenerateCmixKeys(t *testing.T) {
	//Test generateCmixKeys
	cmixGrp, _ := generateGroups(def)
	cmixPrivKey, _, err := generateCmixKeys(cmixGrp)
	if err != nil {
		t.Errorf("%+v", err)
	}

	if !csprng.InGroup(cmixPrivKey.Bytes(), cmixGrp.GetPBytes()) {
		t.Errorf("Generated cmix private key is not in the cmix group!")
	}
	//Error case
	_, _, err = generateCmixKeys(nil)
	if err == nil {
		t.Errorf("Expected error case, should not pass nil into GenerateCmixKeys()")
	}

}

//
func TestGenerateE2eKeys(t *testing.T) {
	//Test generateCmixKeys
	cmixGrp, e2eGrp := generateGroups(def)

	//Test e2e key generation
	e2ePrivKey, _, err := generateE2eKeys(cmixGrp, e2eGrp)
	if err != nil {
		t.Errorf("%+v", err)
	}

	if !csprng.InGroup(e2ePrivKey.Bytes(), cmixGrp.GetPBytes()) {
		t.Errorf("Generated cmix private key is not in the cmix group!")
	}

	//Error case
	_, _, err = generateE2eKeys(nil, nil)
	if err == nil {
		t.Errorf("Expected error case, should not pass nil into GenerateE2eKeys()")
	}

}

//Happy path: tests that it generates a user and puts in the registry
func TestGenerateUserInformation_EmptyNick(t *testing.T) {
	grp, _ := generateGroups(def)
	user.InitUserRegistry(grp)
	_, pubkey, _ := generateRsaKeys(nil)
	_, uid, usr, err := generateUserInformation(pubkey)
	if err != nil {
		t.Errorf("%+v", err)
	}
	retrievedUser, ok := user.Users.GetUser(uid)
	if !ok {
		t.Errorf("UserId not inserted into registry")
	}

	if !reflect.DeepEqual(usr, retrievedUser) {
		t.Errorf("Did not retrieve correct user. \n\treceived: %v\n\texpected: %v", retrievedUser, usr)
	}

	if usr.Username != "" {
		t.Errorf("User's username should be initally")
	}

}

//Happy path: test GenerateUser with a nickname and puts in registry
func TestGenerateUserInformation(t *testing.T) {
	grp, _ := generateGroups(def)
	user.InitUserRegistry(grp)
	nickName := "test"
	_, pubkey, _ := generateRsaKeys(nil)
	_, uid, usr, err := generateUserInformation(pubkey)
	usr.Username = nickName
	if err != nil {
		t.Errorf("%+v", err)
	}

	retrievedUser, ok := user.Users.GetUser(uid)
	if !ok {
		t.Errorf("UserId not inserted into registry")
	}
	if !reflect.DeepEqual(usr, retrievedUser) {
		t.Errorf("Did not retrieve correct user. \n\treceived: %v\n\texpected: %v", retrievedUser, usr)
	}

	if usr.Username != nickName {
		t.Errorf("User's nickname was overwritten\nreceived: %v\n\texpected: %v", usr.Username, nickName)
	}

}
