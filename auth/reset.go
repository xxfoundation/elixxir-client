package auth

import (
	jww "github.com/spf13/jwalterweatherman"
	auth2 "gitlab.com/elixxir/client/auth/store"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/xx_network/primitives/id"
	"io"
)

func (m *Manager) ResetSession(partner, me contact.Contact, rng io.Reader) (id.Round, error) {

	// Delete authenticated channel if it exists.
	if err := storage.E2e().DeletePartner(partner.ID); err != nil {
		jww.WARN.Printf("Unable to delete partner when "+
			"resetting session: %+v", err)
	} else {
		// Delete any stored sent/received requests
		storage.Auth().Delete(partner.ID)
	}

	rqType, _, _, err := storage.Auth().GetRequest(partner.ID)
	if err == nil && rqType == auth2.Sent {
		return 0, errors.New("Cannot reset a session after " +
			"sending request, caller must resend request instead")
	}

	// Try to initiate a clean session request
	return requestAuth(partner, me, rng, true, storage, net)
}
