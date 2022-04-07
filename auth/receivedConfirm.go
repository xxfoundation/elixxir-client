package auth

import (
	"encoding/base64"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/auth/store"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/cmix/historical"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/crypto/contact"
	cAuth "gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/elixxir/primitives/format"
)

type receivedConfirmService struct {
	s *State
	*store.SentRequest
}

func (rcs *receivedConfirmService) Process(msg format.Message,
	receptionID receptionID.EphemeralIdentity, round historical.Round) {

	state := rcs.s

	// lookup keypair
	kp, exist := state.getRegisteredIDs(receptionID.Source)

	if !exist {
		jww.ERROR.Printf("received a confirm for %s, " +
			"but they are not registered with auth, cannot process")
		return
	}

	//parse the confirm
	baseFmt, partnerPubKey, err := handleBaseFormat(msg, state.e2e.GetGroup())
	if err != nil {
		em := fmt.Sprintf("Failed to handle auth confirm: %s", err)
		jww.WARN.Print(em)
		state.event.Report(10, "Auth", "ConfirmError", em)
		return
	}

	jww.TRACE.Printf("processing confirm: \n\t MYPUBKEY: %s "+
		"\n\t PARTNERPUBKEY: %s \n\t ECRPAYLOAD: %s \n\t MAC: %s",
		kp.pubkey.TextVerbose(16, 0),
		partnerPubKey.TextVerbose(16, 0),
		base64.StdEncoding.EncodeToString(baseFmt.data),
		base64.StdEncoding.EncodeToString(msg.GetMac()))

	// decrypt the payload
	success, payload := cAuth.Decrypt(rcs.GetMyPrivKey(), partnerPubKey,
		baseFmt.GetEcrPayload(), msg.GetMac(), state.e2e.GetGroup())

	if !success {
		em := fmt.Sprintf("Received auth confirmation failed its mac " +
			"check")
		jww.WARN.Print(em)
		state.event.Report(10, "Auth", "ConfirmError", em)
		return
	}

	// parse the data
	ecrFmt, err := unmarshalEcrFormat(payload)
	if err != nil {
		em := fmt.Sprintf("Failed to unmarshal auth confirmation's "+
			"encrypted payload: %s", err)
		jww.WARN.Print(em)
		state.event.Report(10, "Auth", "ConfirmError", em)
		return
	}

	partnerSIDHPubKey, err := ecrFmt.GetSidhPubKey()
	if err != nil {
		em := fmt.Sprintf("Could not get auth conf SIDH Pubkey: %s",
			err)
		jww.WARN.Print(em)
		state.event.Report(10, "Auth", "ConfirmError", em)
		return
	}

	jww.TRACE.Printf("handleConfirm PARTNERSIDHPUBKEY: %v",
		partnerSIDHPubKey)

	// check the ownership proof, this verifies the respondent owns the
	// initial identity
	if !cAuth.VerifyOwnershipProof(kp.privkey, rcs.GetPartnerHistoricalPubKey(),
		state.e2e.GetGroup(), ecrFmt.GetOwnership()) {
		jww.WARN.Printf("Failed authenticate identity for auth "+
			"confirmation of %s", rcs.GetPartner())
		return
	}

	// add the partner
	p := session.GetDefaultParams()
	_, err = state.e2e.AddPartner(receptionID.Source, rcs.GetPartner(), partnerPubKey,
		rcs.GetMyPrivKey(), partnerSIDHPubKey, rcs.GetMySIDHPrivKey(), p, p)
	if err != nil {
		jww.WARN.Printf("Failed to create channel with partner %s and "+
			"%s : %+v", rcs.GetPartner(), receptionID.Source, err)
	}

	//todo: trigger backup

	// remove the service used for notifications of the confirm
	confirmFP := rcs.GetFingerprint()
	state.net.DeleteService(receptionID.Source, message.Service{
		Identifier: confirmFP[:],
		Tag:        catalog.Confirm,
	}, nil)

	// callbacks
	c := contact.Contact{
		ID:             rcs.GetPartner().DeepCopy(),
		DhPubKey:       partnerPubKey.DeepCopy(),
		OwnershipProof: ecrFmt.GetOwnership(),
		Facts:          make([]fact.Fact, 0),
	}
	if cb, exists := state.confirmCallbacks.Get(receptionID.Source); exists {
		cb(c, receptionID, round)
	}
}
