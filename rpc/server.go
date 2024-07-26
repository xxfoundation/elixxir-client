////////////////////////////////////////////////////////////////////////////////
// Copyright © 2024 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package rpc

import (
	"encoding/base64"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/identity"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/nike"
	"gitlab.com/elixxir/crypto/nike/ecdh"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
)

func NewServer(net cMixClient,
	callback func(id *id.ID, request []byte) []byte,
	receptionID *id.ID,
	privateKey nike.PrivateKey) Server {
	rpc := &rpcServer{
		me:         receptionID,
		privateKey: privateKey,
		publicKey:  privateKey.Scheme().DerivePublicKey(privateKey),
		net:        net,
		rng:        net.RNGStreamGenerator(),
		listener:   make(chan Request, listenChSize),
		cb:         callback,
		ctr:        0,
	}

	return rpc
}

func (r *rpcServer) Start() {
	jww.INFO.Printf("[RPC] Starting Server with ReceptionID: %s",
		base64.RawStdEncoding.EncodeToString(r.me.Bytes()))
	jww.INFO.Printf("[RPC] Starting Server with PublicKey: %s",
		base64.RawStdEncoding.EncodeToString(r.publicKey.Bytes()))
	r.net.AddIdentityWithHistory(r.me, identity.Forever,
		beginningOfTime, true, r)
}

func (r *rpcServer) Stop() {
	r.net.RemoveIdentity(r.me)
}

func (r *rpcServer) SetCallback(cbFn func(id *id.ID, request []byte) []byte) {
	r.cb = cbFn
}

type rpcServer struct {
	me         *id.ID
	privateKey nike.PrivateKey
	publicKey  nike.PublicKey

	net      cMixClient
	rng      *fastRNG.StreamGenerator
	listener chan Request
	cb       func(id *id.ID, request []byte) []byte
	ctr      int
}

////
// [message.Processor] Interface
////

func (r *rpcServer) String() string {
	return "rpcServer"
}

func (r *rpcServer) Process(cMixMsg format.Message, _ []string, _ []byte,
	ephID receptionID.EphemeralIdentity, round rounds.Round) {
	ct := reconstructCiphertext(cMixMsg)

	// Extract publicKey from requesting client
	// (NOTE: this part gets complicated if it's a multipart request which
	// we do not support for now)
	clientPubKey := ecdh.ECDHNIKE.NewEmptyPublicKey()
	pkSz := clientPubKey.Scheme().PublicKeySize()

	err := clientPubKey.FromBytes(ct[:pkSz])
	if err != nil {
		jww.ERROR.Printf("[RPC] invalid pubkey, dropping msg: %s, %v",
			cMixMsg.GetKeyFP(), ct[:pkSz])
		return
	}

	cipher, err := startNoiseServer(r.privateKey, clientPubKey)
	if err != nil {
		jww.ERROR.Printf("[RPC] couldn't make cipher for msg: %s, %+v",
			cMixMsg.GetKeyFP(), err)
		return
	}

	// Decrypt with r.cipher
	pt, err := cipher.ReadMessage(ct[pkSz:])
	if err != nil {
		jww.ERROR.Printf("[RPC] couldn't decrypt msg: %s, %+v",
			cMixMsg.GetKeyFP(), err)
		jww.ERROR.Printf("[RPC] GARBLED: %v, %v", ct[:pkSz], ct[pkSz:])
		return
	}

	// "reconstruct" partitions, althought there is always only ever 1
	// in the request.
	msg, err := reconstructPartitions([][]byte{pt})
	if err != nil {
		jww.ERROR.Printf("[RPC] couldn't repartition msg: %s, %+v",
			cMixMsg.GetKeyFP(), err)
		return
	}

	// Sanity check that the request includes a reply ID and
	// at least 1 byte of a message.
	if len(msg) <= id.ArrIDLen {
		jww.ERROR.Printf("[RPC] decoded msg too short: %s, %v",
			cMixMsg.GetKeyFP(), msg)
		return
	}

	idBytes := msg[:id.ArrIDLen]
	request := msg[id.ArrIDLen:]

	replyMsgId := genResponseMsgID(msg, cipher.SharedKey)
	ephCmixId, err := id.Unmarshal(idBytes)
	if len(msg) <= id.ArrIDLen {
		jww.ERROR.Printf("[RPC] unable to decode sender id: %s, %+v",
			cMixMsg.GetKeyFP(), err)
		return
	}

	maxPayloadSz := uint64(maxPayloadLen(r.net))
	// NOTE: The server does not prepend the ephemeral pubkey,
	//       only replyMsgId
	headerSz := maxPayloadSz - handshakeCiphertextOverhead - msgIdSz
	otherSz := maxPayloadSz - channelCiphertextOverhead - msgIdSz

	// Do the reply inside a separate routine that catches any crashes
	go func() {
		sendResp := func(response []byte) {
			parts := partitionMessage(response, headerSz, otherSz)

			// Encrypt & Send Reply
			ciphertexts := make([][]byte, len(parts))
			for i := 0; i < len(parts); i++ {
				ct := cipher.WriteMessage(parts[i])
				if i != 0 {
					replyMsgId = genResponseMsgID(parts[i-1],
						cipher.SharedKey)
				}
				ct = append(replyMsgId[:], ct...)
				ciphertexts[i] = ct
			}
			for {
				rnd, _, err := send(r.net, ephCmixId,
					ciphertexts,
					cmix.GetDefaultCMIXParams())
				if err != nil {
					jww.ERROR.Printf(
						"[RPC] bad reply to %s,%s: %+v",
						ephCmixId, cMixMsg.GetKeyFP(),
						err)
					continue
				}
				jww.INFO.Printf("[RPC] reply to %s,%s "+
					"sent: %s",
					ephCmixId, cMixMsg.GetKeyFP(), rnd.ID)
				return
			}
		}

		// Response with Internal Server Error on crash
		defer func() {
			if r := recover(); r != nil {
				jww.ERROR.Printf("[RPC] crash with request: "+
					"%s, %s, %+v", cMixMsg.GetKeyFP(),
					base64.RawStdEncoding.EncodeToString(
						request),
					errors.Errorf("%+v", r))
				sendResp([]byte("Internal Server Error"))
			}
		}()

		// Otherwise send the response
		response := r.cb(ephCmixId, request)
		sendResp(response)
	}()
}
