////////////////////////////////////////////////////////////////////////////////
// Copyright © 2024 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package rpc

import (
	"bytes"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/nike"
	"gitlab.com/elixxir/crypto/nike/ecdh"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/yawning/nyquist.git"
	"gitlab.com/yawning/nyquist.git/dh"
)

var (
	handshakeCiphertextOverhead uint64
	channelCiphertextOverhead   uint64
)

func init() {
	var err error
	protocol, err = nyquist.NewProtocol("Noise_NK_25519_ChaChaPoly_BLAKE2s")
	panicOnError(err)

	// We need to calculate ciphertext overhead so we can
	// partition our packets.
	rng := csprng.NewSystemRNG()
	priv1, pub1 := ecdh.ECDHNIKE.NewKeypair(rng)
	priv2, pub2 := ecdh.ECDHNIKE.NewKeypair(rng)

	client := startNoiseClient(priv1, pub2)
	server, err := startNoiseServer(priv2, pub1)
	panicOnError(err)

	plaintext := []byte("Hello, World!")

	// Handshake Overhead
	cipher1 := client.WriteMessage(plaintext)
	plaintextOut, err := server.ReadMessage(cipher1)
	panicOnError(err)
	if !bytes.Equal(plaintext, plaintextOut) {
		jww.FATAL.Panicf("encrypt init failure")
	}
	cipher2 := server.WriteMessage(plaintext)
	plaintextOut, err = client.ReadMessage(cipher2)
	panicOnError(err)
	if !bytes.Equal(plaintext, plaintextOut) {
		jww.FATAL.Panicf("encrypt init failure")
	}
	handshakeCiphertextOverhead = uint64(len(cipher1) - len(plaintext))

	// Channel overhead
	cipher3 := client.WriteMessage(plaintext)
	plaintextOut, err = server.ReadMessage(cipher3)
	panicOnError(err)
	if !bytes.Equal(plaintext, plaintextOut) {
		jww.FATAL.Panicf("encrypt init failure")
	}
	channelCiphertextOverhead = uint64(len(cipher3) - len(plaintext))

	jww.INFO.Printf("[RPC] CiphertextOverheads: %d, %d",
		handshakeCiphertextOverhead,
		channelCiphertextOverhead)
}

// noise holds the handshake state, notes when the handshake is complete,
// and uses the individual cipher states as appropriate after the handshake.
// The structure itself wraps the handshake Read/Write interface to provide
// a channel like interface.
//
// This has a limitation in that the initiator (client) can only send one
// message first, then it has to wait for the server to respond to continue
// sending messages. We need to do more research to see if this is OK to do
// inside the noise framework (it should be) before we drop this limitation.
//
// For now, clients should send an initial message, then wait for a
// server response until sending additional data for the handshake to
// complete on the server side.
type noise struct {
	hs         *nyquist.HandshakeState
	hsDone     bool
	sendCipher *nyquist.CipherState
	recvCipher *nyquist.CipherState
}

func newNoise(hs *nyquist.HandshakeState) *noise {
	return &noise{
		hs:         hs,
		hsDone:     false,
		sendCipher: nil,
		recvCipher: nil,
	}
}

// ReadMessage from the remote sender
func (n *noise) ReadMessage(ciphertext []byte) ([]byte, error) {
	if !n.hsDone {
		pt, err := n.hs.ReadMessage(nil, ciphertext)
		if err == nyquist.ErrDone {
			n.hsDone = true
			// NOTE: If we finish when reading, the
			// send/recv ciphers are backwards from Write
			// side
			n.sendCipher = n.hs.GetStatus().CipherStates[1]
			n.recvCipher = n.hs.GetStatus().CipherStates[0]
			return pt, nil
		}
		return pt, err
	}

	// NOTE: we could use associated data from previous message
	// plaintext, for now we will not for simplicity.
	return n.recvCipher.DecryptWithAd(nil, nil, ciphertext)
}

// WriteMessage to a remote receiver
func (n *noise) WriteMessage(plaintext []byte) []byte {
	if !n.hsDone {
		ct, err := n.hs.WriteMessage(nil, plaintext)
		if err == nyquist.ErrDone {
			n.hsDone = true
			n.sendCipher = n.hs.GetStatus().CipherStates[0]
			n.recvCipher = n.hs.GetStatus().CipherStates[1]
		} else {
			// NOTE: The most likely scenario is a message
			// query too big to fit into a single packet, causing
			// an out of order error here. We leave this as a
			// limitation for now, but we could potentially
			// fix this by splitting the cipher state after
			// first read/write and setting the appropriate field
			// on the noise object.
			panicOnError(err)
		}
		return ct
	}

	ct, err := n.sendCipher.EncryptWithAd(nil, nil, plaintext)
	panicOnError(err)
	return ct
}

// Ready is true when the handshake is completed
func (n *noise) Ready() bool {
	return n.hsDone
}

// A noise client has a local ephemeral key and a remote static key
// It is the initiator of the handshake
func startNoiseClient(ephPrivKey nike.PrivateKey,
	serverStaticPubKey nike.PublicKey) *noise {
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
	return newNoise(hs)
}

// A noise server has a static private key and a remote ephemeral key
// It does not initiate a handshake
func startNoiseServer(serverStaticPrivKey nike.PrivateKey,
	ephPubKey nike.PublicKey) (*noise, error) {
	privKey := privateToNyquist(serverStaticPrivKey)
	theirPubKey := publicToNyquist(ephPubKey)

	cfg := &nyquist.HandshakeConfig{
		Protocol:        protocol,
		Prologue:        version,
		LocalStatic:     privKey,
		RemoteEphemeral: theirPubKey,
		IsInitiator:     false,
	}
	hs, err := nyquist.NewHandshake(cfg)
	if err != nil {
		return nil, err
	}
	return newNoise(hs), nil
}

// Key conversions with nyquist

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

// Error handling functions

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
