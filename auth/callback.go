package auth

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/contact"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage/auth"
	"gitlab.com/elixxir/client/storage/e2e"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	cAuth "gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/primitives/format"
	"strings"
)

func (m *Manager) StartProcessies() stoppable.Stoppable {

	stop := stoppable.NewSingle("Auth")
	authStore := m.storage.Auth()
	grp := m.storage.E2e().GetGroup()

	go func() {
		select {
		case <-stop.Quit():
			return
		case msg := <-m.rawMessages:
			//lookup the message, check if it is an auth request
			cmixMsg := format.Unmarshal(msg.Payload)
			fp := cmixMsg.GetKeyFP()
			jww.INFO.Printf("RAW AUTH FP: %v", fp)
			// this takes the request lock if it is a specific fp,
			// all exits after this need to call fail or Delete if it is
			// specific
			fpType, sr, myHistoricalPrivKey, err := authStore.GetFingerprint(fp)
			if err != nil {
				// if the lookup fails, ignore the message. It is likely
				// garbled or for a different protocol
				break
			}

			//denote that the message is not garbled
			m.storage.GetGarbledMessages().Remove(cmixMsg)

			switch fpType {
			// if it is general, that means a new request has been received
			case auth.General:
				m.handleRequest(cmixMsg, myHistoricalPrivKey, grp)
			// if it is specific, that means the original request was sent
			// by this users and a confirmation has been received
			case auth.Specific:
				m.handleConfirm(cmixMsg, sr, grp)
			}
		}
	}()
	return stop
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

	jww.INFO.Printf("handleRequest MYPUBKEY: %v", myPubKey.Bytes())
	jww.INFO.Printf("handleRequest PARTNERPUBKEY: %v", partnerPubKey.Bytes())

	//decrypt the message
	success, payload := cAuth.Decrypt(myHistoricalPrivKey,
		partnerPubKey, baseFmt.GetSalt(), baseFmt.GetEcrPayload(),
		cmixMsg.GetMac(), grp)

	if !success {
		jww.WARN.Printf("Recieved auth request failed " +
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

	/*do state edge checks*/
	// check if a relationship already exists.
	// if it does and the keys used are the same as we have, send a
	// confirmation in case there are state issues.
	// do not store
	if _, err := m.storage.E2e().GetPartner(partnerID); err == nil {
		jww.WARN.Printf("Recieved Auth request for %s, "+
			"channel already exists. Ignoring", partnerID)
		//exit
		return
	} else {
		//check if the relationship already exists,
		rType, sr2, _, err := m.storage.Auth().GetRequest(partnerID)
		if err != nil && !strings.Contains(err.Error(), auth.NoRequest) {
			// if another error is recieved, print it and exit
			jww.WARN.Printf("Recieved new Auth request for %s, "+
				"internal lookup produced bad result: %+v",
				partnerID, err)
			return
		} else {
			//handle the events where the relationship already exists
			switch rType {
			// if this is a duplicate, ignore the message
			case auth.Receive:
				jww.WARN.Printf("Recieved new Auth request for %s, "+
					"is a duplicate", partnerID)
				return
			// if we sent a request, then automatically confirm
			// then exit, nothing else needed
			case auth.Sent:
				// do the confirmation
				if err := m.doConfirm(sr2, grp, partnerPubKey,
					ecrFmt.GetOwnership()); err != nil {
					jww.WARN.Printf("Confirmation failed: %s", err)
				}
				//exit
				return
			}
		}
	}

	//process the inner payload
	facts, msg, err := contact.UnstringifyFactList(
		string(requestFmt.msgPayload))
	if err != nil {
		jww.WARN.Printf("failed to parse facts and message "+
			"from Auth Request: %s", err)
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
	// create the auth storage
	if err = m.storage.Auth().AddReceived(c); err != nil {
		jww.WARN.Printf("failed to store contact Auth "+
			"Request: %s", err)
		return
	}

	//  fixme: if a crash occurs before or during the calls, the notification
	//  will never be sent.
	cbList := m.requestCallbacks.Get(c.ID)
	for i:=0;i<len(cbList);i++{
		cb := cbList[i]
		jww.INFO.Printf("callback type: %T", cb)
		jww.INFO.Printf("printed internal callback: %#v", cb)
		rcb := (cb).([]interface{})[0].(interfaces.RequestCallback)
		go rcb(c, msg)
	}
	return
}

func (m *Manager) handleConfirm(cmixMsg format.Message, sr *auth.SentRequest,
	grp *cyclic.Group) {
	// check if relationship already exists
	if mgr, err := m.storage.E2e().GetPartner(sr.GetPartner()); mgr != nil || err == nil {
		jww.WARN.Printf("Cannot confirm auth for %s, channel already "+
			"exists.", sr.GetPartner())
		m.storage.Auth().Fail(sr.GetPartner())
		return
	}

	// extract the message
	baseFmt, partnerPubKey, err := handleBaseFormat(cmixMsg, grp)
	if err != nil {
		jww.WARN.Printf("Failed to handle auth confirm: %s", err)
		m.storage.Auth().Fail(sr.GetPartner())
		return
	}

	jww.INFO.Printf("handleConfirm PARTNERPUBKEY: %v", partnerPubKey.Bytes())
	jww.INFO.Printf("handleConfirm SRMYPUBKEY: %v", sr.GetMyPubKey().Bytes())

	// decrypt the payload
	success, payload := cAuth.Decrypt(sr.GetMyPrivKey(),
		partnerPubKey, baseFmt.GetSalt(), baseFmt.GetEcrPayload(),
		cmixMsg.GetMac(), grp)

	if !success {
		jww.WARN.Printf("Recieved auth confirmation failed its mac " +
			"check")
		m.storage.Auth().Fail(sr.GetPartner())
		return
	}

	ecrFmt, err := unmarshalEcrFormat(payload)
	if err != nil {
		jww.WARN.Printf("Failed to unmarshal auth confirmation's "+
			"encrypted payload: %s", err)
		m.storage.Auth().Fail(sr.GetPartner())
		return
	}

	// finalize the confirmation
	if err := m.doConfirm(sr, grp, partnerPubKey, ecrFmt.GetOwnership()); err != nil {
		jww.WARN.Printf("Confirmation failed: %s", err)
		m.storage.Auth().Fail(sr.GetPartner())
		return
	}
}

func (m *Manager) doConfirm(sr *auth.SentRequest, grp *cyclic.Group,
	partnerPubKey *cyclic.Int, ownershipProof []byte) error {
	// verify the message came from the intended recipient
	if !cAuth.VerifyOwnershipProof(sr.GetMyPrivKey(),
		sr.GetPartnerHistoricalPubKey(), grp, ownershipProof) {
		return errors.Errorf("Failed authenticate identity for auth "+
			"confirmation of %s", sr.GetPartner())
	}

	// fixme: channel can get into a bricked state if the first save occurs and
	// the second does not
	p := e2e.GetDefaultSessionParams()
	if err := m.storage.E2e().AddPartner(sr.GetPartner(),
		partnerPubKey, sr.GetMyPrivKey(), p, p); err != nil {
		return errors.Errorf("Failed to create channel with partner (%s) "+
			"after confirmation: %+v",
			sr.GetPartner(), err)
	}

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
		Facts:          make([]contact.Fact, 0),
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
