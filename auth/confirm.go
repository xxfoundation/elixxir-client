////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package auth

import (
	"fmt"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/auth/store"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/event"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/crypto/contact"
	cAuth "gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
)

// Confirm sends a confirmation for a received request. It can only be called
// once. This both sends keying material to the other party and creates a
// channel in the e2e handler, after which e2e messages can be sent to the
// partner using e2e.Handler.SendE2e.
// The round the request is initially sent on will be returned, but the request
// will be listed as a critical message, so the underlying cmix client will
// auto resend it in the event of failure.
// A confirm cannot be sent for a contact who has not sent a request or
// who is already a partner. This can only be called once for a specific contact.
// If the request must be resent, use ReplayConfirm
func (s *state) Confirm(partner contact.Contact) (
	id.Round, error) {
	return s.confirm(partner, s.params.ConfirmTag)
}

func (s *state) confirm(partner contact.Contact, serviceTag string) (
	id.Round, error) {

	// check that messages can be sent over the network
	if !s.net.IsHealthy() {
		return 0, errors.New("Cannot confirm authenticated message " +
			"when the network is not healthy")
	}

	var sentRound id.Round

	//run the handler
	err := s.store.HandleReceivedRequest(partner.ID,
		func(rr *store.ReceivedRequest) error {
			// verify the passed contact matches what is stored
			if rr.GetContact().DhPubKey.Cmp(partner.DhPubKey) != 0 {
				return errors.New("pending Auth Request has different " +
					"pubkey than stored")
			}

			/*cryptographic generation*/

			// generate ownership proof
			ownership := cAuth.MakeOwnershipProof(s.e2e.GetHistoricalDHPrivkey(),
				partner.DhPubKey, s.e2e.GetGroup())

			rng := s.rng.GetStream()

			// generate new keypair
			dhPriv, dhPub := genDHKeys(s.e2e.GetGroup(), rng)
			sidhVariant := util.GetCompatibleSIDHVariant(
				rr.GetTheirSidHPubKeyA().Variant())
			sidhPriv, sidhPub := util.GenerateSIDHKeyPair(sidhVariant, rng)

			rng.Close()

			/*construct message*/
			// we build the payload before we save because it is technically
			// fallible which can get into a bricked state if it fails
			baseFmt := newBaseFormat(s.net.GetMaxMessageLength(),
				s.e2e.GetGroup().GetP().ByteLen())
			ecrFmt := newEcrFormat(baseFmt.GetEcrPayloadLen())

			// setup the encrypted payload
			ecrFmt.SetOwnership(ownership)
			ecrFmt.SetSidHPubKey(sidhPub)
			// confirmation has no custom payload

			// encrypt the payload
			ecrPayload, mac := cAuth.Encrypt(dhPriv, partner.DhPubKey,
				ecrFmt.data, s.e2e.GetGroup())

			// get the fingerprint from the old ownership proof
			fp := cAuth.MakeOwnershipProofFP(rr.GetContact().OwnershipProof)

			// final construction
			baseFmt.SetEcrPayload(ecrPayload)
			baseFmt.SetPubKey(dhPub)

			jww.TRACE.Printf("SendConfirm PARTNERPUBKEY: %v",
				partner.DhPubKey.TextVerbose(16, 0))
			jww.TRACE.Printf("SendConfirm MYPUBKEY: %v", dhPub.TextVerbose(16, 0))

			jww.TRACE.Printf("SendConfirm ECRPAYLOAD: %v", baseFmt.GetEcrPayload())
			jww.TRACE.Printf("SendConfirm MAC: %v", mac)

			// warning: channel can get into a bricked state if the first save
			// occurs and the second does not or the two occur and the storage
			// into critical messages does not occur

			// create local relationship
			p := s.sessionParams
			_, err := s.e2e.AddPartner(partner.ID, partner.DhPubKey, dhPriv,
				rr.GetTheirSidHPubKeyA(), sidhPriv, p, p)
			if err != nil {
				em := fmt.Sprintf("Failed to create channel with partner (%s) "+
					"on confirmation, this is likley a replay: %s",
					partner.ID, err.Error())
				jww.WARN.Print(em)
				s.event.Report(10, "Auth", "SendConfirmError", em)
			}

			if s.backupTrigger != nil {
				s.backupTrigger("confirmed authenticated channel")
			}

			jww.INFO.Printf("Confirming Auth from %s to %s, msgDigest: %s",
				partner.ID, s.e2e.GetReceptionID(),
				format.DigestContents(baseFmt.Marshal()))

			//service used for notification only

			/*send message*/
			if err = s.store.StoreConfirmation(partner.ID, baseFmt.Marshal(),
				mac, fp); err == nil {
				jww.WARN.Printf("Failed to store confirmation for replay "+
					"for relationship between %s and %s, cannot be replayed: %+v",
					partner.ID, s.e2e.GetReceptionID(), err)
			}

			//send confirmation
			sentRound, err = sendAuthConfirm(s.net, partner.ID, fp,
				baseFmt.Marshal(), mac, s.event, serviceTag)

			return err
		})
	return sentRound, err
}

func sendAuthConfirm(net cmixClient, partner *id.ID,
	fp format.Fingerprint, payload, mac []byte, event event.Reporter,
	serviceTag string) (
	id.Round, error) {
	svc := message.Service{
		Identifier: fp[:],
		Tag:        serviceTag,
		Metadata:   nil,
	}

	cmixParam := cmix.GetDefaultCMIXParams()
	cmixParam.DebugTag = "auth.Confirm"
	cmixParam.Critical = true
	sentRound, _, err := net.Send(partner, fp, svc, payload, mac, cmixParam)
	if err != nil {
		// if the send fails just set it to failed, it will but automatically
		// retried
		jww.WARN.Printf("Auth Confirm with %s (msgDigest: %s) failed "+
			"to transmit: %+v, will be handled by critical messages",
			partner, format.DigestContents(payload), err)
		return 0, nil
	}

	em := fmt.Sprintf("Confirm Request with %s (msgDigest: %s) sent on round %d",
		partner, format.DigestContents(payload), sentRound)
	jww.INFO.Print(em)
	event.Report(1, "Auth", "SendConfirm", em)
	return sentRound, nil
}
