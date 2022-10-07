package ratchet

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/interfaces/nike"
	"gitlab.com/elixxir/client/nike/hybrid"
	"gitlab.com/elixxir/primitives/format"
)

type testRekeyTrigger struct {
	ids  []ID
	keys []nike.PublicKey
}

func (t *testRekeyTrigger) TriggerRekey(ratchetID ID, pubKey nike.PublicKey) {
	fmt.Println("TriggerRekey")
	t.ids = append(t.ids, ratchetID)
	t.keys = append(t.keys, pubKey)
}

type testFPTracker struct {
	added   []format.Fingerprint
	deleted []format.Fingerprint
}

func (t *testFPTracker) AddKey(fp format.Fingerprint) {
	t.added = append(t.added, fp)
}
func (t *testFPTracker) DeleteKey(fp format.Fingerprint) {
	t.deleted = append(t.deleted, fp)
}

// TestXXRatchet_Smoke performs a smoke test to be sure the
// ratchet is operating as intended. Specifically, we create 2 new
// ratchets, then check to see:
//  1. That they exhaust after the proper number of decryptions
//  2. They cannot encrypt more than the stated number of keys
//  3. That Rekey'ing produces the proper keys + state on both sides
//  4. That rekey triggers via encrypt are called and that the keys
//     can be rekeyed on the other side.
func TestXXRatchet_Smoke(t *testing.T) {
	alicePriv, alicePub := hybrid.CTIDHDiffieHellman.NewKeypair()
	bobPriv, bobPub := hybrid.CTIDHDiffieHellman.NewKeypair()

	// note we do not use defaults here, this is intentional
	params := session.Params{
		MinKeys:               10,
		MaxKeys:               20,
		RekeyThreshold:        0.5,
		NumRekeys:             5,
		UnconfirmedRetryRatio: 0.1,
	}

	rktAlice := &testRekeyTrigger{
		ids:  make([]ID, 0),
		keys: make([]nike.PublicKey, 0),
	}
	fptAlice := &testFPTracker{
		added:   make([]format.Fingerprint, 0),
		deleted: make([]format.Fingerprint, 0),
	}
	rktBob := &testRekeyTrigger{
		ids:  make([]ID, 0),
		keys: make([]nike.PublicKey, 0),
	}
	fptBob := &testFPTracker{
		added:   make([]format.Fingerprint, 0),
		deleted: make([]format.Fingerprint, 0),
	}

	alice := NewXXRatchet(alicePriv, alicePub, bobPub, params, rktAlice,
		fptAlice)
	bob := NewXXRatchet(bobPriv, bobPub, alicePub, params, rktBob, fptBob)

	// Alice send ratchets should == bob receive ratchets
	aliceSends := alice.SendRatchets()
	bobRecvs := bob.ReceiveRatchets()
	if len(aliceSends) != len(bobRecvs) || len(aliceSends) != 1 {
		t.Errorf("incorrect length on bob receive ratchets and alice "+
			"send ratchets: 1 != %d != %d", len(bobRecvs),
			len(aliceSends))
	}
	if aliceSends[0].String() != bobRecvs[0].String() {
		t.Errorf("ratchet id mismatch (alice<>bob): %s != %s",
			aliceSends[0], bobRecvs[0])
	}

	// Bob send ratchets should == alice receive ratchets
	bobSends := bob.SendRatchets()
	aliceRecvs := alice.ReceiveRatchets()
	if len(bobSends) != len(aliceRecvs) || len(bobSends) != 1 {
		t.Errorf("incorrect length on bob receive ratchets and bob "+
			"send ratchets: 1 != %d != %d", len(aliceRecvs),
			len(bobSends))
	}
	if bobSends[0].String() != aliceRecvs[0].String() {
		t.Errorf("ratchet id mismatch (bob<>alice): %s != %s",
			bobSends[0], aliceRecvs[0])
	}

	// Ratchet FPs added should be MaxKeys + NumRekeys
	msg1 := []byte("hello bob")
	for i := 0; i < int(params.MaxKeys)+1; i++ {
		t.Logf("Encrypt %d", i)
		_, err := alice.Encrypt(bobRecvs[0], msg1)
		require.NoError(t, err)
	}
	require.Equal(t, len(fptBob.added), int(params.MaxKeys))
	require.Equal(t, len(fptAlice.added), int(params.MaxKeys))

	require.Equal(t, len(rktAlice.keys), int(params.MaxKeys)/2)
}

// Property Based Tests
//  1. When Alice initiates an auth channel with Bob, Alice sends
//     auth request to Bob, when Bob decides to confirm:
//     a. Bob creates the relationship using Alice's public key.
//     b. Bob has a sending ratchet in the "triggered"
//     (NewSessionTriggered) list.
//     c. Bob has a receiving ratchet.
//     d. Bob sends a confirmation, if this fails, the sending ratchet
//     is moved to the "created" state.
//  2. Bob sends the confirmation. Bob can resend until a final
//     acknowledgement is received. When Alice receives it:
//     a. Alice creates the relationship using bob's public key.
//     b. Alice has a sending ratchet in the "confirmed" list.
//     c. Alice has a receiving ratchet.
//     d. Alice sends a final auth acknowledgement.
//  3. When Bob receives the auth acknowledgement:
//     a. Bob's sending ratchet is moved to the "confirmed" list.
//  4. When Bob receives a message for a receiving ratchet, and there is only
//     one sending ratchet (i.e., it's new from the auth system, but not
//     acknowledged)
//     a. Bob's only sending ratchet is moved to the "confirmed" list if it is in
//     the "created" or "triggered" state.
//     b. A warning is printed with the state and relationship information.
//  5. When Alice starts to run out of Sending keys in the Sending
//     Ratchet, and decides to send a rekey trigger:
//     a. Alice creates a sending ratchet added to the "sending"
//     list. This uses the pre-existing public key of Bob to derive
//     it's secret.
//     b. If the ratchet is able to be sent, it is moved to the "sent" list.
//     c. If the the ratchet cannot be sent, it is moved to the "confirmed" list.
//  6. When Bob sees the rekey trigger:
//     a. Bob creates a new receiving ratchet with his existing key and
//     Alice's public key.
//     b. Bob sends a confirmation.
//  7. When Alice receive's Bob's confirmation:
//     a. The ratchet is moved from to the "confirmed" state.
//  8. External entities can:
//     a. Move "sent" and "sending" ratchets to unconfirmed.
//     b. Create new relationships like the auth system described in 1-4.
//  9. Sending Messages:
//     a. Ratchets are moved forward, keys are marked as used, then ratchet state
//     is saved.
//  10. Receiving Messages:
//     a. A callback is registered to report new key fingerprints.
//     b. Decryption can fail if a key is reused.

/*

func TestXXRatchetEncryptDecrypt(t *testing.T) {
	salt := make([]byte, 32)
	_, err := rand.Read(salt)
	require.NoError(t, err)

	size := uint32(32)

	alicePrivKey, alicePubKey := DefaultNIKE.NewKeypair()
	bobPrivKey, bobPubKey := DefaultNIKE.NewKeypair()

	aliceRatchet := DefaultXXRatchetFactory.NewXXRatchet(alicePrivKey, alicePubKey, bobPubKey)
	bobRatchet := DefaultXXRatchetFactory.NewXXRatchet(bobPrivKey, bobPubKey, alicePubKey)
	aliceRatchet.Encrypt()

}
*/
