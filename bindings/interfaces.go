////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package bindings

/*
// Client is defined inside the api package. At minimum, it implements all of
// functionality defined here. A Client handles all network connectivity, key
// generation, and storage for a given cryptographic identity on the cmix
// network.
// These threads may become a significant drain on battery when offline, ensure
// they are stopped if there is no internet access
type client interface {
	// ----- Network -----
	// StartNetworkFollower kicks off the tracking of the network. It starts
	// long running network client threads and returns an object for checking
	// state and stopping those threads.
	// Call this when returning from sleep and close when going back to
	// sleep.
	StartNetworkFollower() error

	// StopNetworkFollower stops the network follower if it is running.
	// It returns errors if the Follower is in the wrong status to stop or if it
	// fails to stop it.
	// if the network follower is running and this fails, the client object will
	// most likely be in an unrecoverable state and need to be trashed.
	StopNetworkFollower(timeoutMS int) error

	// NetworkFollowerStatus gets the state of the network follower.
	// Returns:
	// 	Stopped 	- 0
	// 	Starting	- 1000
	// 	Running		- 2000
	// 	Stopping	- 3000
	NetworkFollowerStatus() int

	// Returns true if the following of the network is in a state where messages
	// can be sent, false otherwise
	IsNetworkHealthy() bool

	// Registers a callback which gets triggered every time network health
	// changes
	RegisterNetworkHealthCB(func(bool))

	// ----- Reception -----

	// RegisterListener records and installs a listener for messages
	// matching specific uid, msgType, and/or username
	RegisterListener(uid []byte, msgType int, username string,
		listener Listener)

	// ----- Transmission -----

	// SendE2E sends an end-to-end payload to the provided recipient with
	// the provided msgType. Returns the list of rounds in which parts of
	// the message were sent or an error if it fails.
	SendE2E(payload, recipient []byte, msgType int) (RoundList, error)
	// SendUnsafe sends an unencrypted payload to the provided recipient
	// with the provided msgType. Returns the list of rounds in which parts
	// of the message were sent or an error if it fails.
	// NOTE: Do not use this function unless you know what you are doing.
	// This function always produces an error message in client logging.
	SendUnsafe(payload, recipient []byte, msgType int) (RoundList, error)
	// SendCMIX sends a "raw" CMIX message payload to the provided
	// recipient. Note that both SendE2E and SendUnsafe call SendCMIX.
	// Returns the round ID of the round the payload was sent or an error
	// if it fails.
	SendCMIX(payload, recipient []byte) (int, error)

	// ----- Notifications -----

	// RegisterForNotifications allows a client to register for push
	// notifications.
	// Note that clients are not required to register for push notifications
	// especially as these rely on third parties (i.e., Firebase *cough*
	// *cough* google's palantir *cough*) that may represent a security
	// risk to the user.
	RegisterForNotifications(token []byte) error
	// UnregisterForNotifications turns of notifications for this client
	UnregisterForNotifications() error

	// ----- Registration -----

	// Returns true if the cryptographic identity has been registered with
	// the CMIX user discovery agent.
	// Note that clients do not need to perform this step if they use
	// out of band methods to exchange cryptographic identities
	// (e.g., QR codes), but failing to be registered precludes usage
	// of the user discovery mechanism (this may be preferred by user).
	IsRegistered() bool

	// RegisterIdentity registers an arbitrary username with the user
	// discovery protocol. Returns an error when it cannot connect or
	// the username is already registered.
	RegisterIdentity(username string) error
	// RegisterEmail makes the users email searchable after confirmation.
	// It returns a registration confirmation token to be used with
	// ConfirmRegistration or an error on failure.
	RegisterEmail(email string) ([]byte, error)
	// RegisterPhone makes the users phone searchable after confirmation.
	// It returns a registration confirmation token to be used with
	// ConfirmRegistration or an error on failure.
	RegisterPhone(phone string) ([]byte, error)
	// ConfirmRegistration sends the user discovery agent a confirmation
	// token (from Register Email/Phone) and code (string sent via Email
	// or SMS to confirm ownership) to confirm ownership.
	ConfirmRegistration(token, code []byte) error

	// ----- Contacts -----

	// GetUser returns the current user Identity for this client. This
	// can be serialized into a byte stream for out-of-band sharing.
	GetUser() (bind.Contact, error)

	// ----- User Discovery -----

	// Search accepts a "separator" separated list of search elements with
	// an associated list of searchTypes. It returns a ContactList which
	// allows you to iterate over the found contact objects.
	Search(data, separator string, searchTypes []byte) ContactList
	// SearchWithHandler is a non-blocking search that also registers
	// a callback interface for user disovery events.
	SearchWithHandler(data, separator string, searchTypes []byte,
		hdlr UserDiscoveryHandler)

	// ----- Key Exchange -----

	// CreateAuthenticatedChannel creates a 1-way authenticated channel
	// so this user can send messages to the desired recipient Contact.
	// To receive confirmation from the remote user, clients must
	// register a listener to do that.
	CreateAuthenticatedChannel(recipient bind.Contact, payload []byte) error
	// RegierAuthEventsHandler registers a callback interface for channel
	// authentication events.
	RegisterAuthEventsHandler(hdlr AuthEventHandler)

	// ----- Network -----

	// RegisterRoundEventsHandler registers a callback interface for round
	// events.
	RegisterRoundEventsHandler()
}

// ContactList contains a list of contacts
type ContactList interface {
	// GetLen returns the number of contacts in the list
	GetLen() int
	// GetContact returns the contact at index i
	GetContact(i int) bind.Contact
}

// ----- Callback interfaces -----



// AuthEventHandler handles authentication requests initiated by
// CreateAuthenticatedChannel
type AuthEventHandler interface {
	// HandleConfirmation handles AuthEvents received after
	// the client has called CreateAuthenticatedChannel for
	// the provided contact. Payload is typically empty but
	// may include a small introductory message.
	HandleConfirmation(contact bind.Contact, payload []byte)
	// HandleRequest handles AuthEvents received before
	// the client has called CreateAuthenticatedChannel for
	// the provided contact. It should prompt the user to accept
	// the channel creation "request" and, if approved,
	// call CreateAuthenticatedChannel for this Contact.
	HandleRequest(contact bind.Contact, payload []byte)
}

// UserDiscoveryHandler handles search results against the user discovery agent.
type UserDiscoveryHandler interface {
	HandleSearchResults(results ContactList)
}*/


// Message is a message received from the cMix network in the clear
// or that has been decrypted using established E2E keys.
type Message interface {
	//Returns the id of the message
	GetID() []byte

	// Returns the message's sender ID, if available
	GetSender() []byte

	// Returns the message payload/contents
	// Parse this with protobuf/whatever according to the message type
	GetPayload() []byte

	// Returns the message's type
	GetMessageType() int

	// Returns the message's timestamp in milliseconds since unix epoc
	GetTimestampMS() int
	// Returns the message's timestamp in ns since unix epoc
	GetTimestampNano() int
}

type User interface {
	GetID() []byte
	GetSalt() []byte
	GetRSAPrivateKeyPem() []byte
	GetRSAPublicKeyPem() []byte
	//IsPrecanned() int
	GetCmixDhPrivateKey() []byte
	GetCmixDhPublicKey() []byte
	GetE2EDhPrivateKey() []byte
	GetE2EDhPublicKey() []byte
}