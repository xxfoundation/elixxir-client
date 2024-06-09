////////////////////////////////////////////////////////////////////////////////
// Copyright © 2024 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package rpc

import (
	"io"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/nike"
	"gitlab.com/elixxir/crypto/nike/ecdh"
	"gitlab.com/yawning/nyquist.git"
	"gitlab.com/yawning/nyquist.git/dh"
)

func init() {
	var err error
	protocol, err = nyquist.NewProtocol("Noise_NK_25519_ChaChaPoly_BLAKE2s")
	panicOnError(err)
}

func startNoiseClient(ephPrivKey nike.PrivateKey,
	serverStaticPubKey nike.PublicKey) *nyquist.HandshakeState {
	privKey := privateToNyquist(ephPrivKey)
	theirPubKey := publicToNyquist(serverStaticPubKey)

	cfg := &nyquist.HandshakeConfig{
		Protocol:       protocol,
		Prologue:       version,
		LocalEphemeral: privKey,
		RemoteStatic:   theirPubKey,
		IsInitiator:    true,
	}
	hs, err := nyquist.NewHandshake(cfg)
	panicOnError(err)
	return hs
}

func startNoiseServer(serverStaticPrivKey nike.PrivateKey,
	ephPubKey nike.PublicKey) (
	*nyquist.HandshakeState, error) {
	privKey := privateToNyquist(serverStaticPrivKey)
	theirPubKey := publicToNyquist(ephPubKey)

	cfg := &nyquist.HandshakeConfig{
		Protocol:        protocol,
		Prologue:        version,
		LocalStatic:     privKey,
		RemoteEphemeral: theirPubKey,
		IsInitiator:     false,
	}
	return nyquist.NewHandshake(cfg)
}

func clientHandshake(serverStaticPubKey nike.PublicKey,
	rng io.Reader) *nyquist.HandshakeState {
	// Per spec, the NK pattern in Noise relies on an ephemeral key. We
	// generate that here and prepend the public form to the message.
	private, public := ecdh.ECDHNIKE.NewKeypair(rng)

	privKey := privateToNyquist(private)
	theirPubKey := publicToNyquist(serverStaticPubKey)

	cfg := &nyquist.HandshakeConfig{
		Protocol:     protocol,
		Prologue:     version,
		LocalStatic:  privKey,
		RemoteStatic: theirPubKey,
		IsInitiator:  true,
	}
	hs, err := nyquist.NewHandshake(cfg)
	panicOnError(err)
	return hs
	// defer hs.Reset()
	// ciphertext, err := hs.WriteMessage(nil, plaintext)
	// panicOnNoiseError(hs, err)
	// return createNoisePayload(ciphertext, public), private
}

// decrypt decrypts the given ciphertext as a Noise X message.
func serverHandshake(ciphertext []byte, serverStaticPrivateKey nike.PrivateKey) (
	[]byte, []byte, error) {

	encrypted, requestorPubKey, err := parseNoisePayload(ciphertext)
	if err != nil {
		return nil, nil, err
	}

	privKey := privateToNyquist(serverStaticPrivateKey)
	theirPubKey := publicToNyquist(requestorPubKey)

	cfg := &nyquist.HandshakeConfig{
		Protocol:     protocol,
		Prologue:     version,
		LocalStatic:  privKey,
		RemoteStatic: theirPubKey,
		IsInitiator:  false,
	}

	return nyquist.NewHandshake(cfg)
	// if err != nil {
	// 	return nil, nil, err
	// }
	// defer hs.Reset()

	// plaintext, err := hs.ReadMessage(nil, encrypted)
	// hs.
	// 	err = recoverErrorOnNoise(hs, err)

	// sharedKey := serverStaticPrivateKey.DeriveSecret(requestorPubKey)

	// return plaintext, sharedKey, err
}

func privateToNyquist(privKey nike.PrivateKey) dh.Keypair {
	p, ok := privKey.(*ecdh.PrivateKey)
	panicOnFailureToCast(ok, "private key")

	myPrivKey, err := protocol.DH.ParsePrivateKey(p.MontgomeryBytes())
	panicOnError(err)

	return myPrivKey
}

func publicToNyquist(pubKey nike.PublicKey) dh.PublicKey {
	p, ok := pubKey.(*ecdh.PublicKey)
	panicOnFailureToCast(ok, "public key")
	myPubKey, err := protocol.DH.ParsePublicKey(p.MontgomeryBytes())
	if err != nil {
		jww.FATAL.Panic(err)
	}
	return myPubKey
}

// createNoisePayload is a helper function which will take the ciphertext
// and format it to fit Noise's specifications. The returned byte data should
// be formatted as such:
// Public Key | Ciphertext
func createNoisePayload(ciphertext []byte, ecdhPublic nike.PublicKey) []byte {
	publicKeySize := len(ecdhPublic.Bytes())
	ciphertextSize := len(ciphertext)
	res := make([]byte, publicKeySize+ciphertextSize)

	copy(res[0:publicKeySize], ecdhPublic.Bytes())
	copy(res[publicKeySize:], ciphertext)
	return res
}

// parseNoisePayload is a helper function which parses the
// ciphertext. This should be the inverse of createNoisePayload,
// returning to the user the encrypted data and the parsed public key.
func parseNoisePayload(payload []byte) ([]byte, nike.PublicKey, error) {
	// Extract the public key from the payload
	publicKeySize := ecdh.ECDHNIKE.PublicKeySize()
	publicKeyBytes := payload[:publicKeySize]
	publicKey, err := ecdh.ECDHNIKE.
		UnmarshalBinaryPublicKey(publicKeyBytes)
	if err != nil {
		return nil, nil, err
	}

	// Extract encrypted data from payload
	ciphertext := payload[publicKeySize:]

	return ciphertext, publicKey, nil
}

func panicOnFailureToCast(ok bool, keyType string) {
	if !ok {
		jww.FATAL.Panicf("%s must be x25519 ECDH", keyType)
	}
}

// panicOnError is a helper function which will panic if the
// error is not nil. This primarily serves as a fix for
// the coverage hit by un-testable error conditions.
func panicOnError(err error) {
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
}

// panicOnNoiseError is a helper function which will panice for errors on the
// Noise protocol's Encrypt/Decrypt. This primarily serves as a fix for
// the coverage hit by un-testable error conditions.
func panicOnNoiseError(hs *nyquist.HandshakeState, err error) {
	switch err {
	case nyquist.ErrDone:
		status := hs.GetStatus()
		if status.Err != nyquist.ErrDone {
			jww.FATAL.Panic(status.Err)
		}
	case nil:
	default:
		jww.FATAL.Panic(err)
	}

}

// recoverErrorOnNoise is a helper function which will handle error on the
// Noise protocol's Encrypt/Decrypt. This primarily serves as a fix for
// the coverage hit by un-testable error conditions.
func recoverErrorOnNoise(hs *nyquist.HandshakeState, err error) error {
	switch err {
	case nyquist.ErrDone:
		status := hs.GetStatus()
		if status.Err != nyquist.ErrDone {
			return status.Err
		}
	case nil:
	default:
		return err
	}
	return nil
}
