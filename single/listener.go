////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package single

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v5/cmix/identity/receptionID"
	cMixMsg "gitlab.com/elixxir/client/v5/cmix/message"
	"gitlab.com/elixxir/client/v5/cmix/rounds"
	"gitlab.com/elixxir/client/v5/single/message"
	"gitlab.com/elixxir/crypto/cyclic"
	cAuth "gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/crypto/e2e/singleUse"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"strings"
)

type listener struct {
	tag       string
	grp       *cyclic.Group
	myID      *id.ID
	myPrivKey *cyclic.Int
	cb        Receiver
	net       ListenCmix
}

// Listen allows a server to listen for single use requests. It will register a
// service relative to the tag and myID as the identifier. Only a single
// listener can be active for a tag-myID pair, and an error will be returned if
// that is violated. When requests are received, they will be called on the
// Receiver interface.
func Listen(tag string, myID *id.ID, privKey *cyclic.Int, net ListenCmix,
	e2eGrp *cyclic.Group, cb Receiver) Listener {

	l := &listener{
		tag:       tag,
		grp:       e2eGrp,
		myID:      myID,
		myPrivKey: privKey,
		cb:        cb,
		net:       net,
	}

	svc := cMixMsg.Service{
		Identifier: myID[:],
		Tag:        tag,
		Metadata:   myID[:],
	}

	net.AddService(myID, svc, l)

	return l
}

// Process decrypts and collates the encrypted single-use request message.
func (l *listener) Process(ecrMsg format.Message,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	err := l.process(ecrMsg, receptionID, round)
	if err != nil {
		jww.ERROR.Printf(
			"[SU] Failed to process single-use request to %s on tag %q: %+v",
			l.myID, l.tag, err)
	}
}

// process is a helper functions for Process that returns errors for easier
// testing.
func (l *listener) process(ecrMsg format.Message,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) error {
	// Unmarshal the cMix message contents to a request message
	request, err := message.UnmarshalRequest(ecrMsg.GetContents(),
		l.grp.GetP().ByteLen())
	if err != nil {
		return errors.Errorf("could not unmarshal contents: %+v", err)
	}

	// Generate DH key and symmetric key
	senderPubkey := request.GetPubKey(l.grp)
	dhKey := l.grp.Exp(senderPubkey, l.myPrivKey, l.grp.NewInt(1))
	key := singleUse.NewRequestKey(dhKey)

	// Verify the MAC
	if !singleUse.VerifyMAC(key, request.GetPayload(), ecrMsg.GetMac()) {
		return errors.New("failed to verify MAC")
	}

	// Decrypt the request message payload
	fp := ecrMsg.GetKeyFP()
	decryptedPayload := cAuth.Crypt(key, fp[:24], request.GetPayload())

	// Unmarshal payload
	requestPayload, err := message.UnmarshalRequestPayload(decryptedPayload)
	if err != nil {
		return errors.Errorf("could not unmarshal decrypted payload: %+v", err)
	}

	cbFunc := func(payloadContents []byte, rounds []rounds.Round) {
		used := uint32(0)
		r := Request{
			sender:         requestPayload.GetRecipientID(request.GetPubKey(l.grp)),
			senderPubKey:   senderPubkey,
			dhKey:          dhKey,
			tag:            l.tag,
			maxParts:       requestPayload.GetMaxResponseParts(),
			used:           &used,
			requestPayload: payloadContents,
			net:            l.net,
		}

		go l.cb.Callback(&r, receptionID, rounds)
	}

	if numParts := requestPayload.GetNumRequestParts(); numParts > 1 {
		c := message.NewCollator(numParts)
		_, _, err = c.Collate(requestPayload)
		if err != nil {
			return errors.Errorf("could not collate initial payload: %+v", err)
		}

		cyphers := makeCyphers(dhKey, numParts,
			singleUse.NewRequestPartKey, singleUse.NewRequestPartFingerprint)
		ridCollector := newRoundIdCollector(int(numParts))

		for i, cy := range cyphers {
			p := &requestPartProcessor{
				myId:     l.myID,
				tag:      l.tag,
				cb:       cbFunc,
				c:        c,
				cy:       cy,
				roundIDs: ridCollector,
			}

			err = l.net.AddFingerprint(l.myID, cy.getFingerprint(), p)
			if err != nil {
				return errors.Errorf("could not add fingerprint for single-"+
					"use request part %d of %d: %+v", i, numParts, err)
			}
		}

		l.net.CheckInProgressMessages()
	} else {
		cbFunc(requestPayload.GetContents(), []rounds.Round{round})
	}

	return nil
}

// Stop stops the listener from receiving messages.
func (l *listener) Stop() {
	svc := cMixMsg.Service{
		Identifier: l.myID[:],
		Tag:        l.tag,
	}
	l.net.DeleteService(l.myID, svc, l)
}

// String prints a name that identifies this single use listener. Adheres to the
// fmt.Stringer interface.
func (l *listener) String() string {
	return "SingleUse(" + l.myID.String() + ")"
}

// GoString prints the fields of the listener in a human-readable form.
// Adheres to the fmt.GoStringer interface and prints values passed as an
// operand to a %#v format.
func (l *listener) GoString() string {
	cb := "<nil>"
	if l.cb != nil {
		cb = fmt.Sprintf("%p", l.cb)
	}
	net := "<nil>"
	if l.net != nil {
		net = fmt.Sprintf("%p", l.net)
	}
	fields := []string{
		"tag:" + fmt.Sprintf("%q", l.tag),
		"grp:" + l.grp.GetFingerprintText(),
		"myID:" + l.myID.String(),
		"myPrivKey:\"" + l.myPrivKey.Text(10) + "\"",
		"cb:" + cb,
		"net:" + net,
	}

	return strings.Join(fields, " ")
}
