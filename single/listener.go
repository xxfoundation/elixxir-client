package single

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	cmixMsg "gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/single/message"
	"gitlab.com/elixxir/crypto/cyclic"
	cAuth "gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/crypto/e2e/singleUse"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
)

const listenerProcessorName = "listenerProcessorName"

type Receiver interface {
	Callback(*Request, receptionID.EphemeralIdentity, []rounds.Round)
}

type Listener interface {
	// Stop unregisters the listener
	Stop()
}

type listener struct {
	myId      *id.ID
	myPrivKey *cyclic.Int
	tag       string
	grp       *cyclic.Group
	cb        Receiver
	net       CMix
}

// Listen allows a server to listen for single use requests. It will register a
// service relative to the tag and myID as the identifier. Only a single
// listener can be active for a tag-myID pair, and an error will be returned if
// that is violated. When requests are received, they will be called on the
// Receiver interface.
func Listen(tag string, myId *id.ID, privKey *cyclic.Int, net CMix,
	e2eGrp *cyclic.Group, cb Receiver) Listener {

	l := &listener{
		myId:      myId,
		myPrivKey: privKey,
		tag:       tag,
		grp:       e2eGrp,
		cb:        cb,
		net:       net,
	}

	svc := cmixMsg.Service{
		Identifier: myId[:],
		Tag:        tag,
		Metadata:   myId[:],
	}

	net.AddService(myId, svc, l)

	return l
}

func (l *listener) Process(ecrMsg format.Message,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	// Unmarshal the cMix message contents to a request message
	requestMsg, err := message.UnmarshalRequest(ecrMsg.GetContents(),
		l.grp.GetP().ByteLen())
	if err != nil {
		jww.WARN.Printf("Failed to unmarshal contents on single use "+
			"request to %s on tag %s: %+v", l.myId, l.tag, err)
		return
	}

	// Generate DH key and symmetric key
	senderPubkey := requestMsg.GetPubKey(l.grp)
	dhKey := l.grp.Exp(senderPubkey, l.myPrivKey, l.grp.NewInt(1))
	key := singleUse.NewRequestKey(dhKey)

	// Verify the MAC
	if !singleUse.VerifyMAC(key, requestMsg.GetPayload(), ecrMsg.GetMac()) {
		jww.WARN.Printf("mac check failed on single use request to %s "+
			"on tag %s", l.myId, l.tag)
		return
	}

	// Decrypt the request message payload
	fp := ecrMsg.GetKeyFP()
	decryptedPayload := cAuth.Crypt(key, fp[:24], requestMsg.GetPayload())
	// Unmarshal payload
	payload, err := message.UnmarshalRequestPayload(decryptedPayload)
	if err != nil {
		jww.WARN.Printf("[SU] Failed to unmarshal decrypted payload on "+
			"single use request to %s on tag %s: %+v", l.myId, l.tag, err)
		return
	}

	cbFunc := func(payloadContents []byte, rounds []rounds.Round) {
		used := uint32(0)

		r := Request{
			sender:         payload.GetRecipientID(requestMsg.GetPubKey(l.grp)),
			senderPubKey:   senderPubkey,
			dhKey:          dhKey,
			tag:            l.tag,
			maxParts:       payload.GetMaxResponseParts(),
			used:           &used,
			requestPayload: payloadContents,
			net:            l.net,
		}

		go l.cb.Callback(&r, receptionID, rounds)
	}

	if numParts := payload.GetNumParts(); numParts > 1 {
		c := message.NewCollator(numParts)
		_, _, err = c.Collate(payload)
		if err != nil {

			return
		}
		cyphers := makeCyphers(dhKey, numParts,
			singleUse.NewRequestPartKey, singleUse.NewRequestPartFingerprint)
		ridCollector := newRoundIdCollector(int(numParts))
		for i, cy := range cyphers {
			key = singleUse.NewRequestPartKey(dhKey, uint64(i+1))
			fp = singleUse.NewRequestPartFingerprint(dhKey, uint64(i+1))
			p := &requestPartProcessor{
				myId:     l.myId,
				tag:      l.tag,
				cb:       cbFunc,
				c:        c,
				cy:       cy,
				roundIDs: ridCollector,
			}
			err = l.net.AddFingerprint(l.myId, fp, p)
			if err != nil {
				jww.ERROR.Printf("Failed to add fingerprint for request part "+
					"%d of %d (%s): %+v", i, numParts, l.tag, err)
				return
			}
		}

		l.net.CheckInProgressMessages()
	} else {
		cbFunc(payload.GetContents(), []rounds.Round{round})
	}
}

func (l *listener) String() string {
	return listenerProcessorName

}

func (l *listener) Stop() {
	svc := cmixMsg.Service{
		Identifier: l.myId[:],
		Tag:        l.tag,
	}
	l.net.DeleteService(l.myId, svc, l)
}
