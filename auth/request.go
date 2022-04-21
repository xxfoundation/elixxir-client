///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package auth

import (
	"fmt"
	"io"
	"strings"

	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/ratchet"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	cAuth "gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
)

const terminator = ";"

// Request sends a contact request from the user identity in the imported e2e
// structure to the passed contact, as well as the passed facts (will error if
// they are too long).
// The other party must accept the request by calling Confirm in order to be
// able to send messages using e2e.Handler.SendE2e. When the other party does so,
// the "confirm" callback will get called.
// The round the request is initially sent on will be returned, but the request
// will be listed as a critical message, so the underlying cmix client will
// auto resend it in the event of failure.
// A request cannot be sent for a contact who has already received a request or
// who is already a partner.
func (s *state) Request(partner contact.Contact, myfacts fact.FactList) (id.Round, error) {
	// check that an authenticated channel does not already exist
	if _, err := s.e2e.GetPartner(partner.ID); err == nil ||
		!strings.Contains(err.Error(), ratchet.NoPartnerErrorStr) {
		return 0, errors.Errorf("Authenticated channel already " +
			"established with partner")
	}

	return s.request(partner, myfacts, false)
}

// request internal helper
func (s *state) request(partner contact.Contact, myfacts fact.FactList, reset bool) (id.Round, error) {

	jww.INFO.Printf("request(...) called")

	//do key generation
	rng := s.rng.GetStream()
	defer rng.Close()

	me := s.e2e.GetReceptionID()

	dhPriv, dhPub := genDHKeys(s.e2e.GetGroup(), rng)
	sidhPriv, sidhPub := util.GenerateSIDHKeyPair(
		sidh.KeyVariantSidhA, rng)

	ownership := cAuth.MakeOwnershipProof(s.e2e.GetHistoricalDHPrivkey(),
		partner.DhPubKey, s.e2e.GetGroup())
	confirmFp := cAuth.MakeOwnershipProofFP(ownership)

	// Add the sent request and use the return to build the send. This will
	// replace the send with an old one if one was in process, wasting the key
	// generation above. This is considered a reasonable loss due to the increase
	// in code simplicity of this approach
	sr, err := s.store.AddSent(partner.ID, partner.DhPubKey, dhPriv, dhPub,
		sidhPriv, sidhPub, confirmFp, reset)
	if err != nil {
		if sr == nil {
			return 0, err
		} else {
			jww.INFO.Printf("Resending request to %s from %s because "+
				"one was already sent", partner.ID, me)
		}
	}

	// cMix fingerprint. Used in old versions by the recipient can recognize
	// this is a request message. Unchanged for backwards compatability
	// (the SIH is used now)
	requestfp := cAuth.MakeRequestFingerprint(partner.DhPubKey)

	// My fact data so we can display in the interface.
	msgPayload := []byte(myfacts.Stringify() + terminator)

	// Create the request packet.
	request, mac, err := createRequestAuth(partner.ID, msgPayload, ownership,
		dhPriv, dhPub, partner.DhPubKey, sidhPub,
		s.e2e.GetGroup(), s.net.GetMaxMessageLength())
	if err != nil {
		return 0, err
	}
	contents := request.Marshal()

	jww.TRACE.Printf("Request ECRPAYLOAD: %v", request.GetEcrPayload())
	jww.TRACE.Printf("Request MAC: %v", mac)

	jww.INFO.Printf("Requesting Auth with %s, msgDigest: %s",
		partner.ID, format.DigestContents(contents))

	//register the confirm fingerprint to pick up confirm
	err = s.net.AddFingerprint(me, confirmFp, &receivedConfirmService{
		s:           s,
		SentRequest: sr,
	})
	if err != nil {
		return 0, errors.Errorf("cannot register fingerprint request "+
			"to %s from %s, bailing request: %+v", partner.ID, me,
			err)
	}

	//register service for notification on confirmation
	s.net.AddService(me, message.Service{
		Identifier: confirmFp[:],
		Tag:        s.params.getConfirmTag(reset),
		Metadata:   partner.ID[:],
	}, nil)

	p := cmix.GetDefaultCMIXParams()
	p.DebugTag = "auth.Request"
	svc := message.Service{
		Identifier: partner.ID.Marshal(),
		Tag:        s.params.RequestTag,
		Metadata:   nil,
	}
	round, _, err := s.net.Send(partner.ID, requestfp, svc, contents, mac, p)
	if err != nil {
		// if the send fails just set it to failed, it will
		// but automatically retried
		return 0, errors.WithMessagef(err, "Auth Request with %s "+
			"(msgDigest: %s) failed to transmit: %+v", partner.ID,
			format.DigestContents(contents), err)
	}

	em := fmt.Sprintf("Auth Request with %s (msgDigest: %s) sent"+
		" on round %d", partner.ID, format.DigestContents(contents), round)
	jww.INFO.Print(em)
	s.event.Report(1, "Auth", "RequestSent", em)
	return round, nil

}

// genDHKeys is a short helper to generate a Diffie-Helman Keypair
func genDHKeys(dhGrp *cyclic.Group, csprng io.Reader) (priv, pub *cyclic.Int) {
	numBytes := len(dhGrp.GetPBytes())
	newPrivKey := diffieHellman.GeneratePrivateKey(numBytes, dhGrp, csprng)
	newPubKey := diffieHellman.GeneratePublicKey(newPrivKey, dhGrp)
	return newPrivKey, newPubKey
}

// createRequestAuth Creates the request packet, including encrypting the
// required parts of it.
func createRequestAuth(sender *id.ID, payload, ownership []byte, myDHPriv,
	myDHPub, theirDHPub *cyclic.Int, mySIDHPub *sidh.PublicKey,
	dhGrp *cyclic.Group, cMixSize int) (*baseFormat, []byte, error) {
	/*generate embedded message structures and check payload*/
	dhPrimeSize := dhGrp.GetP().ByteLen()

	// FIXME: This base -> ecr -> request structure is a little wonky.
	// We should refactor so that is is more direct.
	// I recommend we move to a request object that takes:
	//   sender, dhPub, sidhPub, ownershipProof, payload
	// with a Marshal/Unmarshal that takes the Dh/grp needed to gen
	// the session key and encrypt or decrypt.

	// baseFmt wraps ecrFmt. ecrFmt is encrypted
	baseFmt := newBaseFormat(cMixSize, dhPrimeSize)
	// ecrFmt wraps requestFmt
	ecrFmt := newEcrFormat(baseFmt.GetEcrPayloadLen())
	requestFmt, err := newRequestFormat(ecrFmt)
	if err != nil {
		return nil, nil, errors.Errorf("failed to make request format: %+v", err)
	}

	if len(payload) > requestFmt.MsgPayloadLen() {
		return nil, nil, errors.Errorf(
			"Combined message longer than space "+
				"available in payload; available: %v, length: %v",
			requestFmt.MsgPayloadLen(), len(payload))
	}

	/*encrypt payload*/
	requestFmt.SetID(sender)
	requestFmt.SetMsgPayload(payload)
	ecrFmt.SetOwnership(ownership)
	ecrFmt.SetSidHPubKey(mySIDHPub)
	ecrPayload, mac := cAuth.Encrypt(myDHPriv, theirDHPub, ecrFmt.data,
		dhGrp)
	/*construct message*/
	baseFmt.SetEcrPayload(ecrPayload)
	baseFmt.SetPubKey(myDHPub)

	return &baseFmt, mac, nil
}

func (s *state) GetReceivedRequest(partner *id.ID) (contact.Contact, error) {
	return s.store.GetReceivedRequest(partner)
}

func (s *state) VerifyOwnership(received, verified contact.Contact,
	e2e e2e.Handler) bool {
	return VerifyOwnership(received, verified, e2e)
}

func (s *state) DeleteRequest(partnerID *id.ID) error {
	return s.store.DeleteRequest(partnerID)
}

func (s *state) DeleteAllRequests() error {
	return s.store.DeleteAllRequests()
}

func (s *state) DeleteSentRequests() error {
	return s.store.DeleteSentRequests()
}

func (s *state) DeleteReceiveRequests() error {
	return s.store.DeleteReceiveRequests()
}
