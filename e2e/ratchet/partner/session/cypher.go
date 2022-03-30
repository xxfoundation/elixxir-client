///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package session

import (
	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/cyclic"
	dh "gitlab.com/elixxir/crypto/diffieHellman"
	e2eCrypto "gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/format"
)

// GenerateE2ESessionBaseKey returns the baseKey symmetric encryption key root.
// The baseKey is created by hashing the results of the diffie-helman (DH) key
// exchange with the post-quantum secure Supersingular Isogeny DH exchange
// results.
func GenerateE2ESessionBaseKey(myDHPrivKey, theirDHPubKey *cyclic.Int,
	dhGrp *cyclic.Group, mySIDHPrivKey *sidh.PrivateKey,
	theirSIDHPubKey *sidh.PublicKey) *cyclic.Int {
	// DH Key Gen
	dhKey := dh.GenerateSessionKey(myDHPrivKey, theirDHPubKey, dhGrp)

	// SIDH Key Gen
	sidhKey := make([]byte, mySIDHPrivKey.SharedSecretSize())
	mySIDHPrivKey.DeriveSecret(sidhKey, theirSIDHPubKey)

	// Derive key
	h := hash.CMixHash.New()
	h.Write(dhKey.Bytes())
	h.Write(sidhKey)
	keyDigest := h.Sum(nil)
	// NOTE: Sadly the baseKey was a full DH key, and that key was used
	// to create an "IDF" as well as in key generation and potentially other
	// downstream code. We use a KDF to limit scope of the change,'
	// generating into the same group as DH to preserve any kind of
	// downstream reliance on the size of the key for now.
	baseKey := hash.ExpandKey(hash.CMixHash.New, dhGrp, keyDigest,
		dhGrp.NewInt(1))

	jww.INFO.Printf("Generated E2E Base Key: %s", baseKey.Text(16))

	return baseKey
}

type Cypher struct {
	// Links
	session *Session

	fp *format.Fingerprint

	// keyNum is the index of the key by order of creation
	// it is used to identify the key in the key.Session
	keyNum uint32
}

func newKey(session *Session, keynum uint32) *Cypher {
	return &Cypher{
		session: session,
		keyNum:  keynum,
	}
}

// return pointers to higher level management structures
func (k *Cypher) GetSession() *Session { return k.session }

// returns the key fingerprint if it has it, otherwise generates it
// this function does not memoize the fingerprint if it doesnt have it because
// in most cases it will not be used for a long time and as a result should not
// be stored in ram.
func (k *Cypher) Fingerprint() format.Fingerprint {
	if k.fp != nil {
		return *k.fp
	}
	return e2eCrypto.DeriveKeyFingerprint(k.session.baseKey, k.keyNum,
		k.session.relationshipFingerprint)
}

// the E2E key to encrypt msg to its intended recipient
// It also properly populates the associated data, including the MAC, fingerprint,
// and encrypted timestamp
func (k *Cypher) Encrypt(msg format.Message) format.Message {
	fp := k.Fingerprint()
	key := k.generateKey()

	// set the fingerprint
	msg.SetKeyFP(fp)

	// encrypt the payload
	encPayload := e2eCrypto.Crypt(key, fp, msg.GetContents())
	msg.SetContents(encPayload)

	// create the MAC
	// MAC is HMAC(key, ciphertext)
	// Currently, the MAC doesn't include any of the associated data
	MAC := hash.CreateHMAC(encPayload, key[:])
	msg.SetMac(MAC)

	return msg
}

// Decrypt uses the E2E key to decrypt the message
// It returns an error in case of HMAC verification failure
// or in case of a decryption error (related to padding)
func (k *Cypher) Decrypt(msg format.Message) (format.Message, error) {
	fp := k.Fingerprint()
	key := k.generateKey()

	// Verify the MAC is correct
	if !hash.VerifyHMAC(msg.GetContents(), msg.GetMac(), key[:]) {
		return format.Message{}, errors.New("HMAC verification failed for E2E message")
	}

	// Decrypt the payload
	decryptedPayload := e2eCrypto.Crypt(key, fp, msg.GetContents())

	//put the decrypted payload back in the message
	msg.SetContents(decryptedPayload)

	return msg, nil
}

// Use sets the key as used. It cannot be used again.
func (k *Cypher) Use() {
	k.session.useKey(k.keyNum)
}

// generateKey derives the current e2e key from the baseKey and the index
// keyNum and returns it
func (k *Cypher) generateKey() e2eCrypto.Key {
	return e2eCrypto.DeriveKey(k.session.baseKey, k.keyNum,
		k.session.relationshipFingerprint)
}
