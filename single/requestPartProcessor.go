////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package single

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/single/message"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
)

const requestPartProcessorName = "requestPartProcessor"

// requestPartProcessor handles the decryption and collation of request parts.
type requestPartProcessor struct {
	myId     *id.ID
	tag      string
	cb       func(payloadContents []byte, rounds []rounds.Round)
	c        *message.Collator
	cy       cypher
	roundIDs *roundCollector
}

func (rpp *requestPartProcessor) Process(msg format.Message,
	_ receptionID.EphemeralIdentity, round rounds.Round) {

	decrypted, err := rpp.cy.decrypt(msg.GetContents(), msg.GetMac())
	if err != nil {
		jww.ERROR.Printf("[SU] Failed to decrypt single-use request payload "+
			"for %s to %s: %+v", rpp.tag, rpp.myId, err)
		return
	}

	requestPart, err := message.UnmarshalRequestPart(decrypted)
	if err != nil {
		jww.ERROR.Printf("[SU] Failed to unmarshal single-use request part "+
			"payload for %s to %s: %+v", rpp.tag, rpp.myId, err)
		return
	}

	payload, done, err := rpp.c.Collate(requestPart)
	if err != nil {
		jww.ERROR.Printf("[SU] Failed to collate single-use request payload "+
			"for %s to %s: %+v", rpp.tag, rpp.myId, err)
		return
	}

	rpp.roundIDs.add(round)

	if done {
		rpp.cb(payload, rpp.roundIDs.getList())
	}
}

func (rpp *requestPartProcessor) String() string {
	return requestPartProcessorName
}
