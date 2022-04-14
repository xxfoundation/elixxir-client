package single

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/single/message"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/primitives/format"
)

type callbackWrapper func(payload []byte,
	receptionID receptionID.EphemeralIdentity, round rounds.Round, err error)

// responceProcessor Message.Processor interface registered with cmix.Client.
// One is registered for each potential fingerprint.
type responceProcessor struct {
	sendingID receptionID.EphemeralIdentity
	c         *message.Collator
	callback  callbackWrapper
	cy        cypher
	tag       string
	recipient *contact.Contact
}

// Process decrypts a response part and adds it to the collator - returning
// a full response to the callback when all parts are received.
func (rsp *responceProcessor) Process(ecrMsg format.Message,
	receptionID receptionID.EphemeralIdentity,
	round rounds.Round) {

	decrypted, err := rsp.cy.Decrypt(ecrMsg.GetContents(), ecrMsg.GetMac())
	if err != nil {
		jww.ERROR.Printf("Failed to decrypt payload for %s to %s, "+
			"single use may fail: %+v", rsp.tag, rsp.recipient.ID, err)
		return
	}

	payload, done, err := rsp.c.Collate(decrypted)
	if err != nil {
		jww.ERROR.Printf("Failed to collage payload for %s to %s, "+
			"single use may fail: %+v", rsp.tag, rsp.recipient.ID, err)
		return
	}

	if done {
		rsp.callback(payload, receptionID, round, nil)
	}
}
