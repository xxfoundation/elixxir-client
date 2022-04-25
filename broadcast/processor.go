////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package broadcast

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	crypto "gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/elixxir/primitives/format"
)

// Error messages.
const (
	errDecrypt = "[BCAST] Failed to decrypt payload for broadcast %s (%q): %+v"
)

// processor manages the reception and decryption of a broadcast message.
// Adheres to the message.Processor interface.
type processor struct {
	s  *crypto.Symmetric
	cb ListenerFunc
}

// Process decrypts the broadcast message and sends the results on the callback.
func (p *processor) Process(msg format.Message,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {

	payload, err := p.s.Decrypt(msg.GetContents(), msg.GetMac(), msg.GetKeyFP())
	if err != nil {
		jww.ERROR.Printf(errDecrypt, p.s.ReceptionID, p.s.Name, err)
		return
	}

	go p.cb(payload, receptionID, round)
}

// String returns a string identifying the processor for debugging purposes.
func (p *processor) String() string {
	return "symmetricChannel-" + p.s.Name
}
