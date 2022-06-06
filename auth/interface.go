package auth

import (
	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/identity"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
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

	// CallAllReceivedRequests will iterate through all pending contact requests
	// and replay them on the callbacks.
	CallAllReceivedRequests()

	// DeleteRequest deletes sent or received requests for a
	// specific partner ID.
	DeleteRequest(partnerID *id.ID) error

	// DeleteAllRequests clears all requests from client's auth storage.
	DeleteAllRequests() error

	// DeleteSentRequests clears all sent requests from client's auth
	// storage.
	DeleteSentRequests() error

	// DeleteReceiveRequests clears all received requests from client's auth
	// storage.
	DeleteReceiveRequests() error

	// GetReceivedRequest returns a contact if there's a received
	// request for it.
	GetReceivedRequest(partner *id.ID) (contact.Contact, error)

	// VerifyOwnership checks if the received ownership proof is valid
	VerifyOwnership(received, verified contact.Contact, e2e e2e.Handler) bool
}

// Callbacks is the interface for auth callback methods.
type Callbacks interface {
	Request(partner contact.Contact, receptionID receptionID.EphemeralIdentity,
		round rounds.Round)
	Confirm(partner contact.Contact, receptionID receptionID.EphemeralIdentity,
		round rounds.Round)
	Reset(partner contact.Contact, receptionID receptionID.EphemeralIdentity,
		round rounds.Round)
}

// cmixClient is a sub-interface of cmix.Client with
// methods relevant to this package.
type cmixClient interface {
	IsHealthy() bool
	GetMaxMessageLength() int
	AddService(clientID *id.ID, newService message.Service,
		response message.Processor)
	DeleteService(clientID *id.ID, toDelete message.Service,
		processor message.Processor)
	GetIdentity(get *id.ID) (identity.TrackedID, error)
	AddFingerprint(identity *id.ID, fingerprint format.Fingerprint,
		mp message.Processor) error
	DeleteFingerprint(identity *id.ID, fingerprint format.Fingerprint)
	Send(recipient *id.ID, fingerprint format.Fingerprint,
		service message.Service, payload, mac []byte, cmixParams cmix.CMIXParams) (
		id.Round, ephemeral.Id, error)
}

// e2eHandler is a sub-interface of e2e.Handler containing
// methods relevant to this package.
type e2eHandler interface {
	GetHistoricalDHPubkey() *cyclic.Int
	GetHistoricalDHPrivkey() *cyclic.Int
	GetGroup() *cyclic.Group
	AddPartner(partnerID *id.ID,
		partnerPubKey, myPrivKey *cyclic.Int,
		partnerSIDHPubKey *sidh.PublicKey,
		mySIDHPrivKey *sidh.PrivateKey, sendParams,
		receiveParams session.Params) (partner.Manager, error)
	GetPartner(partnerID *id.ID) (partner.Manager, error)
	DeletePartner(partnerId *id.ID) error
	GetReceptionID() *id.ID
}
