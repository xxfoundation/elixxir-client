////////////////////////////////////////////////////////////////////////////////
// Copyright © 2024 xx foundation                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found         //
// in the LICENSE file                                                        //
////////////////////////////////////////////////////////////////////////////////

package rpc

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/crypto/nike"
	"gitlab.com/elixxir/crypto/nike/ecdh"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
)

// TestServer creates a server instance with a mockCmix object, then uses
// a callback analogous to what is done in send_test.go for the TestSend
// function, but additionally tests all the plumbing and logic
// for the server code.
func TestServer(t *testing.T) {
	net := MockCmix(t)
	rng := net.RNGStreamGenerator().GetStream()
	defer rng.Close()

	// The server generates a keypair and a random reception ID.
	serverID, err := generateRandomID(rng)
	require.NoError(t, err)
	serverPriv, serverPub := ecdh.ECDHNIKE.NewKeypair(rng)

	// The callback returns a reply that is 10x the max payload
	// size, which should become 11 cMix Messages.
	maxPayloadSz := uint64(maxPayloadLen(net))
	cbFn := func(id *id.ID, request []byte) []byte {
		reply := make([]byte, maxPayloadSz*10)
		for i := 0; i < len(reply); i++ {
			reply[i] = request[i%len(request)]
		}
		return reply
	}

	server := NewServer(net, cbFn, serverID, serverPriv)
	server.Start()

	expMsg := []byte("Hello, World!")
	r := Send(net, serverID, serverPub, expMsg,
		cmix.GetDefaultCMIXParams())
	require.NotNil(t, net.processor)

	responses := make([][]byte, 0)

	success := func(data []byte) {
		var prettyJSON bytes.Buffer
		err := json.Indent(&prettyJSON, data, "", "\t")
		require.NoError(t, err)
		// t.Logf("%s", prettyJSON.String())
		responses = append(responses, data)
	}
	fail := func(err error) {
		require.NoError(t, err)
	}

	r.Callback(success, fail)

	d := r.Wait()
	require.Equal(t, uint64(len(d)), 10*maxPayloadSz)
}

func TestMarshalling(t *testing.T) {
	rng := csprng.NewSystemRNG()

	for i := 0; i < 10; i++ {
		pr1, pu1 := ecdh.ECDHNIKE.NewKeypair(rng)
		pr2, pu2 := ecdh.ECDHNIKE.NewKeypair(rng)

		// sanity check
		nk1 := pr1.DeriveSecret(pu2)
		nk2 := pr2.DeriveSecret(pu1)
		require.Equal(t, nk1, nk2)

		// Now Dump to Base64 string
		pr1Str := base64.RawStdEncoding.EncodeToString(pr1.Bytes())
		pu1Str := base64.RawStdEncoding.EncodeToString(pu1.Bytes())
		pr2Str := base64.RawStdEncoding.EncodeToString(pr2.Bytes())
		pu2Str := base64.RawStdEncoding.EncodeToString(pu2.Bytes())

		srvPriv, srvPub := checkKeys(pr1Str, pu1Str, t)
		cliPriv, cliPub := checkKeys(pr2Str, pu2Str, t)

		sk1 := srvPriv.DeriveSecret(cliPub)
		sk2 := cliPriv.DeriveSecret(srvPub)

		t.Logf("sk1: %+v", sk1)
		t.Logf("sk2: %+v", sk2)

		require.Equal(t, sk1, sk2)
		require.Equal(t, nk1, sk1)
	}
}

func TestVectors(t *testing.T) {
	rng := csprng.NewSystemRNG()

	serverPriv, serverPub := ecdh.ECDHNIKE.NewKeypair(rng)
	clientPriv, clientPub := ecdh.ECDHNIKE.NewKeypair(rng)

	nk1 := serverPriv.DeriveSecret(clientPub)
	nk2 := clientPriv.DeriveSecret(serverPub)

	require.Equal(t, nk1, nk2)
	t.Logf("nk1: %v", nk1)

	srvPrivStr := "WBfCA63BbC9vaOeRi0FpqlfKjOedQM6TW89AwlfnVKfKrGHPN5J9Qh0cC9I4yzEG6brIveyvwI8RvTeT0OjfTw"
	srvPubStr := "yqxhzzeSfUIdHAvSOMsxBum6yL3sr8CPEb03k9Do308"

	clientPrivStr := "NOPOKU2+SrsQv7DSXkiOkUytGdZeF/+H+j4bn50XbSNnvFqJawvtnofZUW1wlUXsHa55CKLrLUp6nMV8f5orgQ"
	clientPubStr := "Z7xaiWsL7Z6H2VFtcJVF7B2ueQii6y1KepzFfH+aK4E"

	srvPriv, srvPub := checkKeys(srvPrivStr, srvPubStr, t)
	cliPriv, cliPub := checkKeys(clientPrivStr, clientPubStr, t)

	sk1 := srvPriv.DeriveSecret(cliPub)
	sk2 := cliPriv.DeriveSecret(srvPub)

	t.Logf("sk1: %+v", sk1)
	t.Logf("sk2: %+v", sk2)

	require.Equal(t, sk1, sk2)
}

func checkKeys(privStr, pubStr string, t *testing.T) (nike.PrivateKey,
	nike.PublicKey) {
	privBytes, err := base64.RawStdEncoding.DecodeString(privStr)
	require.NoError(t, err)

	priv := ecdh.ECDHNIKE.NewEmptyPrivateKey()
	err = priv.FromBytes(privBytes)
	require.NoError(t, err)

	derivedPub := priv.Scheme().DerivePublicKey(priv)
	require.Equal(t, pubStr, base64.RawStdEncoding.EncodeToString(
		derivedPub.Bytes()))
	pub := ecdh.ECDHNIKE.NewEmptyPublicKey()
	pubBytes, err := base64.RawStdEncoding.DecodeString(pubStr)
	require.NoError(t, err)
	err = pub.FromBytes(pubBytes)
	require.NoError(t, err)
	return priv, pub
}
