////////////////////////////////////////////////////////////////////////////////
// Copyright © 2024 xx foundation                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found         //
// in the LICENSE file                                                        //
////////////////////////////////////////////////////////////////////////////////

package rpc

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/crypto/nike/ecdh"
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

	server := NewServer(net, serverID, cbFn, serverPriv)
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
