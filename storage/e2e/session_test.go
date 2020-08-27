package e2e

import (
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/csprng"
	dh "gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/ekv"
	"testing"
)

func TestSession_generate_noPrivateKeyReceive(t *testing.T) {

	grp := getGroup()
	rng := csprng.NewSystemRNG()
	partnerPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, rng)
	partnerPubKey := dh.GeneratePublicKey(partnerPrivKey, grp)

	//create context objects for general use
	fps := newFingerprints()
	ctx := &context{
		fa:  &fps,
		grp: grp,
		kv:  versioned.NewKV(make(ekv.Memstore)),
	}

	//build the session
	s := &Session{
		partnerPubKey: partnerPubKey,
		params:        GetDefaultSessionParams(),
		manager: &Manager{
			ctx: ctx,
		},
		t: Receive,
	}

	//run the generate command
	s.generate()

	//check that it generated a private key
	if s.myPrivKey == nil {
		t.Errorf("Public key was not generated when missing")
	}

	//verify the basekey is correct
	expectedBaseKey := dh.GenerateSessionKey(s.myPrivKey, s.partnerPubKey, grp)

	if expectedBaseKey.Cmp(s.baseKey) != 0 {
		t.Errorf("generated base key does not match expected base key")
	}

	//verify the ttl was generated
	if s.ttl == 0 {
		t.Errorf("ttl not generated")
	}

	//verify keystates where created
	if s.keyState == nil {
		t.Errorf("keystates not generated")
	}

	//verify keys were registered in the fingerprintMap
	for keyNum := uint32(0); keyNum < s.keyState.numkeys; keyNum++ {
		key := newKey(s, keyNum)
		if _, ok := fps.toKey[key.Fingerprint()]; !ok {
			t.Errorf("key %v not in fingerprint map", keyNum)
		}
	}
}

func TestSession_generate_PrivateKeySend(t *testing.T) {

	grp := getGroup()
	rng := csprng.NewSystemRNG()
	partnerPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, rng)
	partnerPubKey := dh.GeneratePublicKey(partnerPrivKey, grp)

	myPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, rng)

	//create context objects for general use
	fps := newFingerprints()
	ctx := &context{
		fa:  &fps,
		grp: grp,
		kv:  versioned.NewKV(make(ekv.Memstore)),
	}

	//build the session
	s := &Session{
		myPrivKey:     myPrivKey,
		partnerPubKey: partnerPubKey,
		params:        GetDefaultSessionParams(),
		manager: &Manager{
			ctx: ctx,
		},
		t: Send,
	}

	//run the generate command
	s.generate()

	//check that it generated a private key
	if s.myPrivKey.Cmp(myPrivKey) != 0 {
		t.Errorf("Public key was generated when not missing")
	}

	//verify the basekey is correct
	expectedBaseKey := dh.GenerateSessionKey(s.myPrivKey, s.partnerPubKey, grp)

	if expectedBaseKey.Cmp(s.baseKey) != 0 {
		t.Errorf("generated base key does not match expected base key")
	}

	//verify the ttl was generated
	if s.ttl == 0 {
		t.Errorf("ttl not generated")
	}

	//verify keystates where created
	if s.keyState == nil {
		t.Errorf("keystates not generated")
	}

	//verify keys were not registered in the fingerprintMap
	for keyNum := uint32(0); keyNum < s.keyState.numkeys; keyNum++ {
		key := newKey(s, keyNum)
		if _, ok := fps.toKey[key.Fingerprint()]; ok {
			t.Errorf("key %v in fingerprint map", keyNum)
		}
	}
}
