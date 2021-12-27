///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package auth

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/interfaces/preimage"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/edge"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/diffieHellman"
	cAuth "gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"io"
)

func ConfirmRequestAuth(partner contact.Contact, rng io.Reader,
	storage *storage.Session, net interfaces.NetworkManager) (id.Round, error) {

	/*edge checking*/

	// check that messages can be sent over the network
	if !net.GetHealthTracker().IsHealthy() {
		return 0, errors.New("Cannot confirm authenticated message " +
			"when the network is not healthy")
	}

	// check if the partner has an auth in progress
	// this takes the lock, from this point forward any errors need to
	// release the lock
	storedContact, theirSidhKey, err := storage.Auth().GetReceivedRequest(
		partner.ID)
	if err != nil {
		return 0, errors.Errorf(
			"failed to find a pending Auth Request: %s",
			err)
	}
	defer storage.Auth().Done(partner.ID)

	// verify the passed contact matches what is stored
	if storedContact.DhPubKey.Cmp(partner.DhPubKey) != 0 {
		storage.Auth().Done(partner.ID)
		return 0, errors.WithMessage(err,
			"Pending Auth Request has different pubkey than stored")
	}

	grp := storage.E2e().GetGroup()

	/*cryptographic generation*/

	//generate ownership proof
	ownership := cAuth.MakeOwnershipProof(storage.E2e().GetDHPrivateKey(),
		partner.DhPubKey, storage.E2e().GetGroup())

	//generate new keypair
	newPrivKey := diffieHellman.GeneratePrivateKey(256, grp, rng)
	newPubKey := diffieHellman.GeneratePublicKey(newPrivKey, grp)

	sidhVariant := util.GetCompatibleSIDHVariant(theirSidhKey.Variant())
	newSIDHPrivKey := util.NewSIDHPrivateKey(sidhVariant)
	newSIDHPubKey := util.NewSIDHPublicKey(sidhVariant)
	newSIDHPrivKey.Generate(rng)
	newSIDHPrivKey.GeneratePublicKey(newSIDHPubKey)

	/*construct message*/
	// we build the payload before we save because it is technically fallible
	// which can get into a bricked state if it fails
	cmixMsg := format.NewMessage(storage.Cmix().GetGroup().GetP().ByteLen())
	baseFmt := newBaseFormat(cmixMsg.ContentsSize(), grp.GetP().ByteLen())
	ecrFmt := newEcrFormat(baseFmt.GetEcrPayloadLen())

	// setup the encrypted payload
	ecrFmt.SetOwnership(ownership)
	ecrFmt.SetSidHPubKey(newSIDHPubKey)
	// confirmation has no custom payload

	//encrypt the payload
	ecrPayload, mac := cAuth.Encrypt(newPrivKey, partner.DhPubKey,
		ecrFmt.data, grp)

	//get the fingerprint from the old ownership proof
	fp := cAuth.MakeOwnershipProofFP(storedContact.OwnershipProof)
	preimg := preimage.Generate(fp[:], preimage.Confirm)

	//final construction
	baseFmt.SetEcrPayload(ecrPayload)
	baseFmt.SetPubKey(newPubKey)

	cmixMsg.SetKeyFP(fp)
	cmixMsg.SetMac(mac)
	cmixMsg.SetContents(baseFmt.Marshal())

	// fixme: channel can get into a bricked state if the first save occurs and
	// the second does not or the two occur and the storage into critical
	// messages does not occur

	events := net.GetEventManager()

	//create local relationship
	p := storage.E2e().GetE2ESessionParams()
	if err := storage.E2e().AddPartner(partner.ID, partner.DhPubKey,
		newPrivKey, theirSidhKey, newSIDHPrivKey,
		p, p); err != nil {
		em := fmt.Sprintf("Failed to create channel with partner (%s) "+
			"on confirmation, this is likley a replay: %s",
			partner.ID, err.Error())
		jww.WARN.Print(em)
		events.Report(10, "Auth", "SendConfirmError", em)
	}

	//add the preimages
	sessionPartner, err := storage.E2e().GetPartner(partner.ID)
	if err != nil {
		jww.FATAL.Panicf("Cannot find %s right after creating: %+v", partner.ID, err)
	}
	me := storage.GetUser().ReceptionID

	//e2e
	storage.GetEdge().Add(edge.Preimage{
		Data:   sessionPartner.GetE2EPreimage(),
		Type:   preimage.E2e,
		Source: partner.ID[:],
	}, me)

	//slient (rekey)
	storage.GetEdge().Add(edge.Preimage{
		Data:   sessionPartner.GetSilentPreimage(),
		Type:   preimage.Silent,
		Source: partner.ID[:],
	}, me)

	// File transfer end
	storage.GetEdge().Add(edge.Preimage{
		Data:   sessionPartner.GetFileTransferPreimage(),
		Type:   preimage.EndFT,
		Source: partner.ID[:],
	}, me)

	// delete the in progress negotiation
	// this unlocks the request lock
	//fixme - do these deletes at a later date
	/*if err := storage.Auth().Delete(partner.ID); err != nil {
		return 0, errors.Errorf("UNRECOVERABLE! Failed to delete in "+
			"progress negotiation with partner (%s) after creating confirmation: %+v",
			partner.ID, err)
	}*/

	jww.INFO.Printf("Confirming Auth with %s, msgDigest: %s",
		partner.ID, cmixMsg.Digest())

	param := params.GetDefaultCMIX()
	param.IdentityPreimage = preimg
	/*send message*/
	round, _, err := net.SendCMIX(cmixMsg, partner.ID, param)
	if err != nil {
		// if the send fails just set it to failed, it will but automatically
		// retried
		jww.INFO.Printf("Auth Confirm with %s (msgDigest: %s) failed "+
			"to transmit: %+v", partner.ID, cmixMsg.Digest(), err)
		return 0, errors.WithMessage(err, "Auth Confirm Failed to transmit")
	}

	em := fmt.Sprintf("Confirm Request with %s (msgDigest: %s) sent on round %d",
		partner.ID, cmixMsg.Digest(), round)
	jww.INFO.Print(em)
	events.Report(1, "Auth", "SendConfirm", em)

	return round, nil
}
