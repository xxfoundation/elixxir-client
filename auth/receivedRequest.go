////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package auth

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/auth/store"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/e2e/ratchet"
	"gitlab.com/elixxir/crypto/contact"
	cAuth "gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
)

const dummyErr = "dummy error so we dont delete the request"

type receivedRequestService struct {
	s     *state
	reset bool
}

func (rrs *receivedRequestService) Process(message format.Message,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	authState := rrs.s

	// check if the timestamp is before the id was created and therefore
	// should be ignored
	tid, err := authState.net.GetIdentity(receptionID.Source)
	if err != nil {
		jww.ERROR.Printf("received a request on %s which does not exist, "+
			"this should not be possible: %+v", receptionID.Source.String(), err)
		return
	}
	if tid.Creation.After(round.GetEndTimestamp()) {
		jww.INFO.Printf("received a request on %s which was sent before "+
			"creation of the identity, dropping because it is likely old "+
			"(before a reset from backup", receptionID.Source.String())
		return
	}

	//decode the outer format
	baseFmt, partnerPubKey, err := handleBaseFormat(
		message, authState.e2e.GetGroup())
	if err != nil {
		jww.WARN.Printf("Failed to handle auth request: %s", err)
		return
	}

	jww.INFO.Printf("partnerPubKey: %v", partnerPubKey.TextVerbose(16, 0))

	jww.TRACE.Printf("processing requests: \n\t MYHISTORICALPUBKEY: %s "+
		"\n\t PARTNERPUBKEY: %s \n\t ECRPAYLOAD: %s \n\t MAC: %s",
		authState.e2e.GetHistoricalDHPubkey().TextVerbose(16, 0),
		partnerPubKey.TextVerbose(16, 0),
		base64.StdEncoding.EncodeToString(baseFmt.data),
		base64.StdEncoding.EncodeToString(message.GetMac()))

	//Attempt to decrypt the payload
	success, payload := cAuth.Decrypt(authState.e2e.GetHistoricalDHPrivkey(),
		partnerPubKey, baseFmt.GetEcrPayload(), message.GetMac(),
		authState.e2e.GetGroup())

	if !success {
		jww.WARN.Printf("Received auth request of %s failed its mac "+
			"check", receptionID.Source)
		return
	}

	//extract data from the decrypted payload
	partnerID, partnerSIDHPubKey, facts, ownershipProof, err :=
		processDecryptedMessage(payload)
	if err != nil {
		jww.WARN.Printf("Failed to decode the auth request: %+v", err)
		return
	}
	jww.INFO.Printf("\t PARTNERSIDHPUBKEY: %v", partnerSIDHPubKey)

	//create the contact, note that no facts are sent in the payload
	c := contact.Contact{
		ID:             partnerID.DeepCopy(),
		DhPubKey:       partnerPubKey.DeepCopy(),
		OwnershipProof: copySlice(ownershipProof),
		Facts:          facts,
	}

	fp := cAuth.CreateNegotiationFingerprint(partnerPubKey,
		partnerSIDHPubKey)
	em := fmt.Sprintf("Received AuthRequest from %s,"+
		" msgDigest: %s, FP: %s", partnerID,
		format.DigestContents(message.GetContents()),
		base64.StdEncoding.EncodeToString(fp))
	jww.INFO.Print(em)
	authState.event.Report(1, "Auth", "RequestReceived", em)

	// check the uniqueness of the request. Requests can be duplicated, so we
	// must verify this is is not a duplicate, and drop if it is
	newFP, position := authState.store.CheckIfNegotiationIsNew(partnerID, fp)

	if !newFP {
		// if its the newest, resend the confirm
		if position == 0 {
			jww.INFO.Printf("Not new request received from %s to %s "+
				"with fp %s at position %d, resending confirm", partnerID,
				receptionID.Source, base64.StdEncoding.EncodeToString(fp),
				position)

			// check if we already accepted, if we did, resend the confirm if
			// we can load it
			if _, err = authState.e2e.GetPartner(partnerID); err != nil {
				//attempt to load the confirm, if we can, resend it
				confirmPayload, mac, keyfp, err :=
					authState.store.LoadConfirmation(partnerID)
				if err != nil {
					jww.ERROR.Printf("Could not reconfirm a duplicate "+
						"request of an accepted confirm from %s to %s because "+
						"the confirm could not be loaded: %+v", partnerID,
						receptionID.Source, err)
				}
				// resend the confirm. It sends as a critical message, so errors
				// do not need to be handled
				_, _ = sendAuthConfirm(authState.net, partnerID, keyfp,
					confirmPayload, mac, authState.event, authState.params.ResetConfirmTag)
			} else if authState.params.ReplayRequests {
				//if we did not already accept, auto replay the request
				if rrs.reset {
					if cb := authState.partnerCallbacks.getPartnerCallback(c.ID); cb != nil {
						cb.Reset(c, receptionID, round)
					} else {
						authState.callbacks.Reset(c, receptionID, round)
					}
				} else {
					if cb := authState.partnerCallbacks.getPartnerCallback(c.ID); cb != nil {
						cb.Request(c, receptionID, round)
					} else {
						authState.callbacks.Request(c, receptionID, round)
					}
				}
			}
			//if not confirm, and params.replay requests is true, we need to replay
		} else {
			jww.INFO.Printf("Not new request received from %s to %s "+
				"with fp %s at position %d, dropping", partnerID,
				receptionID.Source, base64.StdEncoding.EncodeToString(fp),
				position)
		}
		return
	}

	// if we are a reset, check if we have a relationship. If we do not,
	// this is an invalid reset and we need to treat it like a normal
	// new request
	reset := false
	if rrs.reset {
		jww.INFO.Printf("AuthRequest ResetSession from %s,"+
			" msgDigest: %s, FP: %s", partnerID,
			format.DigestContents(message.GetContents()),
			base64.StdEncoding.EncodeToString(fp))
		// delete only deletes if the partner is present, so we can just call delete
		// instead of checking if it exists and then calling delete, and check the
		// error to see if it did or didnt exist
		// Note: due to the newFP handling above, this can ONLY run in the event of
		// a reset or when the partner doesnt exist, so it is safe
		if err = authState.e2e.DeletePartner(partnerID); err != nil {
			if !strings.Contains(err.Error(), ratchet.NoPartnerErrorStr) {
				jww.FATAL.Panicf("Failed to do actual partner deletion: %+v", err)
			}
		} else {
			reset = true
			_ = authState.store.DeleteConfirmation(partnerID)
			_ = authState.store.DeleteSentRequest(partnerID)
		}
	}

	// if a new, unique request is received when one already exists, delete the
	// old one and process the new one
	// this works because message pickup is generally time-sequential.
	if err = authState.store.DeleteReceivedRequest(partnerID); err != nil {
		if !strings.Contains(err.Error(), store.NoRequestFound) {
			jww.FATAL.Panicf("Failed to delete old received request: %+v",
				err)
		}
	}

	// if a sent request already exists, that means we requested at the same
	// time they did. We need to auto-confirm if we are randomly selected
	// (the one with the smaller id,when looked at as long unsigned integer,
	// is selected)
	// (SIDH keys have polarity, so both sent keys cannot be used together)
	autoConfirm := false
	bail := false
	err = authState.store.HandleSentRequest(partnerID,
		func(request *store.SentRequest) error {

			//if this code is running, then we know we sent a request and can
			//auto accept
			//This runner will auto delete the sent request if successful

			//verify ownership proof
			if !cAuth.VerifyOwnershipProof(authState.e2e.GetHistoricalDHPrivkey(),
				partnerPubKey, authState.e2e.GetGroup(), ownershipProof) {
				jww.WARN.Printf("Invalid ownership proof from %s to %s "+
					"received, discarding msgDigest: %s, fp: %s",
					partnerID, receptionID.Source,
					format.DigestContents(message.GetContents()),
					base64.StdEncoding.EncodeToString(fp))
			}

			if !iShouldResend(partnerID, receptionID.Source) {
				// return an error so the store layer does not delete the request
				// because the other side will confirm it
				bail = true
				return errors.Errorf(dummyErr)
			}

			jww.INFO.Printf("Received AuthRequest from %s to %s,"+
				" msgDigest: %s, fp: %s which has been requested, auto-confirming",
				partnerID, receptionID.Source,
				format.DigestContents(message.GetContents()),
				base64.StdEncoding.EncodeToString(fp))
			return nil
		})
	if bail {
		jww.INFO.Printf("Received AuthRequest from %s to %s,"+
			" msgDigest: %s, fp: %s which has been requested, not auto-confirming, "+
			" is other's responsibility",
			partnerID, receptionID.Source,
			format.DigestContents(message.GetContents()),
			base64.StdEncoding.EncodeToString(fp))
		return
	}
	//set the autoconfirm
	autoConfirm = err == nil

	// warning: the client will never be notified of the channel creation if a
	// crash occurs after the store but before the conclusion of the callback
	//create the auth storage
	if err = authState.store.AddReceived(c, partnerSIDHPubKey, round); err != nil {
		em := fmt.Sprintf("failed to store contact Auth "+
			"Request: %s", err)
		jww.WARN.Print(em)
		authState.event.Report(10, "Auth", "RequestError", em)
		return
	}

	// auto-confirm if we should
	if autoConfirm || reset {
		_, _ = authState.confirm(c, authState.params.getConfirmTag(reset))
		//handle callbacks
		if autoConfirm {
			if cb := authState.partnerCallbacks.getPartnerCallback(c.ID); cb != nil {
				cb.Confirm(c, receptionID, round)
			} else {
				authState.callbacks.Confirm(c, receptionID, round)
			}
		} else if reset {
			if cb := authState.partnerCallbacks.getPartnerCallback(c.ID); cb != nil {
				cb.Reset(c, receptionID, round)
			} else {
				authState.callbacks.Reset(c, receptionID, round)
			}
		}
	} else {
		if cb := authState.partnerCallbacks.getPartnerCallback(c.ID); cb != nil {
			cb.Request(c, receptionID, round)
		} else {
			authState.callbacks.Request(c, receptionID, round)
		}
	}
}

func (rrs *receivedRequestService) String() string {
	return fmt.Sprintf("authRequest(%s)",
		rrs.s.e2e.GetReceptionID())
}

func processDecryptedMessage(b []byte) (*id.ID, *sidh.PublicKey, fact.FactList,
	[]byte, error) {
	//decode the ecr format
	ecrFmt, err := unmarshalLegacySIDHEcrFormat(b)
	if err != nil {
		return nil, nil, nil, nil, errors.WithMessage(err, "Failed to "+
			"unmarshal auth request's encrypted payload")
	}

	partnerSIDHPubKey, err := ecrFmt.GetSidhPubKey()
	if err != nil {
		return nil, nil, nil, nil, errors.WithMessage(err, "Could not "+
			"unmarshal partner SIDH Pubkey")
	}

	//decode the request format
	requestFmt, err := newRequestFormat(ecrFmt.GetPayload())
	if err != nil {
		return nil, nil, nil, nil, errors.WithMessage(err, "Failed to "+
			"unmarshal auth request's internal payload")
	}

	partnerID, err := requestFmt.GetID()
	if err != nil {
		return nil, nil, nil, nil, errors.WithMessage(err, "Failed to "+
			"unmarshal auth request's sender ID")
	}

	facts, _, err := fact.UnstringifyFactList(
		string(requestFmt.msgPayload))
	if err != nil {
		return nil, nil, nil, nil, errors.WithMessage(err, "Failed to "+
			"unmarshal auth request's facts")
	}

	return partnerID, partnerSIDHPubKey, facts, ecrFmt.GetOwnership(), nil
}

func iShouldResend(partner, me *id.ID) bool {
	myBytes := me.Bytes()
	theirBytes := partner.Bytes()
	i := 0
	for ; myBytes[i] == theirBytes[i] && i < len(myBytes); i++ {
	}
	return myBytes[i] < theirBytes[i]
}

func copySlice(s []byte) []byte {
	c := make([]byte, len(s))
	copy(c, s)
	return c
}
