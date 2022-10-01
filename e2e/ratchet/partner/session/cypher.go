////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package session

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/elixxir/client/interfaces/nike"
	"gitlab.com/elixxir/crypto/cyclic"
	dh "gitlab.com/elixxir/crypto/diffieHellman"
	e2eCrypto "gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/format"
)

// GenerateE2ESessionBaseKey returns the baseKey symmetric encryption key root.
// The baseKey is created by hashing the results of the Diffie-Hellman (DH) key
// exchange with the post-quantum secure Supersingular Isogeny DH exchange
// results.
func GenerateE2ESessionBaseKey(myDHPrivKey, theirDHPubKey *cyclic.Int,
	dhGrp *cyclic.Group, myPQPrivKey nike.PrivateKey,
	theirPQPubKey nike.PublicKey) *cyclic.Int {
	// DH Key Gen
	dhKey := dh.GenerateSessionKey(myDHPrivKey, theirDHPubKey, dhGrp)

	// PQ Key Gen
	pqKey := myPQPrivKey.DeriveSecret(theirPQPubKey)

	// Derive key
	h := hash.CMixHash.New()
	h.Write(dhKey.Bytes())
	h.Write(pqKey)
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

// Cypher manages the cryptographic material for E2E messages and provides
// methods to encrypt and decrypt them.
type Cypher interface {

	// GetSession return pointers to higher level management structures.
	GetSession() *Session

	// Fingerprint returns the Cypher key fingerprint, if it has it. Otherwise,
	// it generates and returns a new one.
	Fingerprint() format.Fingerprint

	// Encrypt uses the E2E key to encrypt the message to its intended
	// recipient. It also properly populates the associated data, including the
	// MAC, fingerprint, and encrypted timestamp. It generates a residue of the
	// key used to encrypt the contents.
	Encrypt(contents []byte) (ecrContents, mac []byte, residue e2eCrypto.KeyResidue)

	// Decrypt uses the E2E key to decrypt the message. It returns an error in
	// case of HMAC verification failure or in case of a decryption error
	// (related to padding). It generates a residue of the
	// key used to encrypt the contents.
	Decrypt(msg format.Message) (decryptedPayload []byte, residue e2eCrypto.KeyResidue, err error)

	// Use sets the key as used. It cannot be used again.
	Use()
}

// cypher adheres to the Cypher interface.
type cypher struct {
	session *Session

	fp *format.Fingerprint

	// keyNum is the index of the key by order of creation. It is used to
	// identify the key in the cypher.Session.
	keyNum uint32
}

func newCypher(session *Session, keyNum uint32) *cypher {
	return &cypher{
		session: session,
		keyNum:  keyNum,
	}
}

// GetSession return pointers to higher level management structures.
func (k *cypher) GetSession() *Session { return k.session }

// Fingerprint returns the Cypher key fingerprint, if it has it. Otherwise, it
// generates and returns a new one. This function does not memoize the
// fingerprint if it does not have it because in most cases, it will not be used
// for a long time and as a result, should not be stored in memory.
func (k *cypher) Fingerprint() format.Fingerprint {
	if k.fp != nil {
		return *k.fp
	}
	return e2eCrypto.DeriveKeyFingerprint(
		k.session.baseKey, k.keyNum, k.session.relationshipFingerprint)
}

// Encrypt uses the E2E key to encrypt the message to its intended recipient. It
// also properly populates the associated data, including the MAC, fingerprint,
// and encrypted timestamp. It generates a residue of the key used to encrypt the contents.
func (k *cypher) Encrypt(contents []byte) (ecrContents, mac []byte, residue e2eCrypto.KeyResidue) {
	fp := k.Fingerprint()
	key := k.generateKey()

	residue = e2eCrypto.NewKeyResidue(key)

	// encrypt the payload
	ecrContents = e2eCrypto.Crypt(key, fp, contents)

	// Create the MAC, which is HMAC(key, ciphertext)
	// Currently, the MAC doesn't include any of the associated data
	mac = hash.CreateHMAC(ecrContents, key[:])

	return ecrContents, mac, residue
}

// Decrypt uses the E2E key to decrypt the message. It returns an error in case
// of HMAC verification failure or in case of a decryption error (related to
// padding). It generates a residue of the key used to encrypt the contents
func (k *cypher) Decrypt(msg format.Message) (decryptedPayload []byte, residue e2eCrypto.KeyResidue, err error) {
	fp := k.Fingerprint()
	key := k.generateKey()

	// Verify the MAC is correct
	if !hash.VerifyHMAC(msg.GetContents(), msg.GetMac(), key[:]) {
		return nil, e2eCrypto.KeyResidue{}, errors.New("HMAC verification failed for E2E message")
	}

	// Decrypt the payload
	decryptedPayload = e2eCrypto.Crypt(key, fp, msg.GetContents())

	// Construct residue
	residue = e2eCrypto.NewKeyResidue(key)

	return decryptedPayload, residue, nil
}

// Use sets the key as used. It cannot be used again.
func (k *cypher) Use() {
	k.session.useKey(k.keyNum)
}

// generateKey derives the current e2e key from the baseKey and the index keyNum
// and returns it.
func (k *cypher) generateKey() e2eCrypto.Key {
	return e2eCrypto.DeriveKey(
		k.session.baseKey, k.keyNum, k.session.relationshipFingerprint)
}
