package e2e

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/network/historical"
	"gitlab.com/elixxir/client/network/identity/receptionID"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/primitives/format"
)

type UnsafeProcessor struct {
	m   *manager
	tag string
}

func (up *UnsafeProcessor) Process(ecrMsg format.Message, receptionID receptionID.EphemeralIdentity,
	round historical.Round) {
	//check if the message is unencrypted
	unencrypted, sender := e2e.IsUnencrypted(ecrMsg)
	if !unencrypted {
		jww.ERROR.Printf("Received a non unencrypted message in e2e "+
			"service %s, A message might have dropped!", up.tag)
	}

	//Parse
	message, done := up.m.partitioner.HandlePartition(sender,
		ecrMsg.GetContents(), nil)

	if done {
		message.RecipientID = receptionID.Source
		message.EphemeralID = receptionID.EphId
		message.Round = round
		message.Encrypted = false
		up.m.Switchboard.Speak(message)
	}
}
