///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package auth

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/preimage"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage/auth"
	"gitlab.com/elixxir/client/storage/edge"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	cAuth "gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/elixxir/primitives/format"
	"strings"
)

func (m *Manager) StartProcesses() (stoppable.Stoppable, error) {
	stop := stoppable.NewSingle("Auth")

	go func() {
		for {
			select {
			case <-stop.Quit():
				stop.ToStopped()
				return
			case msg := <-m.rawMessages:
				m.processAuthMessage(msg)
			}
		}
	}()

	return stop, nil
}

func (m *Manager) processAuthMessage(msg message.Receive) {
	authStore := m.storage.Auth()
	//lookup the message, check if it is an auth request
	cmixMsg := format.Unmarshal(msg.Payload)
	fp := cmixMsg.GetKeyFP()
	jww.INFO.Printf("RAW AUTH FP: %v", fp)
	// this takes the request lock if it is a specific fp, all
	// exits after this need to call fail or delete if it is
	// specific
	fpType, sr, myHistoricalPrivKey, err := authStore.GetFingerprint(fp)
	if err != nil {
		jww.TRACE.Printf("FINGERPRINT FAILURE: %s", err.Error())
		// if the lookup fails, ignore the message. It is
		// likely garbled or for a different protocol
		return
	}

	//denote that the message is not garbled
	m.storage.GetGarbledMessages().Remove(cmixMsg)
	grp := m.storage.E2e().GetGroup()

	switch fpType {
	case auth.General:
		// if it is general, that means a new request has
		// been received
		m.handleRequest(cmixMsg, myHistoricalPrivKey, grp)
	case auth.Specific:
		// if it is specific, that means the original request was sent
		// by this users and a confirmation has been received
		jww.INFO.Printf("Received AuthConfirm from %s, msgDigest: %s",
			sr.GetPartner(), cmixMsg.Digest())
		m.handleConfirm(cmixMsg, sr, grp)
	}
}

func (m *Manager) handleRequest(cmixMsg format.Message,
	myHistoricalPrivKey *cyclic.Int, grp *cyclic.Group) {
	//decode the outer format
	baseFmt, partnerPubKey, err := handleBaseFormat(cmixMsg, grp)
	if err != nil {
		jww.WARN.Printf("Failed to handle auth request: %s", err)
		return
	}

	myPubKey := diffieHellman.GeneratePublicKey(myHistoricalPrivKey, grp)

	jww.TRACE.Printf("handleRequest MYPUBKEY: %v", myPubKey.Bytes())
	jww.TRACE.Printf("handleRequest PARTNERPUBKEY: %v", partnerPubKey.Bytes())

	//decrypt the message
	jww.TRACE.Printf("handleRequest SALT: %v", baseFmt.GetSalt())
	jww.TRACE.Printf("handleRequest ECRPAYLOAD: %v", baseFmt.GetEcrPayload())
	jww.TRACE.Printf("handleRequest MAC: %v", cmixMsg.GetMac())

	success, payload := cAuth.Decrypt(myHistoricalPrivKey,
		partnerPubKey, baseFmt.GetSalt(), baseFmt.GetEcrPayload(),
		cmixMsg.GetMac(), grp)

	if !success {
		jww.WARN.Printf("Received auth request failed " +
			"its mac check")
		return
	}

	//decode the ecr format
	ecrFmt, err := unmarshalEcrFormat(payload)
	if err != nil {
		jww.WARN.Printf("Failed to unmarshal auth "+
			"request's encrypted payload: %s", err)
		return
	}

	//decode the request format
	requestFmt, err := newRequestFormat(ecrFmt)
	if err != nil {
		jww.WARN.Printf("Failed to unmarshal auth "+
			"request's internal payload: %s", err)
		return
	}

	partnerID, err := requestFmt.GetID()
	if err != nil {
		jww.WARN.Printf("Failed to unmarshal auth "+
			"request's sender ID: %s", err)
		return
	}

	events := m.net.GetEventManager()
	em := fmt.Sprintf("Received AuthRequest from %s,"+
		" msgDigest: %s", partnerID, cmixMsg.Digest())
	jww.INFO.Print(em)
	events.Report(1, "Auth", "RequestReceived", em)

	/*do state edge checks*/
	// check if a relationship already exists.
	// if it does and the keys used are the same as we have, send a
	// confirmation in case there are state issues.
	// do not store
	if _, err := m.storage.E2e().GetPartner(partnerID); err == nil {
		em := fmt.Sprintf("Received Auth request for %s, "+
			"channel already exists. Ignoring", partnerID)
		jww.WARN.Print(em)
		events.Report(5, "Auth", "RequestIgnored", em)
		//exit
		return
	} else {
		//check if the relationship already exists,
		rType, sr2, _, err := m.storage.Auth().GetRequest(partnerID)
		if err != nil && !strings.Contains(err.Error(), auth.NoRequest) {
			// if another error is received, print it and exit
			em := fmt.Sprintf("Received new Auth request for %s, "+
				"internal lookup produced bad result: %+v",
				partnerID, err)
			jww.ERROR.Print(em)
			events.Report(10, "Auth", "RequestError", em)
			return
		} else {
			//handle the events where the relationship already exists
			switch rType {
			// if this is a duplicate, ignore the message
			case auth.Receive:
				em := fmt.Sprintf("Received new Auth request for %s, "+
					"is a duplicate", partnerID)
				jww.WARN.Print(em)
				events.Report(5, "Auth", "DuplicateRequest", em)
				return
			// if we sent a request, then automatically confirm
			// then exit, nothing else needed
			case auth.Sent:
				jww.INFO.Printf("Received AuthRequest from %s,"+
					" msgDigest: %s which has been requested, auto-confirming",
					partnerID, cmixMsg.Digest())
				// do the confirmation
				if err := m.doConfirm(sr2, grp, partnerPubKey, m.storage.E2e().GetDHPrivateKey(),
					sr2.GetPartnerHistoricalPubKey(), ecrFmt.GetOwnership()); err != nil {
					em := fmt.Sprintf("Auto Confirmation with %s failed: %s",
						partnerID, err)
					jww.WARN.Print(em)
					events.Report(10, "Auth",
						"RequestError", em)
				}
				//exit
				return
			}
		}
	}

	//process the inner payload
	facts, msg, err := fact.UnstringifyFactList(
		string(requestFmt.msgPayload))
	if err != nil {
		em := fmt.Sprintf("failed to parse facts and message "+
			"from Auth Request: %s", err)
		jww.WARN.Print(em)
		events.Report(10, "Auth", "RequestError", em)
		return
	}

	//create the contact
	c := contact.Contact{
		ID:             partnerID,
		DhPubKey:       partnerPubKey,
		OwnershipProof: copySlice(ecrFmt.ownership),
		Facts:          facts,
	}

	// fixme: the client will never be notified of the channel creation if a
	// crash occurs after the store but before the conclusion of the callback
	//create the auth storage
	if err = m.storage.Auth().AddReceived(c); err != nil {
		em := fmt.Sprintf("failed to store contact Auth "+
			"Request: %s", err)
		jww.WARN.Print(em)
		events.Report(10, "Auth", "RequestError", em)
		return
	}

	//  fixme: if a crash occurs before or during the calls, the notification
	//  will never be sent.
	cbList := m.requestCallbacks.Get(c.ID)
	for _, cb := range cbList {
		rcb := cb.(interfaces.RequestCallback)
		go rcb(c, msg)
	}
	return
}

func (m *Manager) handleConfirm(cmixMsg format.Message, sr *auth.SentRequest,
	grp *cyclic.Group) {
	events := m.net.GetEventManager()

	// check if relationship already exists
	if mgr, err := m.storage.E2e().GetPartner(sr.GetPartner()); mgr != nil || err == nil {
		em := fmt.Sprintf("Cannot confirm auth for %s, channel already "+
			"exists.", sr.GetPartner())
		jww.WARN.Print(em)
		events.Report(10, "Auth", "ConfirmError", em)
		m.storage.Auth().Done(sr.GetPartner())
		return
	}

	// extract the message
	baseFmt, partnerPubKey, err := handleBaseFormat(cmixMsg, grp)
	if err != nil {
		em := fmt.Sprintf("Failed to handle auth confirm: %s", err)
		jww.WARN.Print(em)
		events.Report(10, "Auth", "ConfirmError", em)
		m.storage.Auth().Done(sr.GetPartner())
		return
	}

	jww.TRACE.Printf("handleConfirm PARTNERPUBKEY: %v", partnerPubKey.Bytes())
	jww.TRACE.Printf("handleConfirm SRMYPUBKEY: %v", sr.GetMyPubKey().Bytes())

	// decrypt the payload
	jww.TRACE.Printf("handleConfirm SALT: %v", baseFmt.GetSalt())
	jww.TRACE.Printf("handleConfirm ECRPAYLOAD: %v", baseFmt.GetEcrPayload())
	jww.TRACE.Printf("handleConfirm MAC: %v", cmixMsg.GetMac())
	success, payload := cAuth.Decrypt(sr.GetMyPrivKey(),
		partnerPubKey, baseFmt.GetSalt(), baseFmt.GetEcrPayload(),
		cmixMsg.GetMac(), grp)

	if !success {
		em := fmt.Sprintf("Received auth confirmation failed its mac " +
			"check")
		jww.WARN.Print(em)
		events.Report(10, "Auth", "ConfirmError", em)
		m.storage.Auth().Done(sr.GetPartner())
		return
	}

	ecrFmt, err := unmarshalEcrFormat(payload)
	if err != nil {
		em := fmt.Sprintf("Failed to unmarshal auth confirmation's "+
			"encrypted payload: %s", err)
		jww.WARN.Print(em)
		events.Report(10, "Auth", "ConfirmError", em)
		m.storage.Auth().Done(sr.GetPartner())
		return
	}

	// finalize the confirmation
	if err := m.doConfirm(sr, grp, partnerPubKey, sr.GetMyPrivKey(),
		sr.GetPartnerHistoricalPubKey(), ecrFmt.GetOwnership()); err != nil {
		em := fmt.Sprintf("Confirmation failed: %s", err)
		jww.WARN.Print(em)
		events.Report(10, "Auth", "ConfirmError", em)
		m.storage.Auth().Done(sr.GetPartner())
		return
	}
}

func (m *Manager) doConfirm(sr *auth.SentRequest, grp *cyclic.Group,
	partnerPubKey, myPrivateKeyOwnershipProof, partnerPubKeyOwnershipProof *cyclic.Int, ownershipProof []byte) error {
	// verify the message came from the intended recipient
	if !cAuth.VerifyOwnershipProof(myPrivateKeyOwnershipProof,
		partnerPubKeyOwnershipProof, grp, ownershipProof) {
		return errors.Errorf("Failed authenticate identity for auth "+
			"confirmation of %s", sr.GetPartner())
	}

	// fixme: channel can get into a bricked state if the first save occurs and
	// the second does not
	p := m.storage.E2e().GetE2ESessionParams()
	if err := m.storage.E2e().AddPartner(sr.GetPartner(),
		partnerPubKey, sr.GetMyPrivKey(), p, p); err != nil {
		return errors.Errorf("Failed to create channel with partner (%s) "+
			"after confirmation: %+v",
			sr.GetPartner(), err)
	}

	//remove the confirm fingerprint
	fp := sr.GetFingerprint()
	if err := m.storage.GetEdge().Remove(edge.Preimage{
		Data:   preimage.Generate(fp[:], preimage.Confirm),
		Type:   preimage.Confirm,
		Source: sr.GetPartner()[:],
	}, m.storage.GetUser().ReceptionID); err != nil {
		jww.WARN.Printf("Failed delete the preimage for confirm from %s: %+v",
			sr.GetPartner(), err)
	}

	//add the e2e and rekey firngeprints
	//e2e
	sessionPartner, err := m.storage.E2e().GetPartner(sr.GetPartner())
	if err != nil {
		jww.FATAL.Panicf("Cannot find %s right after creating: %+v", sr.GetPartner(), err)
	}
	me := m.storage.GetUser().ReceptionID

	m.storage.GetEdge().Add(edge.Preimage{
		Data:   sessionPartner.GetE2EPreimage(),
		Type:   preimage.E2e,
		Source: sr.GetPartner()[:],
	}, me)

	//silent (rekey)
	m.storage.GetEdge().Add(edge.Preimage{
		Data:   sessionPartner.GetSilentPreimage(),
		Type:   preimage.Silent,
		Source: sr.GetPartner()[:],
	}, me)

	// File transfer end
	m.storage.GetEdge().Add(edge.Preimage{
		Data:   sessionPartner.GetFileTransferPreimage(),
		Type:   preimage.EndFT,
		Source: sr.GetPartner()[:],
	}, me)

	// delete the in progress negotiation
	// this undoes the request lock
	if err := m.storage.Auth().Delete(sr.GetPartner()); err != nil {
		return errors.Errorf("UNRECOVERABLE! Failed to delete in "+
			"progress negotiation with partner (%s) after confirmation: %+v",
			sr.GetPartner(), err)
	}

	//notify the end point
	c := contact.Contact{
		ID:             sr.GetPartner().DeepCopy(),
		DhPubKey:       partnerPubKey.DeepCopy(),
		OwnershipProof: copySlice(ownershipProof),
		Facts:          make([]fact.Fact, 0),
	}

	//  fixme: if a crash occurs before or during the calls, the notification
	//  will never be sent.
	cbList := m.confirmCallbacks.Get(c.ID)
	for _, cb := range cbList {
		ccb := cb.(interfaces.ConfirmCallback)
		go ccb(c)
	}

	m.net.CheckGarbledMessages()

	return nil
}

func copySlice(s []byte) []byte {
	c := make([]byte, len(s))
	copy(c, s)
	return c
}

func handleBaseFormat(cmixMsg format.Message, grp *cyclic.Group) (baseFormat,
	*cyclic.Int, error) {

	baseFmt, err := unmarshalBaseFormat(cmixMsg.GetContents(),
		grp.GetP().ByteLen())
	if err != nil {
		return baseFormat{}, nil, errors.WithMessage(err, "Failed to"+
			" unmarshal auth")
	}

	if !grp.BytesInside(baseFmt.pubkey) {
		return baseFormat{}, nil, errors.WithMessage(err, "Received "+
			"auth confirmation public key is not in the e2e cyclic group")
	}
	partnerPubKey := grp.NewIntFromBytes(baseFmt.pubkey)
	return baseFmt, partnerPubKey, nil
}
