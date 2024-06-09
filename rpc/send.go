////////////////////////////////////////////////////////////////////////////////
// Copyright © 2024 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package rpc

import (
	"encoding/json"

	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/identity"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/crypto/nike"
	"gitlab.com/elixxir/crypto/nike/ecdh"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/yawning/nyquist.git"
)

func Send(net cMixClient, serverID *id.ID, serverKey nike.PublicKey,
	request []byte, params cmix.CMIXParams) Response {
	res := newResponse(net, serverKey)
	// helper to return errors as part of the response
	responseErr := func(err error) Response {
		go errEvent(res, err)
		return res
	}

	rng := net.RNGStreamGenerator().GetStream()
	defer rng.Close()

	// Generate Ephemeral reception ID
	myID, err := generateRandomID(rng)
	if err != nil {
		return responseErr(err)
	}

	// Generate keys and setup our decryptor
	private, public := ecdh.ECDHNIKE.NewKeypair(rng)
	hs := startNoiseClient(private, serverKey)
	res.cipher = hs

	// Register a listener on that identity
	res.myID = myID
	net.AddIdentity(myID, identity.Forever, false, res)
	// NOTE: this identity is removed when the channels are closed in the
	// Callback call.

	// plaintext is the message part that is encrypted and sent to server
	plaintext := createMessage(myID, request)
	// ciphertext is the encrypted part of the message
	ciphertext, err := hs.WriteMessage(nil, plaintext)
	panicOnNoiseError(hs, err)
	// msg prepends the public key to the ciphertext
	msg := createNoisePayload(ciphertext, public)
	go func() {
		rnd, ids, err := send(net, serverID, msg, params)
		if err != nil {
			errEvent(res, err)
		}
		sentEvent(res, rnd, ids)
		trackRound(net, res, rnd)
	}()

	return res
}

func errEvent(res *response, err error) {
	res.errs <- err
	close(res.errs)
	close(res.listener)
}

func sentEvent(res *response, rnd rounds.Round, ids []ephemeral.Id) {
	json, err := json.Marshal(map[string]interface{}{
		"type": "SentResponse",
		"response": SentResponse{
			Round:        rnd,
			EphemeralIDs: ids,
		},
	})
	if err != nil {
		res.errs <- err
	}
	res.listener <- json
}

func trackRound(net cMixClient, res *response, rnd rounds.Round) {

}

// createMessage creates a combined message, in the spec this can be
// partitioned but for v0 here we just attach the reception id to the
// beginning.
func createMessage(responderID *id.ID, data []byte) []byte {
	idBytes := responderID.Bytes()
	newData := make([]byte, len(data)+len(idBytes))
	copy(newData, idBytes)
	copy(newData[len(idBytes):], data)
	return newData
}

type response struct {
	listener  chan []byte
	errs      chan error
	net       cMixClient
	serverKey nike.PublicKey
	myID      *id.ID
	cipher    *nyquist.HandshakeState
}

func (r *response) Callback(respFn func(response []byte),
	errFn func(err error)) {
	go func() {
		// read until channel closes
		for r := range r.listener {
			respFn(r)
		}
		// if any errors report them
		select {
		case e := <-r.errs:
			errFn(e)
		default:
		}
		// cleanup
		// remove the temp id from the cmix client
		if r.myID != nil {
			r.net.RemoveIdentity(r.myID)
		}
		if r.cipher != nil {
			r.cipher.Reset()
		}
	}()
}

// //
// Processor interface implementation
// The cMix layer deals with these and the users shouldn't be interacting
// with these parts.
// //
func (r *response) String() string {
	return "rpcResponse"
}

func (r *response) Process(msg format.Message, _ []string, _ []byte,
	ephID receptionID.EphemeralIdentity, round rounds.Round) {
	// parse the cmix into our message format
	ciphertext := reconstructCiphertext(msg)
	// decrypt
	plaintext, err := r.cipher.ReadMessage(nil, ciphertext)
	err = recoverErrorOnNoise(r.cipher, err)
	if err != nil {
		r.errs <- err
	}
	// send the raw response to the callback listener
	r.listener <- plaintext
}

// NOTE: unbuffered for now, which means we block on send until the
// caller/receiver reads from the channel.
func newResponse(net cMixClient, serverKey nike.PublicKey) *response {
	return &response{
		listener:  make(chan []byte),
		errs:      make(chan error),
		serverKey: serverKey,
		net:       net,
	}
}
