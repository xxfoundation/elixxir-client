package e2e

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/network/historical"
	"gitlab.com/elixxir/client/network/identity/receptionID"
	"gitlab.com/elixxir/primitives/format"
)

type processor struct {
	cy *session.Cypher
	m  *manager
}

func (p *processor) Process(ecrMsg format.Message, receptionID receptionID.EphemeralIdentity,
	round historical.Round) {
	//ensure the key is used before returning
	defer p.cy.Use()

	//decrypt
	contents, err := p.cy.Decrypt(ecrMsg)
	if err != nil {
		jww.ERROR.Printf("Decryption Failed of %s (fp: %s), dropping: %+v",
			ecrMsg.Digest(), p.cy.Fingerprint(), err)
		return
	}

	//Parse
	sess := p.cy.GetSession()
	message, done := p.m.partitioner.HandlePartition(sess.GetPartner(),
		contents, sess.GetRelationshipFingerprint())

	if done {
		message.RecipientID = receptionID.Source
		message.EphemeralID = receptionID.EphId
		message.Round = round
		message.Encrypted = true
		p.m.Switchboard.Speak(message)
	}
}
