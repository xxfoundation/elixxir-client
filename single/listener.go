package single

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix"
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

type Receiver interface {
	Callback(*Request, receptionID.EphemeralIdentity, rounds.Round)
}

type Listener interface {
	// Stop unregisters the listener
	Stop()
}

type listener struct {
	myId      *id.ID
	myPrivkey *cyclic.Int
	tag       string
	grp       *cyclic.Group
	cb        Receiver
	net       cmix.Client
}

func Listen(tag string, myId *id.ID, privkey *cyclic.Int, net cmix.Client,
	e2eGrp *cyclic.Group, cb Receiver) Listener {

	l := &listener{
		myId:      myId,
		myPrivkey: privkey,
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
	receptionID receptionID.EphemeralIdentity,
	round rounds.Round) {

	// Unmarshal the CMIX message contents to a transmission message
	transmitMsg, err := message.UnmarshalRequest(ecrMsg.GetContents(),
		l.grp.GetP().ByteLen())
	if err != nil {
		jww.WARN.Printf("failed to unmarshal contents on single use "+
			"request to %s on tag %s: %+v", l.myId, l.tag, err)
		return
	}

	// Generate DH key and symmetric key
	senderPubkey := transmitMsg.GetPubKey(l.grp)
	dhKey := l.grp.Exp(senderPubkey, l.myPrivkey,
		l.grp.NewInt(1))
	key := singleUse.NewTransmitKey(dhKey)

	// Verify the MAC
	if !singleUse.VerifyMAC(key, transmitMsg.GetPayload(), ecrMsg.GetMac()) {
		jww.WARN.Printf("mac check failed on single use request to %s "+
			"on tag %s", l.myId, l.tag)
		return
	}

	// Decrypt the transmission message payload
	fp := ecrMsg.GetKeyFP()
	decryptedPayload := cAuth.Crypt(key, fp[:24], transmitMsg.GetPayload())

	// Unmarshal payload
	payload, err := message.UnmarshalRequestPayload(decryptedPayload)
	if err != nil {
		jww.WARN.Printf("failed to unmarshal decrypted payload on "+
			"single use request to %s on tag %s: %+v", l.myId, l.tag, err)
		return
	}

	used := uint32(0)

	r := Request{
		sender:         payload.GetRID(transmitMsg.GetPubKey(l.grp)),
		senderPubKey:   senderPubkey,
		dhKey:          dhKey,
		tag:            l.tag,
		maxParts:       0,
		used:           &used,
		requestPayload: payload.GetContents(),
		net:            l.net,
	}

	go l.cb.Callback(&r, receptionID, round)
}

func (l *listener) Stop() {
	svc := cmixMsg.Service{
		Identifier: l.myId[:],
		Tag:        l.tag,
	}
	l.net.DeleteService(l.myId, svc, l)
}
