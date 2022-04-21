package e2e

import (
	"fmt"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/primitives/format"
)

type UnsafeProcessor struct {
	m   *manager
	tag string
}

func (up *UnsafeProcessor) Process(ecrMsg format.Message,
	receptionID receptionID.EphemeralIdentity,
	round rounds.Round) {
	//check if the message is unencrypted
	jww.INFO.Printf("Unsafe PRocessed received: contents: %v, fp: %v, mac: %v, sih: %v",
		ecrMsg.GetContents(), ecrMsg.GetKeyFP(), ecrMsg.GetMac(), ecrMsg.GetSIH())
	unencrypted, sender := e2e.IsUnencrypted(ecrMsg)
	if !unencrypted && sender == nil {
		jww.ERROR.Printf("unencrypted message failed MAC check: %v",
			ecrMsg)
		return
	}
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

func (up *UnsafeProcessor) String() string {
	return fmt.Sprintf("Unsafe(%s)", up.m.myID)
}
