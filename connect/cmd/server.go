package cmd

import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/catalog"
	cmdUtils "gitlab.com/elixxir/client/cmdUtils"
	"gitlab.com/elixxir/client/connect"
	"gitlab.com/elixxir/client/xxdk"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Secure (authenticated) connection server path
func secureConnServer(forceLegacy bool, statePass []byte, statePath, regCode string,
	cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams) {
	connChan := make(chan connect.Connection, 1)

	// Load client state and identity------------------------------------------
	net := cmdUtils.LoadOrInitCmix(statePass, statePath, regCode, cmixParams)
	identity := cmdUtils.LoadOrInitReceptionIdentity(forceLegacy, net)

	// Save contact file-------------------------------------------------------
	cmdUtils.WriteContact(identity.GetContact())

	// Handle incoming connections---------------------------------------------
	authCb := connect.AuthenticatedCallback(
		func(connection connect.AuthenticatedConnection) {
			partnerId := connection.GetPartner().PartnerId()
			jww.INFO.Printf("[CONN] Received authenticated connection from %s", partnerId)
			fmt.Println("Established authenticated connection with client")

			_, err := connection.RegisterListener(catalog.XxMessage, listener{"AuthServer"})
			if err != nil {
				jww.FATAL.Panicf("Failed to register listener for client message!")
			}

			connChan <- connection
		})

	// Start connection server-------------------------------------------------
	connectionParam := connect.DefaultConnectionListParams()
	connectServer, err := connect.StartAuthenticatedServer(identity,
		authCb, net, e2eParams, connectionParam)
	if err != nil {
		jww.FATAL.Panicf("Failed to start authenticated "+
			"connection server: %v", err)
	}

	fmt.Println("Established connection server, begin listening...")
	jww.INFO.Printf("[CONN] Established connection server, begin listening...")

	// Start network threads---------------------------------------------------
	networkFollowerTimeout := 5 * time.Second
	err = connectServer.Messenger.StartNetworkFollower(networkFollowerTimeout)
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
	// Provide a callback that will be signalled when network health
	// status changes
	connectServer.Messenger.GetCmix().AddHealthCallback(
		func(isConnected bool) {
			connected <- isConnected
		})
	// Wait until connected or crash on timeout
	waitUntilConnected(connected)

	// Wait for connection establishment----------------------------------------

	// Wait for connection to be established
	connectionTimeout := time.NewTimer(240 * time.Second)
	select {
	case conn := <-connChan:
		// Perform functionality shared by client & server
		miscConnectionFunctions(connectServer.Messenger, conn)

	case <-connectionTimeout.C:
		connectionTimeout.Stop()
		jww.FATAL.Panicf("[CONN] Failed to establish connection within " +
			"default time period, closing process")
	}

	// Keep server running to receive messages------------------------------------
	serverTimeout := viper.GetDuration(connectionServerTimeoutFlag)
	if serverTimeout != 0 {
		timer := time.NewTimer(serverTimeout)
		select {
		case <-timer.C:
			fmt.Println("Shutting down connection server")
			timer.Stop()
			return
		}
	}

	// Keep app running to receive messages------------------------------------

	// Wait until the user terminates the program
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	err = connectServer.Messenger.StopNetworkFollower()
	if err != nil {
		jww.ERROR.Printf("Failed to stop network follower: %+v", err)
	} else {
		jww.INFO.Printf("Stopped network follower.")
	}

	os.Exit(0)

}

// Insecure (unauthenticated) connection server path
func insecureConnServer(forceLegacy bool, statePass []byte, statePath, regCode string,
	cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams) {

	connChan := make(chan connect.Connection, 1)

	// Load client state and identity------------------------------------------
	net := cmdUtils.LoadOrInitCmix(statePass, statePath, regCode, cmixParams)
	identity := cmdUtils.LoadOrInitReceptionIdentity(forceLegacy, net)

	// Save contact file-------------------------------------------------------
	cmdUtils.WriteContact(identity.GetContact())

	// Handle incoming connections---------------------------------------------
	cb := connect.Callback(func(connection connect.Connection) {
		partnerId := connection.GetPartner().PartnerId()
		jww.INFO.Printf("[CONN] Received connection request from %s", partnerId)
		fmt.Println("Established connection with client")

		_, err := connection.RegisterListener(catalog.XxMessage, listener{"ConnectionServer"})
		if err != nil {
			jww.FATAL.Panicf("Failed to register listener for client message!")
		}

		connChan <- connection
	})

	// Start connection server-------------------------------------------------
	connectionParam := connect.DefaultConnectionListParams()
	connectServer, err := connect.StartServer(identity,
		cb, net, e2eParams, connectionParam)
	if err != nil {
		jww.FATAL.Panicf("[CONN] Failed to start connection server: %v", err)
	}

	fmt.Println("Established connection server, begin listening...")
	jww.INFO.Printf("[CONN] Established connection server, begin listening...")

	// Start network threads---------------------------------------------------
	networkFollowerTimeout := 5 * time.Second
	err = connectServer.Messenger.StartNetworkFollower(networkFollowerTimeout)
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
	// Provide a callback that will be signalled when network health
	// status changes
	connectServer.Messenger.GetCmix().AddHealthCallback(
		func(isConnected bool) {
			connected <- isConnected
		})
	// Wait until connected or crash on timeout
	waitUntilConnected(connected)

	// Wait for connection establishment----------------------------------------

	// Wait for connection to be established
	connectionTimeout := time.NewTimer(240 * time.Second)
	select {
	case conn := <-connChan:
		// Perform functionality shared by client & server
		miscConnectionFunctions(connectServer.Messenger, conn)

	case <-connectionTimeout.C:
		connectionTimeout.Stop()
		jww.FATAL.Panicf("[CONN] Failed to establish connection within " +
			"default time period, closing process")
	}

	// Keep server running to receive messages------------------------------------
	if viper.GetDuration(connectionServerTimeoutFlag) != 0 {
		timer := time.NewTimer(viper.GetDuration(connectionServerTimeoutFlag))
		select {
		case <-timer.C:
			fmt.Println("Shutting down connection server")
			timer.Stop()
			return
		}
	}
	// Keep app running to receive messages------------------------------------

	// Wait until the user terminates the program
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	err = connectServer.Messenger.StopNetworkFollower()
	if err != nil {
		jww.ERROR.Printf("Failed to stop network follower: %+v", err)
	} else {
		jww.INFO.Printf("Stopped network follower.")
	}

	os.Exit(0)

}
