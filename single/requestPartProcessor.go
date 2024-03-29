////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package single

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/single/message"
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

func (rpp *requestPartProcessor) Process(msg format.Message, tags []string,
	_ []byte, _ receptionID.EphemeralIdentity, round rounds.Round) {

	decrypted, err := rpp.cy.decrypt(msg.GetContents(), msg.GetMac())
	if err != nil {
		jww.ERROR.Printf("[SU] Failed to decrypt single-use request payload "+
			"%d (%s): %+v", rpp.cy.num, rpp.tag, err)
		return
	}

	requestPart, err := message.UnmarshalRequestPart(decrypted)
	if err != nil {
		jww.ERROR.Printf("[SU] Failed to unmarshal single-use request part "+
			"payload %d (%s): %+v", rpp.cy.num, rpp.tag, err)
		return
	}

	payload, done, err := rpp.c.Collate(requestPart)
	if err != nil {
		jww.ERROR.Printf("[SU] Failed to collate single-use request payload "+
			"%d (%s): %+v", rpp.cy.num, rpp.tag, err)
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
