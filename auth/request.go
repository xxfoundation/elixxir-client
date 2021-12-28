///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package auth

import (
	"fmt"
	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/interfaces/preimage"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/auth"
	"gitlab.com/elixxir/client/storage/e2e"
	"gitlab.com/elixxir/client/storage/edge"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	cAuth "gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"io"
	"strings"
)

const terminator = ";"

func RequestAuth(partner, me contact.Contact, rng io.Reader,
	storage *storage.Session, net interfaces.NetworkManager) (id.Round, error) {
	/*edge checks generation*/

	// check that an authenticated channel does not already exists
	if _, err := storage.E2e().GetPartner(partner.ID); err == nil ||
		!strings.Contains(err.Error(), e2e.NoPartnerErrorStr) {
		return 0, errors.Errorf("Authenticated channel already " +
			"established with partner")
	}

	// check that the request is being sent from the proper ID
	if !me.ID.Cmp(storage.GetUser().ReceptionID) {
		return 0, errors.Errorf("Authenticated channel request " +
			"can only be sent from user's identity")
	}

	//denote if this is a resend of an old request
	resend := false

	//lookup if an ongoing request is occurring
	rqType, sr, _, err := storage.Auth().GetRequest(partner.ID)

	if err == nil {
		if rqType == auth.Receive {
			return 0, errors.Errorf("Cannot send a request after " +
				"receiving a request")
		} else if rqType == auth.Sent {
			resend = true
		} else {
			return 0, errors.Errorf("Cannot send a request after "+
				" a stored request with unknown rqType: %d", rqType)
		}
	} else if !strings.Contains(err.Error(), auth.NoRequest) {
		return 0, errors.WithMessage(err,
			"Cannot send a request after receiving unknown error "+
				"on requesting contact status")
	}

	grp := storage.E2e().GetGroup()

	/*generate embedded message structures and check payload*/
	cmixMsg := format.NewMessage(storage.Cmix().GetGroup().GetP().ByteLen())
	baseFmt := newBaseFormat(cmixMsg.ContentsSize(), grp.GetP().ByteLen())
	ecrFmt := newEcrFormat(baseFmt.GetEcrPayloadLen())
	requestFmt, err := newRequestFormat(ecrFmt)
	if err != nil {
		return 0, errors.Errorf("failed to make request format: %+v", err)
	}

	//check the payload fits
	facts := me.Facts.Stringify()
	msgPayload := facts + terminator
	msgPayloadBytes := []byte(msgPayload)

	/*cryptographic generation*/
	var newPrivKey, newPubKey *cyclic.Int
	var sidHPrivKeyA *sidh.PrivateKey
	var sidHPubKeyA *sidh.PublicKey

	// in this case we have an ongoing request so we can resend the extant
	// request
	if resend {
		newPrivKey = sr.GetMyPrivKey()
		newPubKey = sr.GetMyPubKey()
		sidHPrivKeyA = sr.GetMySIDHPrivKey()
		sidHPubKeyA = sr.GetMySIDHPubKey()
		//in this case it is a new request and we must generate new keys
	} else {
		//generate new keypair
		newPrivKey = diffieHellman.GeneratePrivateKey(256, grp, rng)
		newPubKey = diffieHellman.GeneratePublicKey(newPrivKey, grp)

		sidHPrivKeyA = util.NewSIDHPrivateKey(sidh.KeyVariantSidhA)
		sidHPubKeyA = util.NewSIDHPublicKey(sidh.KeyVariantSidhA)

		if err = sidHPrivKeyA.Generate(rng); err != nil {
			return 0, errors.WithMessagef(err, "RequestAuth: "+
				"could not generate SIDH private key")
		}
		sidHPrivKeyA.GeneratePublicKey(sidHPubKeyA)

	}

	if len(msgPayloadBytes) > requestFmt.MsgPayloadLen() {
		return 0, errors.Errorf("Combined message longer than space "+
			"available in payload; available: %v, length: %v",
			requestFmt.MsgPayloadLen(), len(msgPayloadBytes))
	}

	//generate ownership proof
	ownership := cAuth.MakeOwnershipProof(storage.E2e().GetDHPrivateKey(),
		partner.DhPubKey, storage.E2e().GetGroup())

	jww.TRACE.Printf("RequestAuth MYPUBKEY: %v", newPubKey.Bytes())
	jww.TRACE.Printf("RequestAuth THEIRPUBKEY: %v", partner.DhPubKey.Bytes())

	/*encrypt payload*/
	requestFmt.SetID(storage.GetUser().ReceptionID)
	requestFmt.SetMsgPayload(msgPayloadBytes)
	ecrFmt.SetOwnership(ownership)
	ecrFmt.SetSidHPubKey(sidHPubKeyA)
	ecrPayload, mac := cAuth.Encrypt(newPrivKey, partner.DhPubKey,
		ecrFmt.data, grp)
	confirmFp := cAuth.MakeOwnershipProofFP(ownership)
	requestfp := cAuth.MakeRequestFingerprint(partner.DhPubKey)

	/*construct message*/
	baseFmt.SetEcrPayload(ecrPayload)
	baseFmt.SetPubKey(newPubKey)

	cmixMsg.SetKeyFP(requestfp)
	cmixMsg.SetMac(mac)
	cmixMsg.SetContents(baseFmt.Marshal())

	storage.GetEdge().Add(edge.Preimage{
		Data:   preimage.Generate(confirmFp[:], preimage.Confirm),
		Type:   preimage.Confirm,
		Source: partner.ID[:],
	}, me.ID)

	jww.TRACE.Printf("RequestAuth ECRPAYLOAD: %v", baseFmt.GetEcrPayload())
	jww.TRACE.Printf("RequestAuth MAC: %v", mac)

	/*store state*/
	//fixme: channel is bricked if the first store succedes but the second fails
	//store the in progress auth
	if !resend {
		err = storage.Auth().AddSent(partner.ID, partner.DhPubKey, newPrivKey,
			newPubKey, sidHPrivKeyA, sidHPubKeyA, confirmFp)
		if err != nil {
			return 0, errors.Errorf("Failed to store auth request: %s", err)
		}
	}

	jww.INFO.Printf("Requesting Auth with %s, msgDigest: %s",
		partner.ID, cmixMsg.Digest())

	/*send message*/
	p := params.GetDefaultCMIX()
	p.IdentityPreimage = preimage.GenerateRequest(partner.ID)
	round, _, err := net.SendCMIX(cmixMsg, partner.ID, p)
	if err != nil {
		// if the send fails just set it to failed, it will
		// but automatically retried
		return 0, errors.WithMessagef(err, "Auth Request with %s "+
			"(msgDigest: %s) failed to transmit: %+v", partner.ID,
			cmixMsg.Digest(), err)
	}

	em := fmt.Sprintf("Auth Request with %s (msgDigest: %s) sent"+
		" on round %d", partner.ID, cmixMsg.Digest(), round)
	jww.INFO.Print(em)
	net.GetEventManager().Report(1, "Auth", "RequestSent", em)

	return round, nil
}
