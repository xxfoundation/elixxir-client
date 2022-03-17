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
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/interfaces/preimage"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/edge"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/crypto/contact"
	cAuth "gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
)

func (m *Manager) ConfirmRequestAuth(partner contact.Contact) (id.Round, error) {

	/*edge checking*/

	// check that messages can be sent over the network
	if !m.net.GetHealthTracker().IsHealthy() {
		return 0, errors.New("Cannot confirm authenticated message " +
			"when the network is not healthy")
	}

	// Cannot confirm already established channels
	if _, err := m.storage.E2e().GetPartner(partner.ID); err == nil {
		em := fmt.Sprintf("Cannot ConfirmRequestAuth for %s, "+
			"channel already exists. Ignoring", partner.ID)
		jww.WARN.Print(em)
		m.net.GetEventManager().Report(5, "Auth",
			"ConfirmRequestAuthIgnored", em)
		//exit
		return 0, errors.New(em)
	}

	// check if the partner has an auth in progress
	// this takes the lock, from this point forward any errors need to
	// release the lock
	storedContact, theirSidhKey, err := m.storage.Auth().GetReceivedRequest(
		partner.ID)
	if err != nil {
		return 0, errors.Errorf(
			"failed to find a pending Auth Request: %s",
			err)
	}
	defer m.storage.Auth().Done(partner.ID)

	// verify the passed contact matches what is stored
	if storedContact.DhPubKey.Cmp(partner.DhPubKey) != 0 {
		return 0, errors.WithMessage(err,
			"Pending Auth Request has different pubkey than stored")
	}

	grp := m.storage.E2e().GetGroup()

	/*cryptographic generation*/

	// generate ownership proof
	ownership := cAuth.MakeOwnershipProof(m.storage.E2e().GetDHPrivateKey(),
		partner.DhPubKey, m.storage.E2e().GetGroup())

	rng := m.rng.GetStream()

	// generate new keypair
	dhGrp := grp
	dhPriv, dhPub := genDHKeys(dhGrp, rng)
	sidhVariant := util.GetCompatibleSIDHVariant(theirSidhKey.Variant())
	sidhPriv, sidhPub := util.GenerateSIDHKeyPair(sidhVariant, rng)

	rng.Close()

	/*construct message*/
	// we build the payload before we save because it is technically fallible
	// which can get into a bricked state if it fails
	cmixMsg := format.NewMessage(m.storage.Cmix().GetGroup().GetP().ByteLen())
	baseFmt := newBaseFormat(cmixMsg.ContentsSize(), grp.GetP().ByteLen())
	ecrFmt := newEcrFormat(baseFmt.GetEcrPayloadLen())

	// setup the encrypted payload
	ecrFmt.SetOwnership(ownership)
	ecrFmt.SetSidHPubKey(sidhPub)
	// confirmation has no custom payload

	// encrypt the payload
	ecrPayload, mac := cAuth.Encrypt(dhPriv, partner.DhPubKey,
		ecrFmt.data, grp)

	// get the fingerprint from the old ownership proof
	fp := cAuth.MakeOwnershipProofFP(storedContact.OwnershipProof)
	preimg := preimage.Generate(fp[:], preimage.Confirm)

	// final construction
	baseFmt.SetEcrPayload(ecrPayload)
	baseFmt.SetPubKey(dhPub)

	cmixMsg.SetKeyFP(fp)
	cmixMsg.SetMac(mac)
	cmixMsg.SetContents(baseFmt.Marshal())

	jww.TRACE.Printf("SendConfirm cMixMsg contents: %v",
		cmixMsg.GetContents())

	jww.TRACE.Printf("SendConfirm PARTNERPUBKEY: %v",
		partner.DhPubKey.Bytes())
	jww.TRACE.Printf("SendConfirm MYPUBKEY: %v", dhPub.Bytes())

	jww.TRACE.Printf("SendConfirm ECRPAYLOAD: %v", baseFmt.GetEcrPayload())
	jww.TRACE.Printf("SendConfirm MAC: %v", mac)

	// fixme: channel can get into a bricked state if the first save occurs and
	// the second does not or the two occur and the storage into critical
	// messages does not occur

	events := m.net.GetEventManager()

	// create local relationship
	p := m.storage.E2e().GetE2ESessionParams()
	if err := m.storage.E2e().AddPartner(partner.ID, partner.DhPubKey,
		dhPriv, theirSidhKey, sidhPriv,
		p, p); err != nil {
		em := fmt.Sprintf("Failed to create channel with partner (%s) "+
			"on confirmation, this is likley a replay: %s",
			partner.ID, err.Error())
		jww.WARN.Print(em)
		events.Report(10, "Auth", "SendConfirmError", em)
	}

	m.backupTrigger("confirmed authenticated channel")

	addPreimages(partner.ID, m.storage)

	// delete the in progress negotiation
	// this unlocks the request lock
	// fixme - do these deletes at a later date
	/*if err := storage.Auth().Delete(partner.ID); err != nil {
		return 0, errors.Errorf("UNRECOVERABLE! Failed to delete in "+
			"progress negotiation with partner (%s) after creating confirmation: %+v",
			partner.ID, err)
	}*/

	jww.INFO.Printf("Confirming Auth with %s, msgDigest: %s",
		partner.ID, cmixMsg.Digest())

	param := params.GetDefaultCMIX()
	param.IdentityPreimage = preimg
	param.DebugTag = "auth.Confirm"
	/*send message*/
	round, _, err := m.net.SendCMIX(cmixMsg, partner.ID, param)
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

func addPreimages(partner *id.ID, store *storage.Session) {
	// add the preimages
	sessionPartner, err := store.E2e().GetPartner(partner)
	if err != nil {
		jww.FATAL.Panicf("Cannot find %s right after creating: %+v",
			partner, err)
	}

	// Delete any known pre-existing edges for this partner
	existingEdges, _ := store.GetEdge().Get(partner)
	for i := range existingEdges {
		delete := true
		switch existingEdges[i].Type {
		case preimage.E2e:
		case preimage.Silent:
		case preimage.EndFT:
		case preimage.GroupRq:
		default:
			delete = false
		}

		if delete {
			err = store.GetEdge().Remove(existingEdges[i], partner)
			if err != nil {
				jww.ERROR.Printf(
					"Unable to delete %s edge for %s: %v",
					existingEdges[i].Type, partner, err)
			}
		}
	}

	me := store.GetUser().ReceptionID

	// e2e
	store.GetEdge().Add(edge.Preimage{
		Data:   sessionPartner.GetE2EPreimage(),
		Type:   preimage.E2e,
		Source: partner[:],
	}, me)

	// silent (rekey)
	store.GetEdge().Add(edge.Preimage{
		Data:   sessionPartner.GetSilentPreimage(),
		Type:   preimage.Silent,
		Source: partner[:],
	}, me)

	// File transfer end
	store.GetEdge().Add(edge.Preimage{
		Data:   sessionPartner.GetFileTransferPreimage(),
		Type:   preimage.EndFT,
		Source: partner[:],
	}, me)

	// group Request
	store.GetEdge().Add(edge.Preimage{
		Data:   sessionPartner.GetGroupRequestPreimage(),
		Type:   preimage.GroupRq,
		Source: partner[:],
	}, me)
}
