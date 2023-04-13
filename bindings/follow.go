////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"time"

	"github.com/pkg/errors"
	"gitlab.com/xx_network/primitives/netTime"
)

// StartNetworkFollower kicks off the tracking of the network. It starts long-
// running network threads and returns an object for checking state and
// stopping those threads.
//
// Call this when returning from sleep and close when going back to sleep.
//
// These threads may become a significant drain on battery when offline, ensure
// they are stopped if there is no internet access.
//
// Threads Started:
//   - Network Follower (/network/follow.go)
//     tracks the network events and hands them off to workers for handling.
//   - Historical Round Retrieval (/network/rounds/historical.go)
//     retrieves data about rounds that are too old to be stored by the client.
//   - Message Retrieval Worker Group (/network/rounds/retrieve.go)
//     requests all messages in a given round from the gateway of the last nodes.
//   - Message Handling Worker Group (/network/message/handle.go)
//     decrypts and partitions messages when signals via the Switchboard.
//   - Health Tracker (/network/health),
//     via the network instance, tracks the state of the network.
//   - Garbled Messages (/network/message/garbled.go)
//     can be signaled to check all recent messages that could be decoded. It
//     uses a message store on disk for persistence.
//   - Critical Messages (/network/message/critical.go)
//     ensures all protocol layer mandatory messages are sent. It uses a message
//     store on disk for persistence.
//   - KeyExchange Trigger (/keyExchange/trigger.go)
//     responds to sent rekeys and executes them.
//   - KeyExchange Confirm (/keyExchange/confirm.go)
//     responds to confirmations of successful rekey operations.
//   - Auth Callback (/auth/callback.go)
//     handles both auth confirm and requests.
func (c *Cmix) StartNetworkFollower(timeoutMS int) error {
	timeout := time.Duration(timeoutMS) * time.Millisecond
	return c.api.StartNetworkFollower(timeout)
}

// StopNetworkFollower stops the network follower if it is running. It returns
// an error if the follower is in the wrong state to stop or if it fails to stop
// it.
//
// If the network follower is running and this fails, the Cmix object will
// most likely be in an unrecoverable state and need to be trashed.
func (c *Cmix) StopNetworkFollower() error {
	if err := c.api.StopNetworkFollower(); err != nil {
		return errors.New(fmt.Sprintf("Failed to stop the "+
			"network follower: %+v", err))
	}
	return nil
}

// SetTrackNetworkPeriod allows changing the frequency that follower threads
// are started.
//
// Parameters:
//   - periodMS - The duration of the period, in milliseconds.
func (c *Cmix) SetTrackNetworkPeriod(periodMS int) {
	period := time.Duration(periodMS) * time.Millisecond
	c.api.SetTrackNetworkPeriod(period)
}

// WaitForNetwork will block until either the network is healthy or the passed
// timeout is reached. It will return true if the network is healthy.
func (c *Cmix) WaitForNetwork(timeoutMS int) bool {
	start := netTime.Now()
	timeout := time.Duration(timeoutMS) * time.Millisecond
	for netTime.Since(start) < timeout {
		if c.api.GetCmix().IsHealthy() {
			return true
		}
		time.Sleep(250 * time.Millisecond)
	}
	return false
}

// ReadyToSend determines if the network is ready to send messages on. It
// returns true if the network is healthy and if the client has registered with
// at least 70% of the nodes. Returns false otherwise.
func (c *Cmix) ReadyToSend() bool {
	// Check if the network is currently healthy
	if !c.api.GetCmix().IsHealthy() {
		return false
	}

	// If the network is healthy, then check the number of nodes that the client
	// is currently registered with
	numReg, total, err := c.api.GetNodeRegistrationStatus()
	if err != nil {
		jww.FATAL.Panicf("Failed to get node registration status: %+v", err)
	}

	return numReg >= total*7/10
}

// IsReadyInfo contains information on if the network is ready and how close it
// is to being ready.
//
// Example JSON:
//
//	{
//	  "IsReady": true,
//	  "HowClose": 0.534
//	}
type IsReadyInfo struct {
	IsReady  bool
	HowClose float64
}

// NetworkFollowerStatus gets the state of the network follower. It returns a
// status with the following values:
//
//	Stopped  - 0
//	Running  - 2000
//	Stopping - 3000
func (c *Cmix) NetworkFollowerStatus() int {
	return int(c.api.NetworkFollowerStatus())
}

// NodeRegistrationReport is the report structure which
// Cmix.GetNodeRegistrationStatus returns JSON marshalled.
type NodeRegistrationReport struct {
	NumberOfNodesRegistered int
	NumberOfNodes           int
}

// GetNodeRegistrationStatus returns the current state of node registration.
//
// Returns:
//   - []byte - A marshalled NodeRegistrationReport containing the number of
//     nodes the user is registered with and the number of nodes present in the
//     NDF.
//   - An error if it cannot get the node registration status. The most likely
//     cause is that the network is unhealthy.
func (c *Cmix) GetNodeRegistrationStatus() ([]byte, error) {
	numNodesRegistered, numNodes, err := c.api.GetNodeRegistrationStatus()
	if err != nil {
		return nil, err
	}

	nodeRegReport := NodeRegistrationReport{
		NumberOfNodesRegistered: numNodesRegistered,
		NumberOfNodes:           numNodes,
	}

	return json.Marshal(nodeRegReport)
}

// IsReady returns true if at least percentReady of node registrations has
// completed. If not all have completed, then it returns false and howClose will
// be a percent (0-1) of node registrations completed.
//
// Parameters:
//   - percentReady - The percentage of nodes required to be registered with to
//     be ready. This is a number between 0 and 1.
//
// Returns:
//   - JSON of [IsReadyInfo].
func (c *Cmix) IsReady(percentReady float64) ([]byte, error) {
	isReady, howClose := c.api.IsReady(percentReady)
	return json.Marshal(&IsReadyInfo{isReady, howClose})
}

// PauseNodeRegistrations stops all node registrations and returns a function to
// resume them.
//
// Parameters:
//   - timeoutMS - The timeout, in milliseconds, to wait when stopping threads
//     before failing.
func (c *Cmix) PauseNodeRegistrations(timeoutMS int) error {
	timeout := time.Duration(timeoutMS) * time.Millisecond
	return c.api.PauseNodeRegistrations(timeout)
}

// ChangeNumberOfNodeRegistrations changes the number of parallel node
// registrations up to the initialized maximum.
//
// Parameters:
//   - toRun - The number of parallel node registrations.
//   - timeoutMS - The timeout, in milliseconds, to wait when changing node
//     registrations before failing.
func (c *Cmix) ChangeNumberOfNodeRegistrations(toRun, timeoutMS int) error {
	timeout := time.Duration(timeoutMS) * time.Millisecond
	return c.api.ChangeNumberOfNodeRegistrations(toRun, timeout)
}

// HasRunningProcessies checks if any background threads are running and returns
// true if one or more are.
//
// This is meant to be used when NetworkFollowerStatus returns xxdk.Stopping.
// Due to the handling of comms on iOS, where the OS can block indefinitely, it
// may not enter the stopped state appropriately. This can be used instead.
func (c *Cmix) HasRunningProcessies() bool {
	return c.api.HasRunningProcessies()
}

// IsHealthy returns true if the network is read to be in a healthy state where
// messages can be sent.
func (c *Cmix) IsHealthy() bool {
	return c.api.GetCmix().IsHealthy()
}

// GetRunningProcesses returns the names of all running processes at the time
// of this call. Note that this list may change and is subject to race
// conditions if multiple threads are in the process of starting or stopping.
//
// Returns:
//   - []byte - A JSON marshalled list of all running processes.
//
// JSON Example:
//
//	{
//	  "FileTransfer{BatchBuilderThread, FilePartSendingThread#0, FilePartSendingThread#1, FilePartSendingThread#2, FilePartSendingThread#3}",
//	  "MessageReception Worker 0"
//	}
func (c *Cmix) GetRunningProcesses() ([]byte, error) {
	return json.Marshal(c.api.GetRunningProcesses())
}

// NetworkHealthCallback contains a callback that is used to receive
// notification if network health changes.
type NetworkHealthCallback interface {
	Callback(bool)
}

// AddHealthCallback adds a callback that gets called whenever the network
// health changes. Returns a registration ID that can be used to unregister.
func (c *Cmix) AddHealthCallback(nhc NetworkHealthCallback) int64 {
	return int64(c.api.GetCmix().AddHealthCallback(nhc.Callback))
}

// RemoveHealthCallback removes a health callback using its registration ID.
func (c *Cmix) RemoveHealthCallback(funcID int64) {
	c.api.GetCmix().RemoveHealthCallback(uint64(funcID))
}

type ClientError interface {
	Report(source, message, trace string)
}

// RegisterClientErrorCallback registers the callback to handle errors from the
// long-running threads controlled by StartNetworkFollower and
// StopNetworkFollower.
func (c *Cmix) RegisterClientErrorCallback(clientError ClientError) {
	errChan := c.api.GetErrorsChannel()
	go func() {
		for report := range errChan {
			go clientError.Report(report.Source, report.Message, report.Trace)
		}
	}()
}

// TrackServicesCallback is the callback for [Cmix.TrackServices] which passes a
// JSON-marshalled list of uncompressed backend services. If there was an error
// retrieving or marshalling the service list, there is an error for the second
// parameter which will be non-null.
//
// Parameters:
//   - marshalData - JSON marshalled bytes of [message.ServiceList], which is an
//     array of [id.ID] and [message.Service].
//   - err - JSON unmarshalling error
//
// Example JSON:
//
//	[
//	  {
//	    "Id": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD", // bytes of id.ID encoded as base64 string
//	    "Services": [
//	      {
//	        "Identifier": "AQID",                             // bytes encoded as base64 string
//	        "Tag": "TestTag 1",                               // string
//	        "Metadata": "BAUG"                                // bytes encoded as base64 string
//	      }
//	    ]
//	  },
//	  {
//	    "Id": "AAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD",
//	    "Services": [
//	      {
//	        "Identifier": "AQID",
//	        "Tag": "TestTag 2",
//	        "Metadata": "BAUG"
//	      }
//	    ]
//	  },
//	]
type TrackServicesCallback interface {
	Callback(marshalData []byte, err error)
}

// TrackCompressedServicesCallback is the callback for [Cmix.TrackServices]
// which passes a JSON-marshalled list of compressed backend services. If
// there was an error retrieving or marshalling the service list, there is an
// error for the second parameter which will be non-null.
//
// Parameters:
//   - marshalData - JSON marshalled bytes of [message.CompressedServiceList],
//     which is an array of [id.ID] and [message.CompressedService].
//   - err - JSON unmarshalling error
//
// Example JSON:
//
//	{
//    "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD": [
//      {
//        "Identifier": null,
//        "Tags": [
//          "test"
//        ],
//        "Metadata": null
//      }
//    ],
//    "AAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD": [
//      {
//        "Identifier": null,
//        "Tags": [
//          "test"
//        ],
//        "Metadata": null
//      }
//    ],
//    "AAAAAAAAAAIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD": [
//      {
//        "Identifier": null,
//        "Tags": [
//         	"test"
//        ],
//        "Metadata": null
//      }
//    ]
//  }
type TrackCompressedServicesCallback interface {
	Callback(marshalData []byte, err error)
}

// TrackServicesWithIdentity will return via a callback the list of services the
// backend keeps track of for the provided identity. This may be passed into
// other bindings call which may need context on the available services for this
// single identity. This will only return services for the given identity.
//
// Parameters:
//   - e2eID - e2e object ID in the tracker.
//   - cb - A TrackServicesCallback, which will be passed the marshalled
//     message.ServiceList.
func (c *Cmix) TrackServicesWithIdentity(e2eId int,
	cb TrackServicesCallback, compressedCb TrackCompressedServicesCallback) error {
	// Retrieve the user from the tracker
	user, err := e2eTrackerSingleton.get(e2eId)
	if err != nil {
		return err
	}

	receptionId := user.api.GetReceptionIdentity().ID
	c.api.GetCmix().TrackServices(func(list message.ServiceList, compressedList message.CompressedServiceList) {
		// Pass along normal services
		res := make(message.ServiceList)
		res[*receptionId] = list[*receptionId]
		cb.Callback(json.Marshal(res))

		// Pass along compressed services
		compressedRes := make(message.CompressedServiceList)
		compressedRes[*receptionId] = compressedList[*receptionId]
		compressedCb.Callback(json.Marshal(compressedRes))
	})

	return nil
}

// TrackServices will return via a callback the list of services the
// backend keeps track of, which is formally referred to as a
// [message.ServiceList]. This may be passed into other bindings call which
// may need context on the available services for this client. This will
// provide services for all identities that the client tracks.
//
// Parameters:
//   - cb - A TrackServicesCallback, which will be passed the marshalled
//     message.ServiceList.
func (c *Cmix) TrackServices(cb TrackServicesCallback) {
	c.api.GetCmix().TrackServices(func(list message.ServiceList, compressedList message.CompressedServiceList) {
		cb.Callback(json.Marshal(list))
	})
}
