////////////////////////////////////////////////////////////////////////////////
// Copyright © 2024 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package rpc

import (
	"encoding/json"
	"sync"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/identity"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/crypto/nike"
	"gitlab.com/elixxir/crypto/nike/ecdh"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

const msgIdSz = 4

type msgId [msgIdSz]byte

func Send(net cMixClient, serverID *id.ID, serverKey nike.PublicKey,
	request []byte, params cmix.CMIXParams) Response {
	res := newResponse(net, serverKey)
	res.wg.Add(1)
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

	// Generate keys and setup our encryptor/decryptor
	private, public := ecdh.ECDHNIKE.NewKeypair(rng)
	noise := startNoiseClient(private, serverKey)
	res.cipher = noise

	// Register a listener on that identity
	res.myID = myID
	net.AddIdentity(myID, identity.Forever, false, res)
	// NOTE: this identity is removed when the channels are closed in the
	// Callback call.

	// Calculate partition sizes, note that the ephKeySz is
	// subtracted from the header. When we update to send
	// multi-part queries, the hash of the plaintext of the
	// previous message is used to discover partition order like
	// in the server response code, which is why msgIdSz is
	// subtracted.
	maxPayloadSz := uint64(maxPayloadLen(net))
	ephKeySz := uint64(len(public.Bytes()))
	headerSz := maxPayloadSz - handshakeCiphertextOverhead - ephKeySz
	otherSz := maxPayloadSz - channelCiphertextOverhead - msgIdSz

	// The message at the plaintext network layer is the one time
	// use reception ID + the request
	msg := make([]byte, len(myID.Bytes())+len(request))
	copy(msg, myID.Bytes())
	copy(msg[len(myID.Bytes()):], request)

	// set the initial id for processing the response.
	res.nextId = genResponseMsgID(msg, noise.SharedKey)

	// plaintext is the message part that is encrypted and sent to server
	plaintexts := partitionMessage(msg, headerSz, otherSz)
	// FIXME: we should probably eliminate this restriction
	if len(plaintexts) != 1 {
		return responseErr(errors.Errorf("[RPC] Query too big, %d > %d",
			len(request), int(headerSz)-2-len(myID.Bytes())))
	}
	// ciphertext is the encrypted part of the message
	msgToSend := make([]byte, maxPayloadSz)
	copy(msgToSend, public.Bytes())
	copy(msgToSend[ephKeySz:], noise.WriteMessage(plaintexts[0]))
	go func() {
		rnd, ids, err := send(net, serverID, msgToSend, params)
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
	res.Close()
}

func sentEvent(res *response, rnd rounds.Round, ids []ephemeral.Id) {
	json, err := json.Marshal(map[string]interface{}{
		"type": "SentMessage",
		"response": SentMessage{
			Round:        rnd,
			EphemeralIDs: ids,
		},
	})
	if err != nil {
		res.errs <- err
		res.Close()
	}
	res.listener <- json
}

func trackRound(net cMixClient, res *response, rnd rounds.Round) {
	cb := func(allRoundsSucceeded, timedOut bool,
		rounds map[id.Round]cmix.RoundResult) {
		json, err := json.Marshal(map[string]interface{}{
			"type": "RoundResults",
			"response": RoundResults{
				Success:  allRoundsSucceeded,
				TimedOut: timedOut,
				Results:  rounds,
			},
		})
		if err != nil {
			res.errs <- err
			res.Close()
		} else {
			res.listener <- json
		}
	}

	net.GetRoundResults(maxRoundWaitTime, cb, rnd.ID)
}

type response struct {
	listener  chan []byte
	errs      chan error
	net       cMixClient
	serverKey nike.PublicKey
	myID      *id.ID
	cipher    *noise
	// Used for message reception to reconstruct partitioned message
	nextId  msgId
	replies map[msgId][]byte
	parts   [][]byte
	reply   []byte
	sync.Mutex
	wg sync.WaitGroup
}

// Close closes the listener channel so that the callback
// listeners complete.
func (r *response) Close() {
	close(r.listener)
	r.wg.Done()
}

func (r *response) Callback(respFn func(response []byte),
	errFn func(err error)) Response {
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
	return r
}

func (r *response) Wait() []byte {
	r.wg.Wait()
	return r.reply
}

// //
// Processor interface implementation
// The cMix layer deals with these and the users shouldn't be interacting
// with these parts.
// //
func (r *response) String() string {
	return "rpcResponse"
}

func (r *response) Process(cMixMsg format.Message, _ []string, _ []byte,
	ephID receptionID.EphemeralIdentity, round rounds.Round) {
	r.Lock()
	defer r.Unlock()

	// parse the received cmix message into our message format
	msg := reconstructCiphertext(cMixMsg)
	var curId msgId
	copy(curId[:], msg[:msgIdSz])

	ciphertext := msg[msgIdSz:]
	r.replies[curId] = ciphertext

	// Attempt to decrypt as much as we can
	ciphertext, ok := r.replies[r.nextId]
	for ok {
		plaintext, err := r.cipher.ReadMessage(ciphertext)
		if err != nil {
			r.errs <- err
			r.Close()
			return
		}
		r.parts = append(r.parts, plaintext)
		r.nextId = genResponseMsgID(plaintext, r.cipher.SharedKey)
		ciphertext, ok = r.replies[r.nextId]
	}

	// Attempt reconstruction
	msg, err := reconstructPartitions(r.parts)
	if err != nil && err != ErrMissingParts {
		r.errs <- err
		r.Close()
		return
	}
	if err != nil && err == ErrMissingParts {
		jww.INFO.Printf("[RPC] missing parts for reconstruction")
		return
	}

	r.reply = msg

	json, err := json.Marshal(map[string]interface{}{
		"type": "QueryResponse",
		"response": QueryResponse{
			Message: msg,
		},
	})
	if err != nil && err != ErrMissingParts {
		r.errs <- err
		r.Close()
		return
	}

	// send the raw response to the callback listener
	r.listener <- json
	r.Close()
}

// NOTE: unbuffered for now, which means we block on send until the
// caller/receiver reads from the channel.
func newResponse(net cMixClient, serverKey nike.PublicKey) *response {
	return &response{
		listener:  make(chan []byte),
		errs:      make(chan error, 10),
		serverKey: serverKey,
		myID:      nil,
		cipher:    nil,

		net:     net,
		replies: make(map[msgId][]byte),
		parts:   make([][]byte, 0),
		reply:   make([]byte, 0),
	}
}
