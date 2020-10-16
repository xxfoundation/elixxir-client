package auth

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/contact"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/auth"
	"gitlab.com/elixxir/client/storage/e2e"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	cAuth "gitlab.com/elixxir/crypto/e2e/auth"
	jww "github.com/spf13/jwalterweatherman"
	"io"
	"strings"
)

type RequestCallback func(requestor contact.Contact, message string)
type ConfirmCallback func(partner contact.Contact)

func RegisterCallbacks(rcb RequestCallback, ccb ConfirmCallback,
	sw interfaces.Switchboard, storage *storage.Session,
	net interfaces.NetworkManager, rng io.Reader) stoppable.Stoppable {

	rawMessages := make(chan message.Receive, 1000)
	sw.RegisterChannel("Auth", &id.ID{}, message.Raw, rawMessages)

	stop := stoppable.NewSingle("Auth")
	authStore := storage.Auth()
	grp := storage.E2e().GetGroup()

	go func() {
		select {
		case <-stop.Quit():
			return
		case msg := <-rawMessages:
			//lookup the message, check if it is an auth request
			cmixMsg := format.Unmarshal(msg.Payload)
			fp := cmixMsg.GetKeyFP()
			fpType, sr, myHistoricalPrivKey, err := authStore.GetFingerprint(fp)
			if err != nil {
				// if the lookup fails, ignore the message. It is likely
				// garbled or for a different protocol
				break
			}

			//denote that the message is not garbled
			storage.GetGarbledMessages().Remove(cmixMsg)

			switch fpType {
			// if it is general, that means a new request has been received
			case auth.General:
				handleRequest(cmixMsg, myHistoricalPrivKey, grp, storage, rcb,
					ccb, net)
			// if it is specific, that means the original request was sent
			// by this users and a confirmation has been received
			case auth.Specific:
				handleConfirm(cmixMsg, sr, ccb, storage, grp, net)
			}
		}
	}()
	return stop
}

func handleRequest(cmixMsg format.Message, myHistoricalPrivKey *cyclic.Int,
	grp *cyclic.Group, storage *storage.Session, rcb RequestCallback,
	ccb ConfirmCallback, net interfaces.NetworkManager) {
	//decode the outer format
	baseFmt, partnerPubKey, err := handleBaseFormat(cmixMsg, grp)
	if err != nil {
		jww.WARN.Printf("Failed to handle auth request: %s", err)
		return
	}

	//decrypt the message
	success, payload, _ := cAuth.Decrypt(myHistoricalPrivKey,
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
	if _, err := storage.E2e().GetPartner(partnerID); err == nil {
		jww.WARN.Printf("Recieved Auth request for %s, "+
			"channel already exists. Ignoring", partnerID)
		//exit
		return
	} else {
		//check if the relationship already exists,
		rType, sr2, _, err := storage.Auth().GetRequest(partnerID)
		if err != nil && !strings.Contains(err.Error(), auth.NoRequest) {
			// if another error is recieved, print it and exist
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
				if err := doConfirm(sr2, grp, partnerPubKey, ecrFmt.GetOwnership(),
					storage, ccb, net); err != nil {
					jww.WARN.Printf("Confirmation failed: %s", err)
				}
				//exit
				return
			}
		}
	}

	//process the inner payload
	facts, msg, err := contact.UnstringifyFacts(
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

	//create the auth storage
	if err = storage.Auth().AddReceived(c); err != nil {
		jww.WARN.Printf("failed to store contact Auth "+
			"Request: %s", err)
		return
	}

	//call the callback

	go rcb(c, msg)
	return
}

func handleConfirm(cmixMsg format.Message, sr *auth.SentRequest,
	ccb ConfirmCallback, storage *storage.Session, grp *cyclic.Group,
	net interfaces.NetworkManager) {
	// check if relationship already exists
	if m, err := storage.E2e().GetPartner(sr.GetPartner()); m != nil || err == nil {
		jww.WARN.Printf("Cannot confirm auth for %s, channel already "+
			"exists.", sr.GetPartner())
		return
	}

	// extract the message
	baseFmt, partnerPubKey, err := handleBaseFormat(cmixMsg, grp)
	if err != nil {
		jww.WARN.Printf("Failed to handle auth confirm: %s", err)
		return
	}

	// decrypt the payload
	success, payload, _ := cAuth.Decrypt(sr.GetMyPrivKey(),
		partnerPubKey, baseFmt.GetSalt(), baseFmt.GetEcrPayload(),
		cmixMsg.GetMac(), grp)

	if !success {
		jww.WARN.Printf("Recieved auth confirmation failed its mac " +
			"check")
		return
	}

	ecrFmt, err := unmarshalEcrFormat(payload)
	if err != nil {
		jww.WARN.Printf("Failed to unmarshal auth confirmation's "+
			"encrypted payload: %s", err)
		return
	}

	// finalize the confirmation
	if err := doConfirm(sr, grp, partnerPubKey, ecrFmt.GetOwnership(),
		storage, ccb, net); err != nil {
		jww.WARN.Printf("Confirmation failed: %s", err)
		return
	}
}

func doConfirm(sr *auth.SentRequest, grp *cyclic.Group,
	partnerPubKey *cyclic.Int, ownershipProof []byte, storage *storage.Session,
	ccb ConfirmCallback, net interfaces.NetworkManager) error {
	// verify the message came from the intended recipient
	if !cAuth.VerifyOwnershipProof(sr.GetMyPrivKey(),
		sr.GetPartnerHistoricalPubKey(), grp, ownershipProof) {
		return errors.Errorf("Failed authenticate identity for auth "+
			"confirmation of %s", sr.GetPartner())
	}

	// create the relationship
	p := e2e.GetDefaultSessionParams()
	if err := storage.E2e().AddPartner(sr.GetPartner(),
		partnerPubKey, p, p); err != nil {
		return errors.Errorf("Failed to create channel with partner (%s) "+
			"after confirmation: %+v",
			sr.GetPartner(), err)
	}
	net.CheckGarbledMessages()

	// delete the in progress negotiation
	if err := storage.Auth().Delete(sr.GetPartner()); err != nil {
		return errors.Errorf("Failed to delete in progress negotiation "+
			"with partner (%s) after confirmation: %+v",
			sr.GetPartner(), err)
	}

	//notify the end point
	c := contact.Contact{
		ID:             sr.GetPartner().DeepCopy(),
		DhPubKey:       partnerPubKey.DeepCopy(),
		OwnershipProof: copySlice(ownershipProof),
		Facts:          make([]contact.Fact, 0),
	}

	go ccb(c)

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