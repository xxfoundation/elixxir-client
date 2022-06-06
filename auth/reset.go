package auth

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/primitives/id"
)

// Reset sends a contact reset request from the user identity in the imported e2e
// structure to the passed contact, as well as the passed facts (will error if
// they are too long).
// This delete all traces of the relationship with the partner from e2e and
// create a new relationship from scratch.
// The round the reset is initially sent on will be returned, but the request
// will be listed as a critical message, so the underlying cmix client will
// auto resend it in the event of failure.
// A request cannot be sent for a contact who has already received a request or
// who is already a partner.
func (s *state) Reset(partner contact.Contact) (id.Round, error) {

	// Delete authenticated channel if it exists.
	if err := s.e2e.DeletePartner(partner.ID); err != nil {
		jww.WARN.Printf("Unable to delete partner when "+
			"resetting session: %+v", err)
	}

	//clean any data which is present
	_ = s.store.DeleteConfirmation(partner.ID)
	_ = s.store.DeleteSentRequest(partner.ID)
	_ = s.store.DeleteReceivedRequest(partner.ID)

	_ = s.store.DeleteSentRequest(partner.ID)

	// Try to initiate a clean session request
	return s.request(partner, fact.FactList{}, true)
}
