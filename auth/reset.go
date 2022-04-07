package auth

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/primitives/id"
)

func (s *State) ResetSession(partner contact.Contact) (id.Round, error) {

	// Delete authenticated channel if it exists.
	if err := s.e2e.DeletePartner(partner.ID); err != nil {
		jww.WARN.Printf("Unable to delete partner when "+
			"resetting session: %+v", err)
	}

	//clean any data which is present
	_ = s.store.DeleteConfirmation(partner.ID)
	_ = s.store.DeleteSentRequest(partner.ID)
	_ = s.store.DeleteReceivedRequest(partner.ID)

	// Try to initiate a clean session request
	return s.requestAuth(partner, fact.FactList{}, true)
}
