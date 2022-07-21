package cmd

import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	cmdUtils "gitlab.com/elixxir/client/cmdUtils"
	"gitlab.com/elixxir/client/connect"
	"gitlab.com/elixxir/client/xxdk"
	"time"
)

// Secure (authenticated) connection client path
func secureConnClient(cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams) {
	// Load client ------------------------------------------------------------------
	messenger := cmdUtils.InitE2e(cmixParams, e2eParams, xxdk.DefaultAuthCallbacks{})

	// Start network threads---------------------------------------------------------

	// Set networkFollowerTimeout to a value of your choice (seconds)
	networkFollowerTimeout := 5 * time.Second
	err := messenger.StartNetworkFollower(networkFollowerTimeout)
	if err != nil {
		jww.FATAL.Panicf("Failed to start network follower: %+v", err)
	}

	// Wait until connected or crash on timeout
	connected := make(chan bool, 10)
	messenger.GetCmix().AddHealthCallback(
		func(isConnected bool) {
			connected <- isConnected
		})
	cmdUtils.WaitUntilConnected(connected)

	// Connect with the server-------------------------------------------------
	contactPath := viper.GetString(ConnectionFlag)
	serverContact := cmdUtils.GetContactFromFile(contactPath)
	fmt.Println("Sending connection request")

	// Establish connection with partner
	conn, err := connect.ConnectWithAuthentication(serverContact, messenger,
		e2eParams)
	if err != nil {
		jww.FATAL.Panicf("[CONN] Failed to build connection with %s: %v",
			serverContact.ID, err)
	}

	jww.INFO.Printf("[CONN] Established authenticated connection with %s",
		conn.GetPartner().PartnerId())
	fmt.Println("Established authenticated connection with server.")

	miscConnectionFunctions(messenger, conn)

}

// Insecure (unauthenticated) connection client path
func insecureConnClient(cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams) {

	// Load client ------------------------------------------------------------------
	messenger := cmdUtils.InitE2e(cmixParams, e2eParams, xxdk.DefaultAuthCallbacks{})

	// Start network threads---------------------------------------------------------

	// Set networkFollowerTimeout to a value of your choice (seconds)
	networkFollowerTimeout := 5 * time.Second
	err := messenger.StartNetworkFollower(networkFollowerTimeout)
	if err != nil {
		jww.FATAL.Panicf("Failed to start network follower: %+v", err)
	}

	// Set up a wait for the network to be connected
	waitUntilConnected := func(connected chan bool) {
		waitTimeout := 30 * time.Second
		timeoutTimer := time.NewTimer(waitTimeout)
		isConnected := false
		// Wait until we connect or panic if we cannot before the timeout
		for !isConnected {
			select {
			case isConnected = <-connected:
				jww.INFO.Printf("Network Status: %v", isConnected)
				break
			case <-timeoutTimer.C:
				jww.FATAL.Panicf("Timeout on starting network follower")
			}
		}
	}

	// Create a tracker channel to be notified of network changes
	connected := make(chan bool, 10)
	// Provide a callback that will be signalled when network
	// health status changes
	messenger.GetCmix().AddHealthCallback(
		func(isConnected bool) {
			connected <- isConnected
		})
	// Wait until connected or crash on timeout
	waitUntilConnected(connected)

	// Connect with the server-------------------------------------------------
	contactPath := viper.GetString(ConnectionFlag)
	serverContact := cmdUtils.GetContactFromFile(contactPath)
	fmt.Println("Sending connection request")
	jww.INFO.Printf("[CONN] Sending connection request to %s",
		serverContact.ID)

	// Establish connection with partner
	handler, err := connect.Connect(serverContact, messenger,
		e2eParams)
	if err != nil {
		jww.FATAL.Panicf("[CONN] Failed to build connection with %s: %v",
			serverContact.ID, err)

	}

	fmt.Println("Established connection with server")
	jww.INFO.Printf("[CONN] Established connection with %s", handler.GetPartner().PartnerId())

	miscConnectionFunctions(messenger, handler)
}
