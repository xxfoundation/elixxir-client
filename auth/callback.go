///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package auth

import (
	"fmt"
	"strings"

	"github.com/cloudflare/circl/dh/sidh"
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
	"gitlab.com/xx_network/primitives/id"
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
	cmixMsg, err := format.Unmarshal(msg.Payload)
	if err != nil {
		jww.WARN.Printf("Invalid message when unmarshalling: %s",
			err.Error())
		// Ignore this message
		return
	}
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
	baseFmt, partnerPubKey, err := handleBaseFormat(
		cmixMsg, grp)
	if err != nil {
		jww.WARN.Printf("Failed to handle auth request: %s", err)
		return
	}

	myPubKey := diffieHellman.GeneratePublicKey(myHistoricalPrivKey, grp)

	jww.TRACE.Printf("handleRequest MYPUBKEY: %v", myPubKey.Bytes())
	jww.TRACE.Printf("handleRequest PARTNERPUBKEY: %v", partnerPubKey.Bytes())

	//decrypt the message
	jww.TRACE.Printf("handleRequest ECRPAYLOAD: %v", baseFmt.GetEcrPayload())
	jww.TRACE.Printf("handleRequest MAC: %v", cmixMsg.GetMac())

	ecrPayload := baseFmt.GetEcrPayload()
	success, payload := cAuth.Decrypt(myHistoricalPrivKey,
		partnerPubKey, ecrPayload,
		cmixMsg.GetMac(), grp)

	if !success {
		jww.WARN.Printf("Attempting to decrypt old request packet...")
		ecrPayload = append(ecrPayload, baseFmt.GetVersion())
		success, payload = cAuth.Decrypt(myHistoricalPrivKey,
			partnerPubKey, ecrPayload,
			cmixMsg.GetMac(), grp)
	}

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
	partnerSIDHPubKey, err := ecrFmt.GetSidhPubKey()
	if err != nil {
		jww.WARN.Printf("Could not unmarshal partner SIDH Pubkey: %s",
			err)
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
	// Check if this is a reset, which are valid as of version 1
	// Resets happen when our fingerprint is new AND we are
	// the latest fingerprint to be added to the list and we already have
	// a negotiation or authenticated channel in progress
	fp := cAuth.CreateNegotiationFingerprint(partnerPubKey,
		partnerSIDHPubKey)
	newFP, latest := m.storage.Auth().AddIfNew(partnerID, fp)
	resetSession := false
	autoConfirm := false
	if baseFmt.GetVersion() >= 1 && newFP && latest {
		// If we had an existing session and it's new, then yes, we
		// want to reset
		if _, err := m.storage.E2e().GetPartner(partnerID); err == nil {
			jww.INFO.Printf("Resetting session for %s", partnerID)
			resetSession = true
			// Most likely, we got 2 reset sessions at once, so this
			// is a non-fatal error but we will record a warning
			// just in case.
			err = m.storage.E2e().DeletePartner(partnerID)
			if err != nil {
				jww.WARN.Printf("Unable to delete channel: %+v",
					err)
			}
			// Also delete any existing request, sent or received
			m.storage.Auth().Delete(partnerID)
		}
		// If we had an existing negotiation open, then it depends

		// If we've only received, then user has not confirmed, treat as
		// a non-duplicate request, so delete the old one (to cause new
		// callback to be called)
		rType, _, _, err := m.storage.Auth().GetRequest(partnerID)
		if err != nil && rType == auth.Receive {
			m.storage.Auth().Delete(partnerID)
		}

		// If we've already Sent and are now receiving,
		// then we attempt auto-confirm as below
		// This poses a potential problem if it is truly a session
		// reset by the other user, because we may not actually
		// autoconfirm based on our public key compared to theirs.
		// This could result in a permanently broken association, as
		// the other side has attempted to reset it's session and
		// can no longer detect a sent request collision, so this side
		// cannot ever successfully resend.
		// We prevent this by stopping session resets if they
		// are called when the other side is in the "Sent" state.
		// If the other side is in the "received" state we also block,
		// but we could autoconfirm.
		// Note that you can still get into this state by one side
		// deleting requests. In that case, both sides need to clear
		// out all requests and retry negotiation from scratch.
		// NOTE: This protocol part could use an overhaul/second look,
		//       there's got to be a way to do this with far less state
		//       but this is the spec so we're sticking with it for now.

		// If not an existing request, we do nothing.
	} else {
		jww.WARN.Printf("Version: %d, newFP: %v, latest: %v", baseFmt.GetVersion(),
			newFP, latest)
	}

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
		rType, _, c, err := m.storage.Auth().GetRequest(partnerID)
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
				// if the caller of the API wants requests replayed,
				// replay the duplicate request
				if m.replayRequests {
					cbList := m.requestCallbacks.Get(c.ID)
					for _, cb := range cbList {
						rcb := cb.(interfaces.RequestCallback)
						go rcb(c)
					}
				}
				return
			// if we sent a request, then automatically confirm
			// then exit, nothing else needed
			case auth.Sent:
				jww.INFO.Printf("Received AuthRequest from %s,"+
					" msgDigest: %s which has been requested, auto-confirming",
					partnerID, cmixMsg.Digest())

				// Verify this request is legit
				ownership := ecrFmt.GetOwnership()
				if !cAuth.VerifyOwnershipProof(
					myHistoricalPrivKey, partnerPubKey, grp,
					ownership) {
					jww.WARN.Printf("Invalid ownership proof from %s received, discarding msdDigest: %s",
						partnerID, cmixMsg.Digest())
				}

				// Check if I need to resend by comparing the
				// IDs
				myBytes := m.storage.GetUser().ReceptionID.Bytes()
				theirBytes := partnerID.Bytes()
				for i := 0; i < len(myBytes); i++ {
					if myBytes[i] > theirBytes[i] {
						// OK, this side is dropping
						// the request
						// Do we need to delete
						// something here?
						// No, because we will
						// now wait to receive
						// confirmation.
						return
					} else if myBytes[i] < theirBytes[i] {
						break
					}
				}

				// If I do, delete my request on disk
				m.storage.Auth().Delete(partnerID)

				// Do the normal, fall out of this if block and
				// create the contact, note that we use the data
				// sent in the request and not any data we had
				// already

				autoConfirm = true

			}
		}
	}

	//process the inner payload
	facts, _, err := fact.UnstringifyFactList(
		string(requestFmt.msgPayload))
	if err != nil {
		em := fmt.Sprintf("failed to parse facts and message "+
			"from Auth Request: %s", err)
		jww.WARN.Print(em)
		events.Report(10, "Auth", "RequestError", em)
		return
	}

	//create the contact, note that no facts are sent in the payload
	c := contact.Contact{
		ID:             partnerID.DeepCopy(),
		DhPubKey:       partnerPubKey.DeepCopy(),
		OwnershipProof: copySlice(ecrFmt.ownership),
		Facts:          facts,
	}

	// fixme: the client will never be notified of the channel creation if a
	// crash occurs after the store but before the conclusion of the callback
	//create the auth storage
	if err = m.storage.Auth().AddReceived(c, partnerSIDHPubKey); err != nil {
		em := fmt.Sprintf("failed to store contact Auth "+
			"Request: %s", err)
		jww.WARN.Print(em)
		events.Report(10, "Auth", "RequestError", em)
		return
	}

	// We autoconfirm anytime we had already sent a request OR we are
	// resetting an existing session
	var rndNum id.Round
	if autoConfirm || resetSession {
		// Call ConfirmRequestAuth to send confirmation
		rndNum, err = m.confirmRequestAuth(c, true)
		if err != nil {
			jww.ERROR.Printf("Could not ConfirmRequestAuth: %+v",
				err)
			return
		}

		if autoConfirm {
			jww.INFO.Printf("ConfirmRequestAuth to %s on round %d",
				partnerID, rndNum)
			cbList := m.confirmCallbacks.Get(c.ID)
			for _, cb := range cbList {
				ccb := cb.(interfaces.ConfirmCallback)
				go ccb(c)
			}
		}
		if resetSession {
			jww.INFO.Printf("Reset Auth %s on round %d",
				partnerID, rndNum)
			cbList := m.resetCallbacks.Get(c.ID)
			for _, cb := range cbList {
				ccb := cb.(interfaces.ResetNotificationCallback)
				go ccb(c)
			}
		}
	} else {
		//  fixme: if a crash occurs before or during the calls, the notification
		//  will never be sent.
		cbList := m.requestCallbacks.Get(c.ID)
		for _, cb := range cbList {
			rcb := cb.(interfaces.RequestCallback)
			go rcb(c)
		}
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
	baseFmt, partnerPubKey, err := handleBaseFormat(
		cmixMsg, grp)
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
	jww.TRACE.Printf("handleConfirm ECRPAYLOAD: %v", baseFmt.GetEcrPayload())
	jww.TRACE.Printf("handleConfirm MAC: %v", cmixMsg.GetMac())
	success, payload := cAuth.Decrypt(sr.GetMyPrivKey(),
		partnerPubKey, baseFmt.GetEcrPayload(),
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

	partnerSIDHPubKey, err := ecrFmt.GetSidhPubKey()
	if err != nil {
		em := fmt.Sprintf("Could not get auth conf SIDH Pubkey: %s",
			err)
		jww.WARN.Print(em)
		events.Report(10, "Auth", "ConfirmError", em)
		m.storage.Auth().Done(sr.GetPartner())
		return
	}
	jww.TRACE.Printf("handleConfirm PARTNERSIDHPUBKEY: %v",
		partnerSIDHPubKey)

	// finalize the confirmation
	if err := m.doConfirm(sr, grp, partnerPubKey, sr.GetMyPrivKey(),
		sr.GetPartnerHistoricalPubKey(),
		ecrFmt.GetOwnership(),
		partnerSIDHPubKey); err != nil {
		em := fmt.Sprintf("Confirmation failed: %s", err)
		jww.WARN.Print(em)
		events.Report(10, "Auth", "ConfirmError", em)
		m.storage.Auth().Done(sr.GetPartner())
		return
	}
}

func (m *Manager) doConfirm(sr *auth.SentRequest, grp *cyclic.Group,
	partnerPubKey, myPrivateKeyOwnershipProof, partnerPubKeyOwnershipProof *cyclic.Int,
	ownershipProof []byte, partnerSIDHPubKey *sidh.PublicKey) error {
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
		partnerPubKey, sr.GetMyPrivKey(), partnerSIDHPubKey,
		sr.GetMySIDHPrivKey(), p, p); err != nil {
		return errors.Errorf("Failed to create channel with partner (%s) "+
			"after confirmation: %+v",
			sr.GetPartner(), err)
	}

	m.backupTrigger("received confirmation from request")

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

	addPreimages(sr.GetPartner(), m.storage)

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
	if err != nil && baseFmt == nil {
		return baseFormat{}, nil, errors.WithMessage(err, "Failed to"+
			" unmarshal auth")
	}

	if !grp.BytesInside(baseFmt.pubkey) {
		return baseFormat{}, nil, errors.WithMessage(err, "Received "+
			"auth confirmation public key is not in the e2e cyclic group")
	}
	partnerPubKey := grp.NewIntFromBytes(baseFmt.pubkey)

	return *baseFmt, partnerPubKey, nil
}
