///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package auth

import (
	"fmt"
	auth2 "gitlab.com/elixxir/client/auth/store"
	"gitlab.com/elixxir/client/catalog"
	e2e2 "gitlab.com/elixxir/client/e2e/ratchet"
	"gitlab.com/elixxir/client/network"
	"gitlab.com/elixxir/client/network/message"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"io"
	"strings"

	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/interfaces/preimage"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/edge"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	cAuth "gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/xx_network/primitives/id"
)

const terminator = ";"

func (m *Manager) RequestAuth(partner, me contact.Contact,
	originDHPrivKey *cyclic.Int, temporary bool) (id.Round, error) {
	// check that an authenticated channel does not already exist
	if _, err := m.e2e.GetPartner(partner.ID, me.ID); err == nil ||
		!strings.Contains(err.Error(), e2e2.NoPartnerErrorStr) {
		return 0, errors.Errorf("Authenticated channel already " +
			"established with partner")
	}

	return m.requestAuth(partner, me, originDHPrivKey, temporary)
}

// requestAuth internal helper
func (m *Manager) requestAuth(partner, me contact.Contact,
	originDHPrivKey *cyclic.Int, temporary bool) (id.Round, error) {

	//do key generation
	rng := m.rng.GetStream()
	defer rng.Close()

	dhPriv, dhPub := genDHKeys(m.grp, rng)
	sidhPriv, sidhPub := util.GenerateSIDHKeyPair(
		sidh.KeyVariantSidhA, rng)

	ownership := cAuth.MakeOwnershipProof(originDHPrivKey, partner.DhPubKey,
		m.grp)
	confirmFp := cAuth.MakeOwnershipProofFP(ownership)

	// Add the sent request and use the return to build the send. This will
	// replace the send with an old one if one was in process, wasting the key
	// generation above. This is considered a reasonable loss due to the increase
	// in code simplicity of this approach
	sr, err := m.store.AddSent(partner.ID, me.ID, partner.DhPubKey, dhPriv, dhPub,
		sidhPriv, sidhPub, confirmFp, temporary)
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
	msgPayload := []byte(me.Facts.Stringify() + terminator)

	// Create the request packet.
	request, mac, err := createRequestAuth(partner.ID, msgPayload, ownership,
		dhPriv, dhPub, partner.DhPubKey, sidhPub,
		m.grp, m.net.GetMaxMessageLength())
	if err != nil {
		return 0, err
	}
	contents := request.Marshal()

	//todo-register correct service
	m.net.AddService(me.ID, message.Service{
		Identifier: confirmFp[:],
		Tag:        catalog.Confirm,
		Metadata:   partner.ID[:],
	}, nil)

	jww.TRACE.Printf("RequestAuth ECRPAYLOAD: %v", request.GetEcrPayload())
	jww.TRACE.Printf("RequestAuth MAC: %v", mac)

	jww.INFO.Printf("Requesting Auth with %s, msgDigest: %s",
		partner.ID, format.DigestContents(contents))

	p := network.GetDefaultCMIXParams()
	p.DebugTag = "auth.Request"
	s := message.Service{
		Identifier: partner.ID.Marshal(),
		Tag:        catalog.Default,
		Metadata:   nil,
	}
	round, _, err := m.net.SendCMIX(partner.ID, requestfp, s, contents, mac, p)
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
	m.event.Report(1, "Auth", "RequestSent", em)
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
