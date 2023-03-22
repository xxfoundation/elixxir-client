////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package single

import (
	"fmt"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/single/message"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/primitives/format"
)

type callbackWrapper func(payload []byte,
	receptionID receptionID.EphemeralIdentity, rounds []rounds.Round, err error)

// responseProcessor is registered for each potential fingerprint. Adheres to
// the message.Processor interface registered with cmix.Client
type responseProcessor struct {
	sendingID receptionID.EphemeralIdentity
	c         *message.Collator
	callback  callbackWrapper
	cy        cypher
	tag       string
	recipient *contact.Contact
	roundIDs  *roundCollector
}

// Process decrypts a response part and adds it to the collator - returning
// a full response to the callback when all parts are received.
func (rsp *responseProcessor) Process(ecrMsg format.Message, tags []string,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {

	decrypted, err := rsp.cy.decrypt(ecrMsg.GetContents(), ecrMsg.GetMac())
	if err != nil {
		jww.ERROR.Printf("[SU] Failed to decrypt single-use response "+
			"payload for %s to %s: %+v",
			rsp.tag, rsp.recipient.ID, err)
		return
	}

	responsePart, err := message.UnmarshalResponsePart(decrypted)
	if err != nil {
		jww.ERROR.Printf("[SU] Failed to unmarshal single-use response part "+
			"payload for %s to %s: %+v", rsp.tag, rsp.recipient.ID, err)
		return
	}

	payload, done, err := rsp.c.Collate(responsePart)
	if err != nil {
		jww.ERROR.Printf("[SU] Failed to collate single-use response payload "+
			"for %s to %s, single use may fail: %+v",
			rsp.tag, rsp.recipient.ID, err)
		return
	}

	rsp.roundIDs.add(round)

	if done {
		rsp.callback(payload, receptionID, rsp.roundIDs.getList(), nil)
	}
}

func (rsp *responseProcessor) String() string {
	return fmt.Sprintf("SingleUseFP(%s, %s)",
		rsp.sendingID, rsp.recipient.ID)
}
