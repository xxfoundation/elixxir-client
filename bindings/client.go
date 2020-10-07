////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/api"
	"time"
)

// BindingsClient wraps the api.Client, implementing additional functions
// to support the gomobile Client interface
type Client struct {
	api api.Client
}

// NewClient creates client storage, generates keys, connects, and registers
// with the network. Note that this does not register a username/identity, but
// merely creates a new cryptographic identity for adding such information
// at a later date.
//
// Users of this function should delete the storage directory on error.
func NewClient(network, storageDir string, password []byte, regCode string) error {
	return api.NewClient(network, storageDir, password, regCode)
}

// NewPrecannedClient creates an insecure user with predetermined keys with nodes
// It creates client storage, generates keys, connects, and registers
// with the network. Note that this does not register a username/identity, but
// merely creates a new cryptographic identity for adding such information
// at a later date.
//
// Users of this function should delete the storage directory on error.
func NewPrecannedClient(precannedID int, network, storageDir string, password []byte) error {

	if precannedID < 0 {
		return errors.New("Cannot create precanned client with negative ID")
	}

	return api.NewPrecannedClient(uint(precannedID), network, storageDir, password)
}

// LoadClient will load an existing client from the storageDir
// using the password. This will fail if the client doesn't exist or
// the password is incorrect.
// The password is passed as a byte array so that it can be cleared from
// memory and stored as securely as possible using the memguard library.
// LoadClient does not block on network connection, and instead loads and
// starts subprocesses to perform network operations.
func LoadClient(storageDir string, password []byte) (*Client, error) {
	// TODO: This should wrap the bindings ClientImpl, when available.
	client, err := api.LoadClient(storageDir, password)
	if err != nil {
		return nil, err
	}
	return &Client{*client}, nil
}

// StartNetworkFollower kicks off the tracking of the network. It starts
// long running network client threads and returns an object for checking
// state and stopping those threads.
// Call this when returning from sleep and close when going back to
// sleep.
// These threads may become a significant drain on battery when offline, ensure
// they are stopped if there is no internet access
// Threads Started:
//   - Network Follower (/network/follow.go)
//   	tracks the network events and hands them off to workers for handling
//   - Historical Round Retrieval (/network/rounds/historical.go)
//		Retrieves data about rounds which are too old to be stored by the client
//	 - Message Retrieval Worker Group (/network/rounds/retrieve.go)
//		Requests all messages in a given round from the gateway of the last node
//	 - Message Handling Worker Group (/network/message/handle.go)
//		Decrypts and partitions messages when signals via the Switchboard
//	 - Health Tracker (/network/health)
//		Via the network instance tracks the state of the network
//	 - Garbled Messages (/network/message/garbled.go)
//		Can be signaled to check all recent messages which could be be decoded
//		Uses a message store on disk for persistence
//	 - Critical Messages (/network/message/critical.go)
//		Ensures all protocol layer mandatory messages are sent
//		Uses a message store on disk for persistence
//	 - KeyExchange Trigger (/keyExchange/trigger.go)
//		Responds to sent rekeys and executes them
//   - KeyExchange Confirm (/keyExchange/confirm.go)
//		Responds to confirmations of successful rekey operations
func (c *Client) StartNetworkFollower() error {
	return c.api.StartNetworkFollower()
}

// StopNetworkFollower stops the network follower if it is running.
// It returns errors if the Follower is in the wrong status to stop or if it
// fails to stop it.
// if the network follower is running and this fails, the client object will
// most likely be in an unrecoverable state and need to be trashed.
func (c *Client) StopNetworkFollower(timeoutMS int) error {
	timeout := time.Duration(timeoutMS) * time.Millisecond
	return c.api.StopNetworkFollower(timeout)
}

// Gets the state of the network follower. Returns:
// Stopped 	- 0
// Starting - 1000
// Running	- 2000
// Stopping	- 3000
func (c *Client) NetworkFollowerStatus() int {
	return int(c.api.NetworkFollowerStatus())
}

// returns true if the network is read to be in a healthy state where
// messages can be sent
func (c *Client) IsNetworkHealthy() bool {
	return c.api.GetHealth().IsHealthy()
}

// registers the network health callback to be called any time the network
// health changes
func (c *Client) RegisterNetworkHealthCB(nhc NetworkHealthCallback) {
	c.api.GetHealth().AddFunc(nhc.Callback)
}

// RegisterListener records and installs a listener for messages
// matching specific uid, msgType, and/or username
func (c *Client) RegisterListener(uid []byte, msgType int,
	username string, listener Listener) {
}

/*
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
