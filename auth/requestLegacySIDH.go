////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package auth

import (
	"fmt"

	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/message"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	cAuth "gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
)

// request internal helper
func (s *state) requestLegacySIDH(partner contact.Contact, myfacts fact.FactList,
	reset bool) (id.Round, error) {

	jww.INFO.Printf("request(...) called")

	//do key generation
	rng := s.rng.GetStream()
	defer rng.Close()

	me := s.e2e.GetReceptionID()

	dhGrp := s.e2e.GetGroup()

	dhPriv, dhPub := genDHKeys(dhGrp, rng)
	sidhPriv, sidhPub := util.GenerateSIDHKeyPair(
		sidh.KeyVariantSidhA, rng)

	historicalDHPriv := s.e2e.GetHistoricalDHPrivkey()
	historicalDHPub := diffieHellman.GeneratePublicKey(historicalDHPriv,
		dhGrp)

	if !dhGrp.Inside(partner.DhPubKey.GetLargeInt()) {
		return 0, errors.Errorf("partner's DH public key is not in the E2E "+
			"group; E2E group fingerprint is %d and DH key has %d",
			dhGrp.GetFingerprint(), partner.DhPubKey.GetGroupFingerprint())
	}
	ownership := cAuth.MakeOwnershipProof(historicalDHPriv,
		partner.DhPubKey, dhGrp)
	confirmFp := cAuth.MakeOwnershipProofFP(ownership)

	// Add the sent request and use the return to build the
	// send. This will replace the send with an old one if one was
	// in process, wasting the key generation above. This is
	// considered a reasonable loss due to the increase in code
	// simplicity of this approach
	sr, err := s.store.AddSentLegacySIDH(partner.ID, partner.DhPubKey, dhPriv, dhPub,
		sidhPriv, sidhPub, confirmFp, reset)
	if err != nil {
		if sr == nil {
			return 0, err
		} else {
			jww.INFO.Printf("Resending request to %s from %s as "+
				"one was already sent", partner.ID, me)
			dhPriv = sr.GetMyPrivKey()
			dhPub = sr.GetMyPubKey()
			//sidhPriv = sr.GetMySIDHPrivKey()
			sidhPub = sr.GetMySIDHPubKey()
		}
	}

	// cMix fingerprint. Used in old versions by the recipient can recognize
	// this is a request message. Unchanged for backwards compatability
	// (the SIH is used now)
	requestfp := cAuth.MakeRequestFingerprint(partner.DhPubKey)

	// My fact data so we can display in the interface.
	msgPayload := []byte(myfacts.Stringify() + terminator)

	// Create the request packet.
	request, mac, err := createLegacySIDHRequestAuth(me, msgPayload, ownership,
		dhPriv, dhPub, partner.DhPubKey, sidhPub,
		s.e2e.GetGroup(), s.net.GetMaxMessageLength())
	if err != nil {
		return 0, err
	}
	contents := request.Marshal()

	jww.TRACE.Printf("AuthRequest MYPUBKEY: %v", dhPub.TextVerbose(16, 0))
	jww.TRACE.Printf("AuthRequest PARTNERPUBKEY: %v",
		partner.DhPubKey.TextVerbose(16, 0))
	jww.TRACE.Printf("AuthRequest MYSIDHPUBKEY: %s",
		util.StringSIDHPubKey(sidhPub))

	jww.TRACE.Printf("AuthRequest HistoricalPUBKEY: %v",
		historicalDHPub.TextVerbose(16, 0))

	jww.TRACE.Printf("AuthRequest ECRPAYLOAD: %v", request.GetEcrPayload())
	jww.TRACE.Printf("AuthRequest MAC: %v", mac)

	jww.INFO.Printf("Requesting Auth with %s, msgDigest: %s, confirmFp: %s",
		partner.ID, format.DigestContents(contents), confirmFp)

	p := cmix.GetDefaultCMIXParams()
	p.DebugTag = "auth.Request"
	tag := s.params.RequestTag
	if reset {
		tag = s.params.ResetRequestTag
	}
	svc := message.Service{
		Identifier: partner.ID.Marshal(),
		Tag:        tag,
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
		" on round %d", partner.ID, format.DigestContents(contents),
		round)
	jww.INFO.Print(em)
	s.event.Report(1, "Auth", "RequestSent", em)
	return round, nil

}

// createRequestAuth Creates the request packet, including encrypting the
// required parts of it.
func createLegacySIDHRequestAuth(sender *id.ID, payload, ownership []byte, myDHPriv,
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
	baseFmt := newLegacySIDHBaseFormat(cMixSize, dhPrimeSize)
	// ecrFmt wraps requestFmt
	ecrFmt := newLegacySIDHEcrFormat(baseFmt.GetEcrPayloadLen())
	requestFmt, err := newRequestFormat(ecrFmt.GetPayload())
	if err != nil {
		return nil, nil, errors.Errorf(
			"failed to make request format: %+v", err)
	}

	if len(payload) > requestFmt.MsgPayloadLen() {
		return nil, nil, errors.Errorf("Combined message "+
			"longer than space available in "+
			"payload; available: %v, length: %v",
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
