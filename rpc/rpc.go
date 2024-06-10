////////////////////////////////////////////////////////////////////////////////
// Copyright © 2024 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package rpc

import (
	"crypto/ed25519"
	"time"

	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/yawning/nyquist.git"
)

const (
	messageDebugTag  = "rpc"
	messageNonceSize = 4
	listenChSize     = 100
)

var (
	emptyPubKeyBytes = make([]byte, ed25519.PublicKeySize)
	emptyPubKey      = ed25519.PublicKey(emptyPubKeyBytes)
	beginningOfTime  = time.Time{}
	protocol         *nyquist.Protocol
	version          = []byte{0x0, 0x0}
	maxRoundWaitTime = time.Duration(1 * time.Minute)
)

// Response interface for RPC responses from the [Send]
// function. Provides for the ability to call a callback on error or
// success or to listen on the channels directly.
type Response interface {
	// Callback calls success or error function if/when the
	// response is received. Meant to be very similar to Promise/async
	// found in other languages.
	// RPC will call respFn 3 times: sent, round finished, and
	// response received.
	Callback(respFn func(response []byte), errFn func(err error))
}

// Callback is the type of the callback function for the RPC server,
// which is simply the return ID and the request data.
type Callback func(id *id.ID, request []byte)

type Request struct {
	id       *id.ID
	contents []byte
}

type SentMessage struct {
	Round        rounds.Round   `json:"round"`
	EphemeralIDs []ephemeral.Id `json:"ephemeralIDs"`
}

type RoundResults struct {
	Success  bool                          `json:"success"`
	TimedOut bool                          `json:"timedOut"`
	Results  map[id.Round]cmix.RoundResult `json:"results"`
}

type QueryResponse struct {
	Message []byte `json:"message"`
}

// Server is the interface for the RPC server, users should
// add a callback then start listening for RPC requests.
type Server interface {
	// Start listening on the server's cMix Identity
	Start()
	// Stop listening on the server's cMix Identity
	// Note: if you want to stop all network traffic, do
	// so with the cMix network client.
	Stop()
	// Add a Callback to process messages. This is not thread safe
	// and should be called before the server starts listening.
	AddCallback(cbFn Callback) int
	// Delete a callback for processing messages. This is
	// present for testing but generally you should not use it.
	// Not thread safe.
	DeleteCallback(i int) bool
}
