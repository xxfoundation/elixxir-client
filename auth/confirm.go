package auth

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/contact"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/interfaces/utility"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/e2e"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/states"
	"io"
	cAuth "gitlab.com/elixxir/crypto/e2e/auth"
	"time"
	ds "gitlab.com/elixxir/comms/network/dataStructures"
	jww "github.com/spf13/jwalterweatherman"
)

func ConfirmRequestAuth(partner contact.Contact, rng io.Reader,
	storage *storage.Session, net interfaces.NetworkManager) error {

	/*edge checking*/

	// check that messages can be sent over the network
	if !net.GetHealthTracker().IsHealthy() {
		return errors.New("Cannot confirm authenticated message " +
			"when the network is not healthy")
	}

	// check if the partner has an auth in progress
	// this takes the lock, from this point forward any errors need to release
	// the lock
	storedContact, err := storage.Auth().GetReceivedRequest(partner.ID)
	if err != nil {
		return errors.Errorf("failed to find a pending Auth Request: %s",
			err)
	}

	// verify the passed contact matches what is stored
	if storedContact.DhPubKey.Cmp(partner.DhPubKey) != 0 {
		storage.Auth().Fail(partner.ID)
		return errors.Errorf("Pending Auth Request has different "+
			"pubkey than stored",
			err)
	}

	grp := storage.E2e().GetGroup()

	/*cryptographic generation*/

	//generate ownership proof
	ownership := cAuth.MakeOwnershipProof(storage.E2e().GetDHPrivateKey(),
		partner.DhPubKey, storage.E2e().GetGroup())

	//generate new keypair
	newPrivKey := diffieHellman.GeneratePrivateKey(256, grp, rng)
	newPubKey := diffieHellman.GeneratePublicKey(newPrivKey, grp)

	//generate salt
	salt := make([]byte, saltSize)
	_, err = rng.Read(salt)
	if err != nil {
		storage.Auth().Fail(partner.ID)
		return errors.Wrap(err, "Failed to generate salt for "+
			"confirmation")
	}

	/*construct message*/
	// we build the payload before we save because it is technically fallible
	// which can get into a bricked state if it fails
	cmixMsg := format.NewMessage(storage.Cmix().GetGroup().GetP().ByteLen())
	baseFmt := newBaseFormat(cmixMsg.ContentsSize(), grp.GetP().ByteLen())
	ecrFmt := newEcrFormat(baseFmt.GetEcrPayloadLen())

	// setup the encrypted payload
	ecrFmt.SetOwnership(ownership)
	// confirmation has no custom payload

	//encrypt the payload
	ecrPayload, mac := cAuth.Encrypt(newPrivKey, partner.DhPubKey,
		salt, ecrFmt.payload, grp)

	//get the fingerprint from the old ownership proof
	fp := cAuth.MakeOwnershipProofFP(storedContact.OwnershipProof)

	//final construction
	baseFmt.SetEcrPayload(ecrPayload)
	baseFmt.SetSalt(salt)
	baseFmt.SetPubKey(newPubKey)

	cmixMsg.SetKeyFP(fp)
	cmixMsg.SetMac(mac)
	cmixMsg.SetContents(baseFmt.Marshal())

	// fixme: channel can get into a bricked state if the first save occurs and
	// the second does not or the two occur and the storage into critical
	// messages does not occur

	//create local relationship
	p := e2e.GetDefaultSessionParams()
	if err := storage.E2e().AddPartner(partner.ID, partner.DhPubKey, newPrivKey,
		p, p); err != nil {
		storage.Auth().Fail(partner.ID)
		return errors.Errorf("Failed to create channel with partner (%s) "+
			"on confirmation: %+v",
			partner.ID, err)
	}

	// delete the in progress negotiation
	// this unlocks the request lock
	if err := storage.Auth().Delete(partner.ID); err != nil {
		return errors.Errorf("UNRECOVERABLE! Failed to delete in "+
			"progress negotiation with partner (%s) after creating confirmation: %+v",
			partner.ID, err)
	}

	//store the message as a critical message so it will always be sent
	storage.GetCriticalRawMessages().AddProcessing(cmixMsg)

	/*send message*/
	round, err := net.SendCMIX(cmixMsg, params.GetDefaultCMIX())
	if err != nil {
		// if the send fails just set it to failed, it will but automatically
		// retried
		jww.ERROR.Printf("request failed to transmit, will be "+
			"handled on reconnect: %+v", err)
		storage.GetCriticalRawMessages().Failed(cmixMsg)
	}

	/*check message delivery*/
	sendResults := make(chan ds.EventReturn, 1)
	roundEvents := net.GetInstance().GetRoundEvents()

	roundEvents.AddRoundEventChan(round, sendResults, 1*time.Minute,
		states.COMPLETED, states.FAILED)

	success, _, _ := utility.TrackResults(sendResults, 1)
	if !success {
		jww.ERROR.Printf("request failed to transmit, will be " +
			"handled on reconnect")
		storage.GetCriticalRawMessages().Failed(cmixMsg)
	} else {
		storage.GetCriticalRawMessages().Succeeded(cmixMsg)
	}

	return nil
}
