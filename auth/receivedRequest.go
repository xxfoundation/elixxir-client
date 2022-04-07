package auth

import (
	"encoding/base64"
	"fmt"
	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/auth/store"
	"gitlab.com/elixxir/client/cmix/historical"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/e2e/ratchet"
	"gitlab.com/elixxir/crypto/contact"
	cAuth "gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"strings"
)

const dummyerr = "dummy error so we dont delete the request"

type receivedRequestService struct {
	s *State
}

func (rrs *receivedRequestService) Process(message format.Message,
	receptionID receptionID.EphemeralIdentity, round historical.Round) {

	state := rrs.s

	//lookup the keypair
	kp, exist := state.getRegisteredIDs(receptionID.Source)

	if !exist {
		jww.ERROR.Printf("received a confirm for %s, " +
			"but they are not registered with auth, cannot process")
		return
	}

	// check if the timestamp is before the id was created and therefore
	// should be ignored
	tid, err := state.net.GetIdentity(receptionID.Source)
	if err != nil {
		jww.ERROR.Printf("received a request on %s which does not exist, " +
			"this should not be possible")
		return
	}
	if tid.Creation.After(round.GetEndTimestamp()) {
		jww.INFO.Printf("received a request on %s which was sent before " +
			"creation of the identity, dropping because it is likely old " +
			"(before a reset from backup")
		return
	}

	//decode the outer format
	baseFmt, partnerPubKey, err := handleBaseFormat(
		message, state.e2e.GetGroup())
	if err != nil {
		jww.WARN.Printf("Failed to handle auth request: %s", err)
		return
	}

	jww.TRACE.Printf("processing requests: \n\t MYPUBKEY: %s "+
		"\n\t PARTNERPUBKEY: %s \n\t ECRPAYLOAD: %s \n\t MAC: %s",
		kp.pubkey.TextVerbose(16, 0),
		partnerPubKey.TextVerbose(16, 0),
		base64.StdEncoding.EncodeToString(baseFmt.data),
		base64.StdEncoding.EncodeToString(message.GetMac()))

	//Attempt to decrypt the payload
	success, payload := cAuth.Decrypt(kp.privkey, partnerPubKey,
		baseFmt.GetEcrPayload(), message.GetMac(), state.e2e.GetGroup())

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
	state.event.Report(1, "Auth", "RequestReceived", em)

	// check the uniqueness of the request. Requests can be duplicated, so we
	// must verify this is is not a duplicate, and drop if it is
	newFP, position := state.store.CheckIfNegotiationIsNew(partnerID,
		receptionID.Source, fp)

	if !newFP {
		// if its the newest, resend the confirm
		if position == 0 {
			jww.INFO.Printf("Not new request received from %s to %s "+
				"with fp %s at position %d, resending confirm", partnerID,
				receptionID.Source, base64.StdEncoding.EncodeToString(fp),
				position)

			// check if we already accepted, if we did, resend the confirm if
			// we can load it
			if _, err = state.e2e.GetPartner(partnerID, receptionID.Source); err != nil {
				//attempt to load the confirm, if we can, resend it
				confirmPayload, mac, keyfp, err :=
					state.store.LoadConfirmation(partnerID, receptionID.Source)
				if err != nil {
					jww.ERROR.Printf("Could not reconfirm a duplicate "+
						"request of an accepted confirm from %s to %s because "+
						"the confirm could not be loaded: %+v", partnerID,
						receptionID.Source, err)
				}
				// resend the confirm. It sends as a critical message, so errors
				// do not need to be handled
				_, _ = sendAuthConfirm(state.net, partnerID, receptionID.Source,
					keyfp, confirmPayload, mac, state.event)
			} else if state.params.ReplayRequests {
				//if we did not already accept, auto replay the request
				if cb, exist := state.requestCallbacks.Get(receptionID.Source); exist {
					cb(c, receptionID, round)
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

	reset := false
	// check if we have a relationship, given this is a new request, if we have
	// a relationship, this must be a reset, which which case we delete all
	// state and then continue like a new request
	// delete only deletes if the partner is present, so we can just call delete
	// instead of checking if it exists and then calling delete, and check the
	// error to see if it did or didnt exist
	// Note: due to the newFP handling above, this can ONLY run in the event of
	// a reset or when the partner doesnt exist, so it is safe
	if err = state.e2e.DeletePartner(partnerID, receptionID.Source); err != nil {
		if !strings.Contains(err.Error(), ratchet.NoPartnerErrorStr) {
			jww.FATAL.Panicf("Failed to do actual partner deletion: %+v", err)
		}
	} else {
		reset = true
		_ = state.store.DeleteConfirmation(partnerID, receptionID.Source)
		_ = state.store.DeleteSentRequest(partnerID, receptionID.Source)
	}

	// if a new, unique request is received when one already exists, delete the
	// old one and process the new one
	if err = state.store.DeleteReceivedRequest(partnerID, receptionID.Source); err != nil {
		if !strings.Contains(err.Error(), store.NoRequestFound) {
			jww.FATAL.Panicf("Failed to delete old received request: %+v",
				err)
		}
	}

	// if a sent request already exists, that means we requested at the same
	// time they did. We need to auto-confirm if we are randomly selected
	// (SIDH keys have polarity, so both sent keys cannot be used together)
	autoConfirm := false
	bail := false
	err = state.store.HandleSentRequest(partnerID, receptionID.Source,
		func(request *store.SentRequest) error {

			//if this code is running, then we know we sent a request and can
			//auto accept
			//This runner will auto delete the sent request if successful

			//verify ownership proof
			if !cAuth.VerifyOwnershipProof(kp.pubkey, partnerPubKey,
				state.e2e.GetGroup(), ownershipProof) {
				jww.WARN.Printf("Invalid ownership proof from %s to %s "+
					"received, discarding msdDigest: %s, fp: %s",
					partnerID, receptionID.Source,
					format.DigestContents(message.GetContents()),
					base64.StdEncoding.EncodeToString(fp))
			}

			if !iShouldResend(partnerID, receptionID.Source) {
				// return an error so the store layer does not delete the request
				// because the other side will confirm it
				bail = true
				return errors.Errorf(dummyerr)
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
	if err = state.store.AddReceived(receptionID.Source, c, partnerSIDHPubKey); err != nil {
		em := fmt.Sprintf("failed to store contact Auth "+
			"Request: %s", err)
		jww.WARN.Print(em)
		state.event.Report(10, "Auth", "RequestError", em)
		return
	}

	//autoconfirm if we should
	if autoConfirm || reset {
		_, _ = state.confirmRequestAuth(c, receptionID.Source)
		//handle callbacks
		if autoConfirm {
			if cb, exist := state.confirmCallbacks.Get(receptionID.Source); exist {
				cb(c, receptionID, round)
			}
		} else if reset {
			if cb, exist := state.requestCallbacks.Get(receptionID.Source); exist {
				cb(c, receptionID, round)
			}
		}
	} else {
		//otherwise call callbacks
		if cb, exist := state.requestCallbacks.Get(receptionID.Source); exist {
			cb(c, receptionID, round)
		}
	}
}

func processDecryptedMessage(b []byte) (*id.ID, *sidh.PublicKey, fact.FactList,
	[]byte, error) {
	//decode the ecr format
	ecrFmt, err := unmarshalEcrFormat(b)
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
	requestFmt, err := newRequestFormat(ecrFmt)
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
