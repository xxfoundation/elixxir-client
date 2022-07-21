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

// Secure (authenticated) connection server path.
func secureConnServer(net *xxdk.Cmix, identity xxdk.ReceptionIdentity,
	e2eParams xxdk.E2EParams) {

	// Handle incoming connections---------------------------------------------
	connChan := make(chan connect.Connection, 1)
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

	// Wait until connected or crash on timeout
	connected := make(chan bool, 10)
	connectServer.Messenger.GetCmix().AddHealthCallback(
		func(isConnected bool) {
			connected <- isConnected
		})
	cmdUtils.WaitUntilConnected(connected)

	// Wait for connection establishment----------------------------------------
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
	serverTimeout := viper.GetDuration(ConnectionServerTimeoutFlag)
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

// Insecure (unauthenticated) connection server path.
func insecureConnServer(net *xxdk.Cmix, identity xxdk.ReceptionIdentity,
	e2eParams xxdk.E2EParams) {

	// Handle incoming connections---------------------------------------------
	connChan := make(chan connect.Connection, 1)
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

	// Wait until connected or crash on timeout
	connected := make(chan bool, 10)
	connectServer.Messenger.GetCmix().AddHealthCallback(
		func(isConnected bool) {
			connected <- isConnected
		})
	cmdUtils.WaitUntilConnected(connected)

	// Wait for connection establishment----------------------------------------
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
	if viper.GetDuration(ConnectionServerTimeoutFlag) != 0 {
		timer := time.NewTimer(viper.GetDuration(ConnectionServerTimeoutFlag))
		select {
		case <-timer.C:
			fmt.Println("Shutting down connection server")
			timer.Stop()
			return
		}
	}

	// Keep app running to receive messages------------------------------------
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
