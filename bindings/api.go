////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"gitlab.com/elixxir/client/api"
)

// BindingsClient wraps the api.Client, implementing additional functions
// to support the gomobile Client interface
type BindingsClient struct {
	api.Client
}

// NewClient connects and registers to the network using a json encoded
// network information string and then creates a new client at the specified
// storageDir using the specified password. This function will fail
// when:
//   - network information cannot be read or the client cannot connect
//     to the network and register within the defined timeout.
//   - storageDir does not exist and cannot be created
//   - It cannot create, read, or write files inside storageDir
//   - Client files already exist inside storageDir.
//   - cryptographic functionality is unavailable (e.g. random number
//     generation)
// The password is passed as a byte array so that it can be cleared from
// memory and stored as securely as possible using the memguard library.
// NewClient will block until the client has completed registration with
// the network permissioning server.
//
// Users of this function should delete the storage directory on error.
func NewClient(network, storageDir string, password []byte) (Client, error) {
	// TODO: This should wrap the bindings ClientImpl, when available.
	client, err := api.NewClient(network, storageDir, password)
	if err != nil {
		return nil, err
	}
	bindingsClient := &BindingsClient{*client}
	return bindingsClient, nil
}

// LoadClient will load an existing client from the storageDir
// using the password. This will fail if the client doesn't exist or
// the password is incorrect.
// The password is passed as a byte array so that it can be cleared from
// memory and stored as securely as possible using the memguard library.
// LoadClient does not block on network connection, and instead loads and
// starts subprocesses to perform network operations.
func LoadClient(storageDir string, password []byte) (Client, error) {
	// TODO: This should wrap the bindings ClientImpl, when available.
	client, err := api.LoadClient(storageDir, password)
	if err != nil {
		return nil, err
	}
	bindingsClient := &BindingsClient{*client}
	return bindingsClient, nil
}

// RegisterListener records and installs a listener for messages
// matching specific uid, msgType, and/or username
func (b *BindingsClient) RegisterListener(uid []byte, msgType int,
	username string, listener Listener) {
}

// SearchWithHandler is a non-blocking search that also registers
// a callback interface for user disovery events.
func (b *BindingsClient) SearchWithHandler(data, separator string,
	searchTypes []byte, hdlr UserDiscoveryHandler) {
}

// RegisterAuthEventsHandler registers a callback interface for channel
// authentication events.
func (b *BindingsClient) RegisterAuthEventsHandler(hdlr AuthEventHandler) {
}

// RegisterRoundEventsHandler registers a callback interface for round
// events.
func (b *BindingsClient) RegisterRoundEventsHandler(hdlr RoundEventHandler) {
}

// SendE2E sends an end-to-end payload to the provided recipient with
// the provided msgType. Returns the list of rounds in which parts of
// the message were sent or an error if it fails.
func (b *BindingsClient) SendE2E(payload, recipient []byte,
	msgType int) (RoundList, error) {
	return nil, nil
}

// SendUnsafe sends an unencrypted payload to the provided recipient
// with the provided msgType. Returns the list of rounds in which parts
// of the message were sent or an error if it fails.
// NOTE: Do not use this function unless you know what you are doing.
// This function always produces an error message in client logging.
func (b *BindingsClient) SendUnsafe(payload, recipient []byte,
	msgType int) (RoundList, error) {
	return nil, nil
}

// SendCMIX sends a "raw" CMIX message payload to the provided
// recipient. Note that both SendE2E and SendUnsafe call SendCMIX.
// Returns the round ID of the round the payload was sent or an error
// if it fails.
func (b *BindingsClient) SendCMIX(payload, recipient []byte) (int, error) {
	return 0, nil
}

// Search accepts a "separator" separated list of search elements with
// an associated list of searchTypes. It returns a ContactList which
// allows you to iterate over the found contact objects.
func (b *BindingsClient) Search(data, separator string,
	searchTypes []byte) ContactList {
	return nil
}
