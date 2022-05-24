package bindings

import (
	"gitlab.com/xx_network/primitives/netTime"
	"time"
)

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

// WaitForNewtwork will block until either the network is healthy or the
// passed timeout. It will return true if the network is healthy
func (c *Client) WaitForNetwork(timeoutMS int) bool {
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

// IsNetworkHealthy returns true if the network is read to be in a healthy state where
// messages can be sent
func (c *Client) IsNetworkHealthy() bool {
	return c.api.GetCmix().IsHealthy()
}

// A callback when which is used to receive notification if network health
// changes
type NetworkHealthCallback interface {
	Callback(bool)
}

// RegisterNetworkHealthCB registers the network health callback to be called
// any time the network health changes. Returns a unique ID that can be used to
// unregister the network health callback.
func (c *Client) RegisterNetworkHealthCB(nhc NetworkHealthCallback) int64 {
	return int64(c.api.GetCmix().AddHealthCallback(nhc.Callback))
}

func (c *Client) UnregisterNetworkHealthCB(funcID int64) {
	c.api.GetCmix().RemoveHealthCallback(uint64(funcID))
}

type ClientError interface {
	Report(source, message, trace string)
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
