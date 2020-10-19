package auth

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/contact"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"io"
	cAuth "gitlab.com/elixxir/crypto/e2e/auth"
)

func ConfirmRequestAuth(partner contact.Contact, rng io.Reader,
	storage *storage.Session, net interfaces.NetworkManager) error {

	// check that messages can be sent over the network
	if !net.GetHealthTracker().IsHealthy() {
		return errors.New("Cannot confirm authenticated message " +
			"when the network is not healthy")
	}

	// check if the partner has an auth in progress
	storedContact, err := storage.Auth().GetReceivedRequest(partner.ID)
	if err != nil {
		return errors.Errorf("failed to find a pending Auth Request: %s",
			err)
	}

	// verify the passed contact matches what is stored
	if storedContact.DhPubKey.Cmp(partner.DhPubKey) != 0 {
		return errors.Errorf("Pending Auth Request has different "+
			"pubkey than stored",
			err)
	}

	grp := storage.E2e().GetGroup()

	//generate ownership proof
	ownership := cAuth.MakeOwnershipProof(storage.E2e().GetDHPrivateKey(),
		partner.DhPubKey, storage.E2e().GetGroup())

	//generate new keypair
	newPrivKey := diffieHellman.GeneratePrivateKey(256, grp, rng)
	newPubKey := diffieHellman.GeneratePublicKey(newPrivKey, grp)


}
