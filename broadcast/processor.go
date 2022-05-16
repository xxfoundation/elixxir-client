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

type processor struct {
	c      *crypto.Channel
	cb     ListenerFunc
	method Method
}

// Process decrypts the broadcast message and sends the results on the callback.
func (p *processor) Process(msg format.Message,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {

	var payload []byte
	var err error
	switch p.method {
	case Asymmetric:
		unsizedPayload, err := DecodeSizedBroadcast(msg.GetContents())
		if err != nil {
			jww.ERROR.Printf("Failed to decode sized broadcast: %+v", err)
			return
		}
		payload, err = p.c.DecryptAsymmetric(unsizedPayload)
		if err != nil {
			jww.ERROR.Printf(errDecrypt, p.c.ReceptionID, p.c.Name, err)
			return
		}
	case Symmetric:
		payload, err = p.c.DecryptSymmetric(msg.GetContents(), msg.GetMac(), msg.GetKeyFP())
		if err != nil {
			jww.ERROR.Printf(errDecrypt, p.c.ReceptionID, p.c.Name, err)
			return
		}
	default:
		jww.ERROR.Printf("Unrecognized broadcast method %d", p.method)
	}

	go p.cb(payload, receptionID, round)
}

// String returns a string identifying the symmetricProcessor for debugging purposes.
func (p *processor) String() string {
	return "broadcastChannel-" + p.c.Name
}
