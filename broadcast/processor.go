////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package broadcast

import (
	"encoding/binary"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	crypto "gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/elixxir/primitives/format"
)

// Error messages.
const (
	errDecrypt = "[BCAST] Failed to decrypt payload for broadcast %s (%q): %+v"
)

// processor handles channel message decryption and handling. This structure
// adheres to the Processor interface.
type processor struct {
	c      *crypto.Channel
	cb     ListenerFunc
	method Method
}

// Process decrypts the broadcast message and sends the results on the callback.
func (p *processor) Process(msg format.Message, tags []string, metadata []byte,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	// Handle external symmetric decryption
	payload, err := p.c.DecryptSymmetric(
		msg.GetContents(), msg.GetMac(), msg.GetKeyFP())
	if err != nil {
		jww.ERROR.Printf(errDecrypt, p.c.ReceptionID, p.c.Name, err)
		return
	}

	var md [2]byte
	copy(md[:], metadata)

	// Choose handling method
	switch p.method {
	case RSAToPublic:
		p.ProcessAdminMessage(payload, tags, md, receptionID, round)
	case Symmetric:
		p.cb(payload, msg.Marshal(), tags, md, receptionID, round)
	default:
		jww.FATAL.Panicf("Unrecognized broadcast method %d", p.method)
	}
}

// ProcessAdminMessage decrypts an admin message and sends the results on
// the callback.
func (p *processor) ProcessAdminMessage(innerCiphertext []byte,
	tags []string, metadata [2]byte, receptionID receptionID.EphemeralIdentity,
	round rounds.Round) {
	decrypted, err := p.c.DecryptRSAToPublicInner(innerCiphertext)
	if err != nil {
		jww.ERROR.Printf(errDecrypt, p.c.ReceptionID, p.c.Name, err)
		return
	}
	size := binary.BigEndian.Uint16(decrypted[:internalPayloadSizeLength])
	payload :=
		decrypted[internalPayloadSizeLength : size+internalPayloadSizeLength]

	p.cb(payload, innerCiphertext, tags, metadata, receptionID, round)
}

// String returns a string identifying the symmetricProcessor for debugging
// purposes.
func (p *processor) String() string {
	return "broadcastChannel-" + p.c.Name
}
