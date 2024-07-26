////////////////////////////////////////////////////////////////////////////////
// Copyright © 2024 xx foundation                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found         //
// in the LICENSE file                                                        //
////////////////////////////////////////////////////////////////////////////////

package rpc

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/crypto/nike/ecdh"
	"gitlab.com/xx_network/crypto/csprng"
)

// TestCrypto sets up a server/client handshake, then runs 5 tests:
// Complete handshake
// Client -> Server, 3 messages
// Server -> Client 3 messages
// Client -> Server 2 messages
// Client -> Server and Server -> Client interleaved, 1 message each
func TestCrypto(t *testing.T) {

	// Initialize ciphertext overhead
	rng := csprng.NewSystemRNG()
	clientPriv, clientPub := ecdh.ECDHNIKE.NewKeypair(rng)
	serverPriv, serverPub := ecdh.ECDHNIKE.NewKeypair(rng)

	client := startNoiseClient(clientPriv, serverPub)
	server, err := startNoiseServer(serverPriv, clientPub)
	panicOnError(err)

	msg := "Hello, World: %v!"

	// Handshake
	plaintext := []byte(fmt.Sprintf(msg, "hs"))
	ciphertext := client.WriteMessage(plaintext)
	plaintextOut, err := server.ReadMessage(ciphertext)
	require.NoError(t, err)
	require.Equal(t, plaintext, plaintextOut)

	actualOverhead := uint64(len(ciphertext) - len(plaintext))
	require.Equal(t, handshakeCiphertextOverhead, actualOverhead)

	ciphertext = server.WriteMessage(plaintext)
	plaintextOut, err = client.ReadMessage(ciphertext)
	require.NoError(t, err)
	require.Equal(t, plaintext, plaintextOut)

	actualOverhead = uint64(len(ciphertext) - len(plaintext))
	require.Equal(t, handshakeCiphertextOverhead, actualOverhead)

	// Client -> Server, 3 messages
	for i := 0; i < 3; i++ {
		plaintext := []byte(fmt.Sprintf(msg, i))
		ciphertext := client.WriteMessage(plaintext)

		actualOverhead := uint64(len(ciphertext) - len(plaintext))
		require.Equal(t, channelCiphertextOverhead, actualOverhead)

		readMsg, err := server.ReadMessage(ciphertext)
		require.NoError(t, err)

		require.Equal(t, plaintext, readMsg)
	}

	// Server -> Client 3 messages
	for i := 0; i < 3; i++ {
		plaintext := []byte(fmt.Sprintf(msg, i))
		ciphertext := server.WriteMessage(plaintext)

		actualOverhead := uint64(len(ciphertext) - len(plaintext))
		require.Equal(t, channelCiphertextOverhead, actualOverhead)

		readMsg, err := client.ReadMessage(ciphertext)
		require.NoError(t, err)

		require.Equal(t, plaintext, readMsg)
	}

	// Client -> Server 2 messages
	for i := 0; i < 2; i++ {
		plaintext := []byte(fmt.Sprintf(msg, i))
		ciphertext := client.WriteMessage(plaintext)

		actualOverhead := uint64(len(ciphertext) - len(plaintext))
		require.Equal(t, channelCiphertextOverhead, actualOverhead)

		readMsg, err := server.ReadMessage(ciphertext)
		require.NoError(t, err)

		require.Equal(t, plaintext, readMsg)
	}

	// Client -> Server and Server -> Client interleaved, 1 message each
	for i := 0; i < 3; i++ {
		plaintext := []byte(fmt.Sprintf(msg, i))
		ciphertext1 := client.WriteMessage(plaintext)
		ciphertext2 := server.WriteMessage(plaintext)

		readMsg1, err := client.ReadMessage(ciphertext2)
		require.NoError(t, err)
		readMsg2, err := server.ReadMessage(ciphertext1)
		require.NoError(t, err)

		require.Equal(t, plaintext, readMsg1)
		require.Equal(t, plaintext, readMsg2)
	}
}
