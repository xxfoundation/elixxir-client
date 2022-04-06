///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package auth

import (
	"io"
	"strings"

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
	"gitlab.com/xx_network/primitives/id"
)

const terminator = ";"

func RequestAuth(partner, me contact.Contact, rng io.Reader,
	storage *storage.Session, net interfaces.NetworkManager) (id.Round, error) {
	// check that an authenticated channel does not already exist
	if _, err := storage.E2e().GetPartner(partner.ID); err == nil ||
		!strings.Contains(err.Error(), e2e.NoPartnerErrorStr) {
		return 0, errors.Errorf("Authenticated channel already " +
			"established with partner")
	}

	return requestAuth(partner, me, rng, false, storage, net)
}

func ResetSession(partner, me contact.Contact, rng io.Reader,
	storage *storage.Session, net interfaces.NetworkManager) (id.Round, error) {

	// Delete authenticated channel if it exists.
	if err := storage.E2e().DeletePartner(partner.ID); err != nil {
		jww.WARN.Printf("Unable to delete partner when "+
			"resetting session: %+v", err)
	} else {
		// Delete any stored sent/received requests
		storage.Auth().Delete(partner.ID)
	}

	rqType, _, _, err := storage.Auth().GetRequest(partner.ID)
	if err == nil && rqType == auth.Sent {
		return 0, errors.New("Cannot reset a session after " +
			"sending request, caller must resend request instead")
	}

	// Try to initiate a clean session request
	return requestAuth(partner, me, rng, true, storage, net)
}

// requestAuth internal helper
func requestAuth(partner, me contact.Contact, rng io.Reader, reset bool,
	storage *storage.Session, net interfaces.NetworkManager) (id.Round, error) {

	/*edge checks generation*/
	// check that the request is being sent from the proper ID
	if !me.ID.Cmp(storage.GetUser().ReceptionID) {
		return 0, errors.Errorf("Authenticated channel request " +
			"can only be sent from user's identity")
	}

	//denote if this is a resend of an old request
	resend := false

	//lookup if an ongoing request is occurring
	rqType, sr, _, err := storage.Auth().GetRequest(partner.ID)
	if err != nil && !strings.Contains(err.Error(), auth.NoRequest) {
		return 0, errors.WithMessage(err,
			"Cannot send a request after receiving unknown error "+
				"on requesting contact status")
	} else if err == nil {
		switch rqType {
		case auth.Receive:
			if reset {
				storage.Auth().DeleteRequest(partner.ID)
			} else {
				return 0, errors.Errorf("Cannot send a " +
					"request after receiving a request")
			}
		case auth.Sent:
			resend = true
		default:
			return 0, errors.Errorf("Cannot send a request after "+
				"a stored request with unknown rqType: %d",
				rqType)
		}
	}

	/*cryptographic generation*/
	var dhPriv, dhPub *cyclic.Int
	var sidhPriv *sidh.PrivateKey
	var sidhPub *sidh.PublicKey

	// NOTE: E2E group is the group used for DH key exchange, not cMix
	dhGrp := storage.E2e().GetGroup()
	// origin DH Priv key is the DH Key corresponding to the public key
	// registered with user discovery
	originDHPrivKey := storage.E2e().GetDHPrivateKey()

	// If we are resending (valid sent request), reuse those keys
	if resend {
		dhPriv = sr.GetMyPrivKey()
		dhPub = sr.GetMyPubKey()
		sidhPriv = sr.GetMySIDHPrivKey()
		sidhPub = sr.GetMySIDHPubKey()

	} else {
		dhPriv, dhPub = genDHKeys(dhGrp, rng)
		sidhPriv, sidhPub = util.GenerateSIDHKeyPair(
			sidh.KeyVariantSidhA, rng)
	}

	jww.TRACE.Printf("RequestAuth MYPUBKEY: %v", dhPub.Bytes())
	jww.TRACE.Printf("RequestAuth THEIRPUBKEY: %v",
		partner.DhPubKey.Bytes())

	cMixPrimeSize := storage.Cmix().GetGroup().GetP().ByteLen()
	cMixPayloadSize := getMixPayloadSize(cMixPrimeSize)

	sender := storage.GetUser().ReceptionID

	//generate ownership proof
	ownership := cAuth.MakeOwnershipProof(originDHPrivKey, partner.DhPubKey,
		dhGrp)
	confirmFp := cAuth.MakeOwnershipProofFP(ownership)

	// cMix fingerprint so the recipient can recognize this is a
	// request message.
	requestfp := cAuth.MakeRequestFingerprint(partner.DhPubKey)

	// My fact data so we can display in the interface.
	msgPayload := []byte(me.Facts.Stringify() + terminator)

	// Create the request packet.
	request, mac, err := createRequestAuth(sender, msgPayload, ownership,
		dhPriv, dhPub, partner.DhPubKey, sidhPub,
		dhGrp, cMixPayloadSize)
	if err != nil {
		return 0, err
	}
	contents := request.Marshal()

	storage.GetEdge().Add(edge.Preimage{
		Data:   preimage.Generate(confirmFp[:], preimage.Confirm),
		Type:   preimage.Confirm,
		Source: partner.ID[:],
	}, me.ID)

	jww.TRACE.Printf("RequestAuth ECRPAYLOAD: %v", request.GetEcrPayload())
	jww.TRACE.Printf("RequestAuth MAC: %v", mac)

	/*store state*/
	//fixme: channel is bricked if the first store succedes but the second
	//       fails
	//store the in progress auth if this is not a resend.
	if !resend {
		err = storage.Auth().AddSent(partner.ID, partner.DhPubKey,
			dhPriv, dhPub, sidhPriv, sidhPub, confirmFp)
		if err != nil {
			return 0, errors.Errorf(
				"Failed to store auth request: %s", err)
		}
	}

	cMixParams := params.GetDefaultCMIX()
	rndID, err := sendAuthRequest(partner.ID, contents, mac, cMixPrimeSize,
		requestfp, net, cMixParams, reset)
	return rndID, err
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
