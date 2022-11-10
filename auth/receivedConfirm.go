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

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v5/auth/store"
	"gitlab.com/elixxir/client/v5/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v5/cmix/message"
	"gitlab.com/elixxir/client/v5/cmix/rounds"
	"gitlab.com/elixxir/crypto/contact"
	cAuth "gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/elixxir/primitives/format"
)

type receivedConfirmService struct {
	s *state
	*store.SentRequest
	notificationsService message.Service
}

func (rcs *receivedConfirmService) Process(msg format.Message,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {

	authState := rcs.s

	//parse the confirm
	baseFmt, partnerPubKey, err := handleBaseFormat(msg, authState.e2e.GetGroup())
	if err != nil {
		em := fmt.Sprintf("Failed to handle auth confirm: %s", err)
		jww.WARN.Print(em)
		authState.event.Report(10, "Auth", "ConfirmError", em)
		return
	}

	jww.TRACE.Printf("processing confirm: \n\t MYHISTORICALPUBKEY: %s\n\t"+
		"MYPUBKEY: %s\n\t PARTNERPUBKEY: %s \n\t "+
		"ECRPAYLOAD: %s \n\t MAC: %s",
		authState.e2e.GetHistoricalDHPubkey().TextVerbose(16, 0),
		rcs.SentRequest.GetMyPubKey().TextVerbose(16, 0),
		partnerPubKey.TextVerbose(16, 0),
		base64.StdEncoding.EncodeToString(baseFmt.data),
		base64.StdEncoding.EncodeToString(msg.GetMac()))

	// decrypt the payload
	success, payload := cAuth.Decrypt(rcs.GetMyPrivKey(), partnerPubKey,
		baseFmt.GetEcrPayload(), msg.GetMac(), authState.e2e.GetGroup())

	if !success {
		em := fmt.Sprintf("Received auth confirmation " +
			"failed its mac check")
		jww.WARN.Print(em)
		authState.event.Report(10, "Auth", "ConfirmError", em)
		return
	}

	// parse the data
	ecrFmt, err := unmarshalEcrFormat(payload)
	if err != nil {
		em := fmt.Sprintf("Failed to unmarshal auth confirmation's "+
			"encrypted payload: %s", err)
		jww.WARN.Print(em)
		authState.event.Report(10, "Auth", "ConfirmError", em)
		return
	}

	partnerSIDHPubKey, err := ecrFmt.GetSidhPubKey()
	if err != nil {
		em := fmt.Sprintf("Could not get auth conf SIDH Pubkey: %s",
			err)
		jww.WARN.Print(em)
		authState.event.Report(10, "Auth", "ConfirmError", em)
		return
	}

	jww.TRACE.Printf("handleConfirm PARTNERSIDHPUBKEY: %v",
		partnerSIDHPubKey)

	// check the ownership proof, this verifies the respondent owns the
	// initial identity
	if !cAuth.VerifyOwnershipProof(rcs.SentRequest.GetMyPrivKey(),
		rcs.GetPartnerHistoricalPubKey(),
		authState.e2e.GetGroup(), ecrFmt.GetOwnership()) {
		jww.WARN.Printf("Failed authenticate identity for auth "+
			"confirmation of %s", rcs.GetPartner())
		return
	}

	// add the partner
	p := authState.sessionParams
	_, err = authState.e2e.AddPartner(rcs.GetPartner(), partnerPubKey,
		rcs.GetMyPrivKey(), partnerSIDHPubKey, rcs.GetMySIDHPrivKey(), p, p)
	if err != nil {
		jww.WARN.Printf("Failed to create channel with partner %s and "+
			"%s : %+v", rcs.GetPartner(), receptionID.Source, err)
	}

	if rcs.s.backupTrigger != nil {
		rcs.s.backupTrigger("received confirmation from request")
	}

	// remove the service used for notifications of the confirm
	authState.net.DeleteService(receptionID.Source, rcs.notificationsService, nil)

	// callbacks
	c := contact.Contact{
		ID:             rcs.GetPartner().DeepCopy(),
		DhPubKey:       partnerPubKey.DeepCopy(),
		OwnershipProof: ecrFmt.GetOwnership(),
		Facts:          make([]fact.Fact, 0),
	}

	if cb := authState.partnerCallbacks.getPartnerCallback(c.ID); cb != nil {
		cb.Confirm(c, receptionID, round)
	} else {
		authState.callbacks.Confirm(c, receptionID, round)
	}
}

func (rcs *receivedConfirmService) String() string {
	return fmt.Sprintf("authConfirm(%s, %s, %s)",
		rcs.s.e2e.GetReceptionID(), rcs.GetPartner(),
		rcs.GetFingerprint())
}
