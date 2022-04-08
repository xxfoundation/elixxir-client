package auth

import (
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/primitives/id"
)

type State interface {
	// Request sends a contact request from the user identity in the imported
	// e2e structure to the passed contact, as well as the passed facts (will
	// error if they are too long).
	// The other party must accept the request by calling Confirm in order to be
	// able to send messages using e2e.Handler.SendE2e. When the other party
	// does so, the "confirm" callback will get called.
	// The round the request is initially sent on will be returned, but the
	// request will be listed as a critical message, so the underlying cmix
	// client will auto resend it in the event of failure.
	// A request cannot be sent for a contact who has already received a
	// request or who is already a partner.
	// The request sends as a critical message, if the round send on fails, it
	// will be auto resent by the cmix client
	Request(partner contact.Contact, myfacts fact.FactList) (id.Round, error)

	// Confirm sends a confirmation for a received request. It can only be
	// called once. This both sends keying material to the other party and
	// creates a channel in the e2e handler, after which e2e messages can be
	// sent to the partner using e2e.Handler.SendE2e.
	// The round the request is initially sent on will be returned, but the
	// request will be listed as a critical message, so the underlying cmix
	// client will auto resend it in the event of failure.
	// A confirm cannot be sent for a contact who has not sent a request or
	// who is already a partner. This can only be called once for a specific
	// contact.
	// The confirm sends as a critical message, if the round send on fails, it
	// will be auto resent by the cmix client
	// If the confirm must be resent, use ReplayConfirm
	Confirm(partner contact.Contact) (id.Round, error)

	// Reset sends a contact reset request from the user identity in the
	// imported e2e structure to the passed contact, as well as the passed
	// facts (will error if they are too long).
	// This delete all traces of the relationship with the partner from e2e and
	// create a new relationship from scratch.
	// The round the reset is initially sent on will be returned, but the request
	// will be listed as a critical message, so the underlying cmix client will
	// auto resend it in the event of failure.
	// A request cannot be sent for a contact who has already received a
	// request or who is already a partner.
	Reset(partner contact.Contact) (id.Round, error)

	// ReplayConfirm Resends a confirm to the partner.
	// will fail to send if the send relationship with the partner has already
	// ratcheted
	// The confirm sends as a critical message, if the round send on fails, it
	// will be auto resent by the cmix client
	// This will not be useful if either side has ratcheted
	ReplayConfirm(partner *id.ID) (id.Round, error)

	// ReplayRequests will iterate through all pending contact requests and replay
	// them on the callbacks.
	ReplayRequests()
}
