///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/single"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"google.golang.org/grpc/grpclog"
)

var extantClient bool = false
var loginMux sync.Mutex

var clientSingleton *Client

// sets the log level
func init() {
	jww.SetLogThreshold(jww.LevelInfo)
	jww.SetStdoutThreshold(jww.LevelInfo)
}

// BindingsClient wraps the api.Client, implementing additional functions
// to support the gomobile Client interface
type Client struct {
	api       api.Client
	single    *single.Manager
	singleMux sync.Mutex
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

type BackupReport struct {
	RestoredContacts []*id.ID
	Params           string
}

// NewClientFromBackup constructs a new Client from an encrypted backup. The backup
// is decrypted using the backupPassphrase. On success a successful client creation,
// the function will return a JSON encoded list of the E2E partners
// contained in the backup and a json-encoded string of the parameters stored in the backup
func NewClientFromBackup(ndfJSON, storageDir string, sessionPassword,
	backupPassphrase, backupFileContents []byte) ([]byte, error) {
	backupPartnerIds, jsonParams, err := api.NewClientFromBackup(ndfJSON, storageDir,
		sessionPassword, backupPassphrase, backupFileContents)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to create new "+
			"client from backup: %+v", err))
	}

	report := BackupReport{
		RestoredContacts: backupPartnerIds,
		Params:           jsonParams,
	}

	return json.Marshal(report)
}

// Login will load an existing client from the storageDir
// using the password. This will fail if the client doesn't exist or
// the password is incorrect.
// The password is passed as a byte array so that it can be cleared from
// memory and stored as securely as possible using the memguard library.
// Login does not block on network connection, and instead loads and
// starts subprocesses to perform network operations.
func Login(storageDir string, password []byte, parameters string) (*Client, error) {
	loginMux.Lock()
	defer loginMux.Unlock()

	if extantClient {
		return nil, errors.New("cannot login when another session " +
			"already exists")
	}
	// check if a client is already logged in, refuse to login if one is
	p, err := params.GetNetworkParameters(parameters)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to login: %+v", err))
	}

	client, err := api.Login(storageDir, password, p)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to login: %+v", err))
	}
	extantClient = true
	clientSingleton := &Client{api: *client}

	return clientSingleton, nil
}

// returns a previously created client. IF be used if the garbage collector
// removes the client instance on the app side.  Is NOT thread safe relative to
// login, newClient, or newPrecannedClient
func GetClientSingleton() *Client {
	return clientSingleton
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

//RegisterLogWriter registers a callback on which logs are written.
func RegisterLogWriter(writer LogWriter) {
	jww.SetLogOutput(&writerAdapter{lw: writer})
}

// EnableGrpcLogs sets GRPC trace logging
func EnableGrpcLogs(writer LogWriter) {
	logger := &writerAdapter{lw: writer}
	grpclog.SetLoggerV2(grpclog.NewLoggerV2WithVerbosity(
		logger, logger, logger, 99))
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
	return sr, sr.Unmarshal(b)
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
//		Requests all messages in a given round from the gateway of the last nodes
//	 - Message Handling Worker Group (/network/message/handle.go)
//		Decrypts and partitions messages when signals via the Switchboard
//	 - health Tracker (/network/health)
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
func (c *Client) StartNetworkFollower(timeoutMS int) error {
	timeout := time.Duration(timeoutMS) * time.Millisecond
	return c.api.StartNetworkFollower(timeout)
}

// RegisterClientErrorCallback registers the callback to handle errors from the
// long running threads controlled by StartNetworkFollower and StopNetworkFollower
func (c *Client) RegisterClientErrorCallback(clientError ClientError) {
	errChan := c.api.GetErrorsChannel()
	go func() {
		for report := range errChan {
			go clientError.Report(report.Source, report.Message, report.Trace)
		}
	}()
}

// StopNetworkFollower stops the network follower if it is running.
// It returns errors if the Follower is in the wrong status to stop or if it
// fails to stop it.
// if the network follower is running and this fails, the client object will
// most likely be in an unrecoverable state and need to be trashed.
func (c *Client) StopNetworkFollower() error {
	if err := c.api.StopNetworkFollower(); err != nil {
		return errors.New(fmt.Sprintf("Failed to stop the "+
			"network follower: %+v", err))
	}
	return nil
}

// WaitForNewtwork will block until either the network is healthy or the
// passed timeout. It will return true if the network is healthy
func (c *Client) WaitForNetwork(timeoutMS int) bool {
	start := netTime.Now()
	timeout := time.Duration(timeoutMS) * time.Millisecond
	for netTime.Since(start) < timeout {
		if c.api.GetHealth().IsHealthy() {
			return true
		}
		time.Sleep(250 * time.Millisecond)
	}
	return false
}

// Gets the state of the network follower. Returns:
// Stopped 	- 0
// Starting - 1000
// Running	- 2000
// Stopping	- 3000
func (c *Client) NetworkFollowerStatus() int {
	return int(c.api.NetworkFollowerStatus())
}

// HasRunningProcessies checks if any background threads are running.
// returns true if none are running. This is meant to be
// used when NetworkFollowerStatus() returns Stopping.
// Due to the handling of comms on iOS, where the OS can
// block indefiently, it may not enter the stopped
// state apropreatly. This can be used instead.
func (c *Client) HasRunningProcessies() bool {
	return c.api.HasRunningProcessies()
}

// returns true if the network is read to be in a healthy state where
// messages can be sent
func (c *Client) IsNetworkHealthy() bool {
	return c.api.GetHealth().IsHealthy()
}

// RegisterNetworkHealthCB registers the network health callback to be called
// any time the network health changes. Returns a unique ID that can be used to
// unregister the network health callback.
func (c *Client) RegisterNetworkHealthCB(nhc NetworkHealthCallback) int64 {
	return int64(c.api.GetHealth().AddFunc(nhc.Callback))
}

func (c *Client) UnregisterNetworkHealthCB(funcID int64) {
	c.api.GetHealth().RemoveFunc(uint64(funcID))
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

// WaitForRoundCompletion allows the caller to get notified if a round
// has completed (or failed). Under the hood, this uses an API which uses the internal
// round data, network historical round lookup, and waiting on network events
// to determine what has (or will) occur.
//
// The callbacks will return at timeoutMS if no state update occurs
func (c *Client) WaitForRoundCompletion(roundID int,
	rec RoundCompletionCallback, timeoutMS int) error {

	f := func(allRoundsSucceeded, timedOut bool, rounds map[id.Round]api.RoundResult) {
		rec.EventCallback(roundID, allRoundsSucceeded, timedOut)
	}

	timeout := time.Duration(timeoutMS) * time.Millisecond

	return c.api.GetRoundResults([]id.Round{id.Round(roundID)}, timeout, f)
}

// WaitForMessageDelivery allows the caller to get notified if the rounds a
// message was sent in successfully completed. Under the hood, this uses an API
// which uses the internal round data, network historical round lookup, and
// waiting on network events to determine what has (or will) occur.
//
// The callbacks will return at timeoutMS if no state update occurs
//
// This function takes the marshaled send report to ensure a memory leak does
// not occur as a result of both sides of the bindings holding a reference to
// the same pointer.
func (c *Client) WaitForMessageDelivery(marshaledSendReport []byte,
	mdc MessageDeliveryCallback, timeoutMS int) error {
	jww.INFO.Printf("WaitForMessageDelivery(%v, _, %v)",
		marshaledSendReport, timeoutMS)
	sr, err := UnmarshalSendReport(marshaledSendReport)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to "+
			"WaitForMessageDelivery callback due to bad Send Report: %+v", err))
	}

	if sr == nil || sr.rl == nil || len(sr.rl.list) == 0 {
		return errors.New(fmt.Sprintf("Failed to "+
			"WaitForMessageDelivery callback due to invalid Send Report "+
			"unmarshal: %s", string(marshaledSendReport)))
	}

	f := func(allRoundsSucceeded, timedOut bool, rounds map[id.Round]api.RoundResult) {
		results := make([]byte, len(sr.rl.list))
		jww.INFO.Printf("Processing WaitForMessageDelivery report "+
			"for %v, success: %v, timedout: %v", sr.mid, allRoundsSucceeded,
			timedOut)
		for i, r := range sr.rl.list {
			if result, exists := rounds[r]; exists {
				results[i] = byte(result)
			}
		}

		mdc.EventCallback(sr.mid.Marshal(), allRoundsSucceeded, timedOut, results)
	}

	timeout := time.Duration(timeoutMS) * time.Millisecond

	err = c.api.GetRoundResults(sr.rl.list, timeout, f)

	return err
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
	registered, total, err := c.api.GetNodeRegistrationStatus()

	return &NodeRegistrationsStatus{registered, total}, err
}

// DeleteRequest will delete a request, agnostic of request type
// for the given partner ID. If no request exists for this
// partner ID an error will be returned.
func (c *Client) DeleteRequest(requesterUserId []byte) error {
	requesterId, err := id.Unmarshal(requesterUserId)
	if err != nil {
		return err
	}

	jww.DEBUG.Printf("Deleting request for partner ID: %s", requesterId)
	return c.api.DeleteRequest(requesterId)
}

// DeleteAllRequests clears all requests from Client's auth storage.
func (c *Client) DeleteAllRequests() error {
	return c.api.DeleteAllRequests()
}

// DeleteSentRequests clears sent requests from Client's auth storage.
func (c *Client) DeleteSentRequests() error {
	return c.api.DeleteSentRequests()
}

// DeleteReceiveRequests clears receive requests from Client's auth storage.
func (c *Client) DeleteReceiveRequests() error {
	return c.api.DeleteReceiveRequests()
}

// DeleteContact is a function which removes a contact from Client's storage
func (c *Client) DeleteContact(b []byte) error {
	contactObj, err := UnmarshalContact(b)
	if err != nil {
		return err
	}
	return c.api.DeleteContact(contactObj.c.ID)
}

// SetProxiedBins updates the host pool filter that filters out gateways that
// are not in one of the specified bins. The provided bins should be CSV.
func (c *Client) SetProxiedBins(binStringsCSV string) error {
	// Convert CSV to slice of strings
	all, err := csv.NewReader(strings.NewReader(binStringsCSV)).ReadAll()
	if err != nil {
		return err
	}

	binStrings := make([]string, 0, len(all[0]))
	for _, a := range all {
		binStrings = append(binStrings, a...)
	}

	return c.api.SetProxiedBins(binStrings)
}

// GetPreferredBins returns the geographic bin or bins that the provided two
// character country code is a part of. The bins are returned as CSV.
func (c *Client) GetPreferredBins(countryCode string) (string, error) {
	bins, err := c.api.GetPreferredBins(countryCode)
	if err != nil {
		return "", err
	}

	// Convert the slice of bins to CSV
	buff := bytes.NewBuffer(nil)
	csvWriter := csv.NewWriter(buff)
	err = csvWriter.Write(bins)
	if err != nil {
		return "", err
	}
	csvWriter.Flush()

	return buff.String(), nil
}

// GetRateLimitParams retrieves the rate limiting parameters.
func (c *Client) GetRateLimitParams() (uint32, uint32, int64) {
	return c.api.GetRateLimitParams()
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

// getSingle is a function which returns the single mananger if it
// exists or creates a new one, checking appropriate constraints
// (that the network follower is running) if it needs to make one
func (c *Client) getSingle() (*single.Manager, error) {
	c.singleMux.Lock()
	defer c.singleMux.Unlock()
	if c.single == nil {
		apiClient := &c.api
		c.single = single.NewManager(apiClient)
		err := apiClient.AddService(c.single.StartProcesses)
		if err != nil {
			return nil, err
		}
	}

	return c.single, nil
}

// GetInternalClient returns a reference to the client api. This is for internal
// use only and should not be called by bindings clients.
func (c *Client) GetInternalClient() api.Client {
	return c.api
}

func WrapAPIClient(c *api.Client) *Client {
	return &Client{api: *c}
}

// DumpStack returns a string with the stack trace of every running thread.
func DumpStack() (string, error) {
	buf := new(bytes.Buffer)
	err := pprof.Lookup("goroutine").WriteTo(buf, 2)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
