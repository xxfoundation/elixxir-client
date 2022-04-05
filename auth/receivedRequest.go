package auth

import (
	"encoding/base64"
	"fmt"
	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/network/historical"
	"gitlab.com/elixxir/client/network/identity/receptionID"
	cAuth "gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
)

type receivedRequestService struct {
	m *Manager
}

func (rrs *receivedRequestService) Process(message format.Message,
	receptionID receptionID.EphemeralIdentity, round historical.Round) {

	//decode the outer format
	baseFmt, partnerPubKey, err := handleBaseFormat(
		message, rrs.m.grp)
	if err != nil {
		jww.WARN.Printf("Failed to handle auth request: %s", err)
		return
	}

	//lookup the keypair
	kp := rrs.m.registeredIDs[*receptionID.Source]

	jww.TRACE.Printf("processing requests: \n\t MYPUBKEY: %s "+
		"\n\t PARTNERPUBKEY: %s \n\t ECRPAYLOAD: %s \n\t MAC: %s",
		kp.pubkey.TextVerbose(16, 0),
		partnerPubKey.TextVerbose(16, 0),
		base64.StdEncoding.EncodeToString(baseFmt.data),
		base64.StdEncoding.EncodeToString(message.GetMac()))

	//Attempt to decrypt the payload
	success, payload := cAuth.Decrypt(kp.privkey, partnerPubKey,
		baseFmt.GetEcrPayload(), message.GetMac(), rrs.m.grp)

	if !success {
		jww.WARN.Printf("Received auth request of %s failed its mac "+
			"check", receptionID.Source)
		return
	}

	//extract data from the decrypted payload
	partnerID, partnerSIDHPubKey, facts, err := processDecryptedMessage(payload)
	if err != nil {
		jww.WARN.Printf("Failed to decode the auth request: %+v", err)
		return
	}

	em := fmt.Sprintf("Received AuthRequest from %s,"+
		" msgDigest: %s", partnerID, format.DigestContents(message.GetContents()))
	jww.INFO.Print(em)
	rrs.m.event.Report(1, "Auth", "RequestReceived", em)

	// check the uniqueness of the request. Requests can be duplicated, so we must
	// verify this is is not a duplicate, and drop if it is
	fp := cAuth.CreateNegotiationFingerprint(partnerPubKey,
		partnerSIDHPubKey)
	newFP, latest := rrs.m.store.CheckIfNegotiationIsNew(partnerID,
		receptionID.Source, fp)

}

func processDecryptedMessage(b []byte) (*id.ID, *sidh.PublicKey, fact.FactList,
	error) {
	//decode the ecr format
	ecrFmt, err := unmarshalEcrFormat(b)
	if err != nil {
		return nil, nil, nil, errors.WithMessage(err, "Failed to "+
			"unmarshal auth request's encrypted payload")
	}

	partnerSIDHPubKey, err := ecrFmt.GetSidhPubKey()
	if err != nil {
		return nil, nil, nil, errors.WithMessage(err, "Could not "+
			"unmarshal partner SIDH Pubkey")
	}

	//decode the request format
	requestFmt, err := newRequestFormat(ecrFmt)
	if err != nil {
		return nil, nil, nil, errors.WithMessage(err, "Failed to "+
			"unmarshal auth request's internal payload")
	}

	partnerID, err := requestFmt.GetID()
	if err != nil {
		return nil, nil, nil, errors.WithMessage(err, "Failed to "+
			"unmarshal auth request's sender ID")
	}

	facts, _, err := fact.UnstringifyFactList(
		string(requestFmt.msgPayload))
	if err != nil {
		return nil, nil, nil, errors.WithMessage(err, "Failed to "+
			"unmarshal auth request's facts")
	}

	return partnerID, partnerSIDHPubKey, facts, nil
}
