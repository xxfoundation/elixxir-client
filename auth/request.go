///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package auth

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/contact"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/interfaces/utility"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/auth"
	"gitlab.com/elixxir/client/storage/e2e"
	ds "gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/crypto/diffieHellman"
	cAuth "gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/states"
	"io"
	"strings"
	"time"
)

const terminator = ";"

func RequestAuth(partner, me contact.Contact, message string, rng io.Reader,
	storage *storage.Session, net interfaces.NetworkManager) error {
	/*edge checks generation*/

	// check that messages can be sent over the network
	if !net.GetHealthTracker().IsHealthy() {
		return errors.New("Cannot create authenticated message " +
			"when the network is not healthy")
	}

	// check that an authenticated channel does not already exists
	if _, err := storage.E2e().GetPartner(partner.ID); err == nil ||
		!strings.Contains(err.Error(), e2e.NoPartnerErrorStr) {
		return errors.Errorf("Authenticated channel already " +
			"established with partner")
	}

	// check that the request is being sent from the proper ID
	if !me.ID.Cmp(storage.GetUser().ReceptionID) {
		return errors.Errorf("Authenticated channel request " +
			"can only be sent from user's identity")
	}

	// check that the message is properly formed
	if strings.Contains(message, terminator) {
		return errors.Errorf("Message cannot contain '%s'", terminator)
	}

	//lookup if an ongoing request is occurring
	rqType, _, _, err := storage.Auth().GetRequest(partner.ID)
	if err != nil && strings.Contains(err.Error(), auth.NoRequest) {
		err = nil
	}
	if err != nil {
		if rqType == auth.Receive {
			return errors.WithMessage(err,
				"Cannot send a request after "+
					"receiving a request")
		} else if rqType == auth.Sent {
			return errors.WithMessage(err,
				"Cannot send a request after "+
					"already sending one")
		}
	}

	grp := storage.E2e().GetGroup()

	/*generate embedded message structures and check payload*/
	cmixMsg := format.NewMessage(storage.Cmix().GetGroup().GetP().ByteLen())
	baseFmt := newBaseFormat(cmixMsg.ContentsSize(), grp.GetP().ByteLen())
	ecrFmt := newEcrFormat(baseFmt.GetEcrPayloadLen())
	requestFmt, err := newRequestFormat(ecrFmt)
	if err != nil {
		return errors.Errorf("failed to make request format: %+v", err)
	}

	//check the payload fits
	facts := me.Facts.Stringify()
	msgPayload := facts + message + terminator
	msgPayloadBytes := []byte(msgPayload)

	if len(msgPayloadBytes) > requestFmt.MsgPayloadLen() {
		return errors.Errorf("Combined message longer than space "+
			"available in payload; available: %v, length: %v",
			requestFmt.MsgPayloadLen(), len(msgPayloadBytes))
	}

	/*cryptographic generation*/
	//generate salt
	salt := make([]byte, saltSize)
	_, err = rng.Read(salt)
	if err != nil {
		return errors.Wrap(err, "Failed to generate salt")
	}

	//generate ownership proof
	ownership := cAuth.MakeOwnershipProof(storage.E2e().GetDHPrivateKey(),
		partner.DhPubKey, storage.E2e().GetGroup())

	//generate new keypair
	newPrivKey := diffieHellman.GeneratePrivateKey(256, grp, rng)
	newPubKey := diffieHellman.GeneratePublicKey(newPrivKey, grp)

	jww.INFO.Printf("RequestAuth MYPUBKEY: %v", newPubKey.Bytes())
	jww.INFO.Printf("RequestAuth THEIRPUBKEY: %v", partner.DhPubKey.Bytes())

	/*encrypt payload*/
	requestFmt.SetID(storage.GetUser().ReceptionID)
	requestFmt.SetMsgPayload(msgPayloadBytes)
	ecrFmt.SetOwnership(ownership)
	ecrPayload, mac := cAuth.Encrypt(newPrivKey, partner.DhPubKey,
		salt, ecrFmt.data, grp)
	confirmFp := cAuth.MakeOwnershipProofFP(ownership)
	requestfp := cAuth.MakeRequestFingerprint(partner.DhPubKey)

	/*construct message*/
	baseFmt.SetEcrPayload(ecrPayload)
	baseFmt.SetSalt(salt)
	baseFmt.SetPubKey(newPubKey)

	cmixMsg.SetKeyFP(requestfp)
	cmixMsg.SetMac(mac)
	cmixMsg.SetContents(baseFmt.Marshal())
	cmixMsg.SetRecipientID(partner.ID)
	jww.INFO.Printf("PARTNER ID: %s", partner.ID)

	/*store state*/
	//fixme: channel is bricked if the first store succedes but the second fails
	//store the in progress auth
	err = storage.Auth().AddSent(partner.ID, partner.DhPubKey, newPrivKey,
		newPrivKey, confirmFp)
	if err != nil {
		return errors.Errorf("Failed to store auth request: %s", err)
	}

	//store the message as a critical message so it will always be sent
	storage.GetCriticalRawMessages().AddProcessing(cmixMsg)

	//jww.INFO.Printf("CMIX MESSAGE 1: %s, %v, %v, %v", cmixMsg.GetRecipientID(),
	//	cmixMsg.GetKeyFP(), cmixMsg.GetMac(), cmixMsg.GetContents())
	jww.INFO.Printf("CMIX MESSAGE FP: %s, %v", cmixMsg.GetRecipientID(),
		cmixMsg.GetKeyFP())

	/*send message*/
	round, err := net.SendCMIX(cmixMsg, params.GetDefaultCMIX())
	if err != nil {
		// if the send fails just set it to failed, it will but automatically
		// retried
		jww.ERROR.Printf("auth request failed to transmit, will be "+
			"handled on reconnect: %+v", err)
		storage.GetCriticalRawMessages().Failed(cmixMsg)
	}

	/*check message delivery*/
	sendResults := make(chan ds.EventReturn, 1)
	roundEvents := net.GetInstance().GetRoundEvents()

	roundEvents.AddRoundEventChan(round, sendResults, 1*time.Minute,
		states.COMPLETED, states.FAILED)

	success, _, _ := utility.TrackResults(sendResults, 1)
	if !success {
		jww.ERROR.Printf("auth request failed to transmit, will be " +
			"handled on reconnect")
		storage.GetCriticalRawMessages().Failed(cmixMsg)
	} else {
		storage.GetCriticalRawMessages().Succeeded(cmixMsg)
	}

	return nil
}
