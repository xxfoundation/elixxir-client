////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package e2e

import (
	"fmt"

	"gitlab.com/elixxir/client/v4/e2e/ratchet/partner/session"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/primitives/format"
)

type processor struct {
	cy session.Cypher
	m  *manager
}

func (p *processor) Process(ecrMsg format.Message,
	receptionID receptionID.EphemeralIdentity,
	round rounds.Round) {
	jww.TRACE.Printf("[E2E] Process(ecrMsgDigest: %s)", ecrMsg.Digest())
	// ensure the key will be marked used before returning
	defer p.cy.Use()

	contents, residue, err := p.cy.Decrypt(ecrMsg)
	if err != nil {
		jww.ERROR.Printf("decrypt failed of %s (fp: %s), dropping: %+v",
			ecrMsg.Digest(), p.cy.Fingerprint(), err)
		return
	}

	sess := p.cy.GetSession()
	// todo: handle residue here
	message, _, done := p.m.partitioner.HandlePartition(sess.GetPartner(),
		contents, sess.GetRelationshipFingerprint(), residue)
	if done {
		message.RecipientID = receptionID.Source
		message.EphemeralID = receptionID.EphId
		message.Round = round
		message.Encrypted = true
		p.m.Switchboard.Speak(message)
	}
}

func (p *processor) String() string {
	return fmt.Sprintf("E2E(%s): %s",
		p.m.myID, p.cy.GetSession())
}
