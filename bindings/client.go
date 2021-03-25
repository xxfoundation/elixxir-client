///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"
	"errors"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/interfaces/contact"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// sets the log level
func init() {
	jww.SetLogThreshold(jww.LevelInfo)
	jww.SetStdoutThreshold(jww.LevelInfo)
}

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
	if err := api.NewClient(network, storageDir, password, regCode); err != nil {
		return errors.New(fmt.Sprintf("Failed to create new client: %+v",
			err))
	}
	return nil
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

	if err := api.NewPrecannedClient(uint(precannedID), network, storageDir, password); err != nil {
		return errors.New(fmt.Sprintf("Failed to create new precanned "+
			"client: %+v", err))
	}
	return nil
}

// Login will load an existing client from the storageDir
// using the password. This will fail if the client doesn't exist or
// the password is incorrect.
// The password is passed as a byte array so that it can be cleared from
// memory and stored as securely as possible using the memguard library.
// Login does not block on network connection, and instead loads and
// starts subprocesses to perform network operations.
func Login(storageDir string, password []byte, parameters string) (*Client, error) {
	p, err := params.GetNetworkParameters(parameters)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to login: %+v", err))
	}

	client, err := api.Login(storageDir, password, p)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to login: %+v", err))
	}
	return &Client{*client}, nil
}

// sets level of logging. All logs the set level and above will be displayed
// options are:
//	TRACE		- 0
//	DEBUG		- 1
//	INFO 		- 2
//	WARN		- 3
//	ERROR		- 4
//	CRITICAL	- 5
//	FATAL		- 6
// The default state without updates is: INFO
func LogLevel(level int) error {
	if level < 0 || level > 6 {
		return errors.New(fmt.Sprintf("log level is not valid: log level: %d", level))
	}

	threshold := jww.Threshold(level)
	jww.SetLogThreshold(threshold)
	jww.SetStdoutThreshold(threshold)

	switch threshold {
	case jww.LevelTrace:
		fallthrough
	case jww.LevelDebug:
		fallthrough
	case jww.LevelInfo:
		jww.INFO.Printf("Log level set to: %s", threshold)
	case jww.LevelWarn:
		jww.WARN.Printf("Log level set to: %s", threshold)
	case jww.LevelError:
		jww.ERROR.Printf("Log level set to: %s", threshold)
	case jww.LevelCritical:
		jww.CRITICAL.Printf("Log level set to: %s", threshold)
	case jww.LevelFatal:
		jww.FATAL.Printf("Log level set to: %s", threshold)
	}

	return nil
}

//Unmarshals a marshaled contact object, returns an error if it fails
func UnmarshalContact(b []byte) (*Contact, error) {
	c, err := contact.Unmarshal(b)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to Unmarshal "+
			"Contact: %+v", err))
	}
	return &Contact{c: &c}, nil
}

//Unmarshals a marshaled send report object, returns an error if it fails
func UnmarshalSendReport(b []byte) (*SendReport, error) {
	sr := &SendReport{}
	if err := json.Unmarshal(b, sr); err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to Unmarshal "+
			"Send Report: %+v", err))
	}
	return sr, nil
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
func (c *Client) StartNetworkFollower(clientError ClientError) error {
	errChan, err := c.api.StartNetworkFollower()
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to start the "+
			"network follower: %+v", err))
	}

	go func() {
		for report := range errChan {
			go clientError.Report(report.Source, report.Message, report.Trace)
		}
	}()
	return nil
}

// StopNetworkFollower stops the network follower if it is running.
// It returns errors if the Follower is in the wrong status to stop or if it
// fails to stop it.
// if the network follower is running and this fails, the client object will
// most likely be in an unrecoverable state and need to be trashed.
func (c *Client) StopNetworkFollower(timeoutMS int) error {
	timeout := time.Duration(timeoutMS) * time.Millisecond
	if err := c.api.StopNetworkFollower(timeout); err != nil {
		return errors.New(fmt.Sprintf("Failed to stop the "+
			"network follower: %+v", err))
	}
	return nil
}

// WaitForNewtwork will block until either the network is healthy or the
// passed timeout. It will return true if the network is healthy
func (c *Client) WaitForNetwork(timeoutMS int) bool {
	timeout := time.NewTimer(time.Duration(timeoutMS) * time.Millisecond)
	healthyChan := make(chan bool, 1)
	c.api.GetHealth().AddChannel(healthyChan)
	select{
	case <- healthyChan:
		return true
	case <-timeout.C:
		return false
	}
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
// Returns a ListenerUnregister interface which can be
//
// to register for any userID, pass in an id with length 0 or an id with
// all zeroes
//
// to register for any message type, pass in a message type of 0
//
// Message Types can be found in client/interfaces/message/type.go
// Make sure to not conflict with ANY default message types
func (c *Client) RegisterListener(uid []byte, msgType int,
	listener Listener) (*Unregister, error) {
	jww.INFO.Printf("RegisterListener(%v, %d)", uid,
		msgType)

	name := listener.Name()

	var u *id.ID
	if len(uid) == 0 {
		u = &id.ID{}
	} else {
		var err error
		u, err = id.Unmarshal(uid)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("Failed to "+
				"ResgisterListener: %+v", err))
		}
	}

	mt := message.Type(msgType)

	f := func(item message.Receive) {
		listener.Hear(&Message{r: item})
	}

	lid := c.api.GetSwitchboard().RegisterFunc(name, u, mt, f)

	return newListenerUnregister(lid, c.api.GetSwitchboard()), nil
}

// RegisterRoundEventsHandler registers a callback interface for round
// events.
// The rid is the round the event attaches to
// The timeoutMS is the number of milliseconds until the event fails, and the
// validStates are a list of states (one per byte) on which the event gets
// triggered
// States:
//  0x00 - PENDING (Never seen by client)
//  0x01 - PRECOMPUTING
//  0x02 - STANDBY
//  0x03 - QUEUED
//  0x04 - REALTIME
//  0x05 - COMPLETED
//  0x06 - FAILED
// These states are defined in elixxir/primitives/states/state.go
func (c *Client) RegisterRoundEventsHandler(rid int, cb RoundEventCallback,
	timeoutMS int, il *IntList) *Unregister {

	rcb := func(ri *mixmessages.RoundInfo, timedOut bool) {
		cb.EventCallback(int(ri.ID), int(ri.State), timedOut)
	}

	timeout := time.Duration(timeoutMS) * time.Millisecond

	vStates := make([]states.Round, len(il.lst))
	for i, s := range il.lst {
		vStates[i] = states.Round(s)
	}

	roundID := id.Round(rid)

	ec := c.api.GetRoundEvents().AddRoundEvent(roundID, rcb, timeout)

	return newRoundUnregister(roundID, ec, c.api.GetRoundEvents())
}

// RegisterMessageDeliveryCB allows the caller to get notified if the rounds a
// message was sent in successfully completed. Under the hood, this uses the same
// interface as RegisterRoundEventsHandler, but provides a convenient way to use
// the interface in its most common form, looking up the result of message
// retrieval
//
// The callbacks will return at timeoutMS if no state update occurs
//
// This function takes the marshaled send report to ensure a memory leak does
// not occur as a result of both sides of the bindings holding a reference to
// the same pointer.
func (c *Client) WaitForRoundCompletion(marshaledSendReport []byte,
	mdc MessageDeliveryCallback, timeoutMS int) error {

	sr, err := UnmarshalSendReport(marshaledSendReport)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to "+
			"WaitForRoundCompletion callback due to bad Send Report: %+v", err))
	}

	f := func(allRoundsSucceeded, timedOut bool, rounds map[id.Round]api.RoundResult) {
		results := make([]byte, len(sr.rl.list))

		for i, r := range sr.rl.list {
			if result, exists := rounds[r]; exists {
				results[i] = byte(result)
			}
		}

		mdc.EventCallback(sr.mid.Marshal(), allRoundsSucceeded, timedOut, results)
	}

	timeout := time.Duration(timeoutMS) * time.Millisecond

	return c.api.GetRoundResults(sr.rl.list, timeout, f)
}

// Returns a user object from which all information about the current user
// can be gleaned
func (c *Client) GetUser() *User {
	u := c.api.GetUser()
	return &User{u: &u}
}

// GetNodeRegistrationStatus returns a struct with the number of nodes the
// client is registered with and the number total.
func (c *Client) GetNodeRegistrationStatus() (*NodeRegistrationsStatus, error) {
	registered, inProgress, err := c.api.GetNodeRegistrationStatus()

	return &NodeRegistrationsStatus{registered, inProgress}, err
}

/*
// SearchWithHandler is a non-blocking search that also registers
// a callback interface for user disovery events.
func (c *Client) SearchWithHandler(data, separator string,
	searchTypes []byte, hdlr UserDiscoveryHandler) {
}


// RegisterAuthEventsHandler registers a callback interface for channel
// authentication events.
func (b *BindingsClient) RegisterAuthEventsHandler(hdlr AuthEventHandler) {
}

// Search accepts a "separator" separated list of search elements with
// an associated list of searchTypes. It returns a ContactList which
// allows you to iterate over the found contact objects.
func (b *BindingsClient) Search(data, separator string,
	searchTypes []byte) ContactList {
	return nil
}*/
