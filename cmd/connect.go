////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/connect"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/client/xxdk"
	"io/fs"
	"io/ioutil"
	"os"
	"time"
)

// connectionCmd handles the operation of connection operations within the CLI.
var connectionCmd = &cobra.Command{
	Use:   "connection",
	Short: "Runs clients and servers in the connections paradigm.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		logLevel := viper.GetUint(logLevelFlag)
		initLog(logLevel, viper.GetString(logFlag))
		jww.INFO.Printf(Version())

		statePass := parsePassword(viper.GetString(passwordFlag))
		statePath := viper.GetString(sessionFlag)
		regCode := viper.GetString(regCodeFlag)
		cmixParams, e2eParams := initParams()

		if viper.GetBool(connectionStartServerFlag) {
			if viper.GetBool(connectionAuthenticatedFlag) {
				secureConnServer(statePath, regCode, statePass,
					cmixParams, e2eParams)
			} else {
				insecureConnServer(statePath, regCode, statePass,
					cmixParams, e2eParams)
			}
		} else {
			if viper.GetBool(connectionAuthenticatedFlag) {
				secureConnClient(statePath, regCode, statePass,
					cmixParams, e2eParams)
			} else {
				insecureConnClient(statePath, regCode, statePass,
					cmixParams, e2eParams)
			}

		}

	},
}

////////////////////////////////////////////////////////////////////////////////////////////
// Connection Server Logic
////////////////////////////////////////////////////////////////////////////////////////////

// Secure (authenticated) connection server path
func secureConnServer(statePath, regCode string, statePass []byte,
	cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams) {
	connChan := make(chan connect.Connection, 1)

	// Load client state and identity------------------------------------------
	baseClient, identity := initializeBasicConnectionClient(statePath, regCode,
		statePass, cmixParams)

	// Save contact file-------------------------------------------------------
	writeContact(identity.GetContact())

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
		authCb, baseClient, e2eParams, connectionParam)
	if err != nil {
		jww.FATAL.Panicf("Failed to start authenticated "+
			"connection server: %v", err)
	}

	fmt.Println("Established connection server, begin listening...")
	jww.INFO.Printf("[CONN] Established connection server, begin listening...")

	// Start network threads---------------------------------------------------
	networkFollowerTimeout := 5 * time.Second
	err = connectServer.E2e.StartNetworkFollower(networkFollowerTimeout)
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
	connectServer.E2e.GetCmix().AddHealthCallback(
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
		miscConnectionFunctions(connectServer.E2e, conn)

	case <-connectionTimeout.C:
		connectionTimeout.Stop()
		jww.FATAL.Panicf("[CONN] Failed to establish connection within " +
			"default time period, closing process")
	}

	// Keep server running to receive messages------------------------------------
	serverTimeout := viper.GetDuration(connectionServerTimeoutFlag)
	if viper.GetDuration(connectionServerTimeoutFlag) != 0 {
		timer := time.NewTimer(serverTimeout)
		select {
		case <-timer.C:
			fmt.Println("Shutting down connection server")
			timer.Stop()
			return
		}
	}

	// If timeout is not specified, leave as long-running thread
	select {}

}

// Insecure (unauthenticated) connection server path
func insecureConnServer(statePath, regCode string, statePass []byte,
	cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams) {

	connChan := make(chan connect.Connection, 1)

	// Load client state and identity------------------------------------------
	baseClient, identity := initializeBasicConnectionClient(statePath, regCode,
		statePass, cmixParams)

	// Save contact file-------------------------------------------------------
	writeContact(identity.GetContact())

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
		cb, baseClient, e2eParams, connectionParam)
	if err != nil {
		jww.FATAL.Panicf("[CONN] Failed to start connection server: %v", err)
	}

	fmt.Println("Established connection server, begin listening...")
	jww.INFO.Printf("[CONN] Established connection server, begin listening...")

	// Start network threads---------------------------------------------------
	networkFollowerTimeout := 5 * time.Second
	err = connectServer.E2e.StartNetworkFollower(networkFollowerTimeout)
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
	connectServer.E2e.GetCmix().AddHealthCallback(
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
		miscConnectionFunctions(connectServer.E2e, conn)

	case <-connectionTimeout.C:
		connectionTimeout.Stop()
		jww.FATAL.Panicf("[CONN] Failed to establish connection within " +
			"default time period, closing process")
	}

	// Keep server running to receive messages------------------------------------
	serverTimeout := viper.GetDuration(connectionServerTimeoutFlag)
	if viper.GetDuration(connectionServerTimeoutFlag) != 0 {
		timer := time.NewTimer(serverTimeout)
		select {
		case <-timer.C:
			fmt.Println("Shutting down connection server")
			timer.Stop()
			return
		}
	}

	// If timeout is not specified, leave as long-running thread
	select {}

}

////////////////////////////////////////////////////////////////////////////////////////////
// Connection Client Logic
////////////////////////////////////////////////////////////////////////////////////////////

// Secure (authenticated) connection client path
func secureConnClient(statePath, regCode string, statePass []byte,
	cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams) {
	// Load client ------------------------------------------------------------------
	e2eClient := initializeConnectClient(statePath, regCode, statePass,
		cmixParams, e2eParams)

	// Start network threads---------------------------------------------------------

	// Set networkFollowerTimeout to a value of your choice (seconds)
	networkFollowerTimeout := 5 * time.Second
	err := e2eClient.StartNetworkFollower(networkFollowerTimeout)
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
	e2eClient.GetCmix().AddHealthCallback(
		func(isConnected bool) {
			connected <- isConnected
		})
	// Wait until connected or crash on timeout
	waitUntilConnected(connected)

	// Connect with the server-------------------------------------------------
	contactPath := viper.GetString(connectionFlag)
	serverContact := getContactFromFile(contactPath)
	fmt.Println("Sending connection request")

	// Establish connection with partner
	conn, err := connect.ConnectWithAuthentication(serverContact, e2eClient,
		e2eParams)
	if err != nil {
		jww.FATAL.Panicf("[CONN] Failed to build connection with %s: %v",
			serverContact.ID, err)
	}

	jww.INFO.Printf("[CONN] Established authenticated connection with %s",
		conn.GetPartner().PartnerId())
	fmt.Println("Established authenticated connection with server.")

	miscConnectionFunctions(e2eClient, conn)

}

// Insecure (unauthenticated) connection client path
func insecureConnClient(statePath, regCode string, statePass []byte,
	cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams) {
	// Load client ------------------------------------------------------------------
	e2eClient := initializeConnectClient(statePath, regCode, statePass,
		cmixParams, e2eParams)

	// Start network threads---------------------------------------------------------

	// Set networkFollowerTimeout to a value of your choice (seconds)
	networkFollowerTimeout := 5 * time.Second
	err := e2eClient.StartNetworkFollower(networkFollowerTimeout)
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
	e2eClient.GetCmix().AddHealthCallback(
		func(isConnected bool) {
			connected <- isConnected
		})
	// Wait until connected or crash on timeout
	waitUntilConnected(connected)

	// Connect with the server-------------------------------------------------
	contactPath := viper.GetString(connectionFlag)
	serverContact := getContactFromFile(contactPath)
	fmt.Println("Sending connection request")
	jww.INFO.Printf("[CONN] Sending connection request to %s",
		serverContact.ID)

	// Establish connection with partner
	handler, err := connect.Connect(serverContact, e2eClient,
		e2eParams)
	if err != nil {
		jww.FATAL.Panicf("[CONN] Failed to build connection with %s: %v",
			serverContact.ID, err)

	}

	fmt.Println("Established connection with server")
	jww.INFO.Printf("[CONN] Established connection with %s", handler.GetPartner().PartnerId())

	miscConnectionFunctions(e2eClient, handler)
}

////////////////////////////////////////////////////////////////////////////////////////////
// Misc Logic (shared between client & server)
////////////////////////////////////////////////////////////////////////////////////////////

// miscConnectionFunctions contains miscellaneous functionality for the subcommand connect.
// This functionality should be shared between client & server.
func miscConnectionFunctions(client *xxdk.E2e, conn connect.Connection) {
	// Send a message to connection partner--------------------------------------------
	msgBody := viper.GetString(messageFlag)
	paramsE2E := e2e.GetDefaultParams()
	if msgBody != "" {
		// Send message
		jww.INFO.Printf("[CONN] Sending message to %s",
			conn.GetPartner().PartnerId())
		payload := []byte(msgBody)
		for {
			roundIDs, _, _, err := conn.SendE2E(catalog.XxMessage, payload,
				paramsE2E)
			if err != nil {
				jww.FATAL.Panicf("[CONN] Failed to send E2E message: %v", err)
			}

			// Verify message sends were successful when verifySendFlag is present
			if viper.GetBool(verifySendFlag) {
				if !verifySendSuccess(client, paramsE2E, roundIDs,
					conn.GetPartner().PartnerId(), payload) {
					continue
				}

			}
			jww.INFO.Printf("[CONN] Sent message %q to %s", msgBody,
				conn.GetPartner().PartnerId())
			fmt.Printf("Sent message %q to connection partner.\n", msgBody)
			break
		}
	}

	// Disconnect from connection partner--------------------------------------------
	if viper.GetBool(connectionDisconnectFlag) {
		// Close the connection
		if err := conn.Close(); err != nil {
			jww.FATAL.Panicf("Failed to disconnect with %s: %v",
				conn.GetPartner().PartnerId(), err)
		}
		jww.INFO.Printf("[CONN] Disconnected from %s",
			conn.GetPartner().PartnerId())
		fmt.Println("Disconnected from partner")
	}
}

////////////////////////////////////////////////////////////////////////////////////////////
// Init Logic: Creates xxdk.Cmix & xxdk.E2E objects
////////////////////////////////////////////////////////////////////////////////////////////

// Initialize a xxdk.Cmix client. Basic client may be used for server initialization
// or for use in building a connection client.
func initializeBasicConnectionClient(statePath, regCode string, statePass []byte,
	parms xxdk.CMIXParams) (*xxdk.Cmix,
	xxdk.ReceptionIdentity) {
	var baseClient *xxdk.Cmix
	var identity xxdk.ReceptionIdentity
	var err error

	// Check if state exists
	if _, err := os.Stat(statePath); errors.Is(err, fs.ErrNotExist) {

		// create a new client if none exist------------------------------------
		ndfJson, err := ioutil.ReadFile(viper.GetString("ndf"))
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		err = xxdk.NewCmix(string(ndfJson), statePath, statePass, regCode)
		baseClient, err = xxdk.LoadCmix(statePath, statePass,
			parms)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		identity, err = xxdk.MakeReceptionIdentity(baseClient)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		err = xxdk.StoreReceptionIdentity(identityStorageKey, identity, baseClient)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		return baseClient, identity
	}

	// Load a client from storage---------------------------------------------------
	baseClient, err = xxdk.LoadCmix(statePath, statePass,
		xxdk.GetDefaultCMixParams())
	if err != nil {
		jww.FATAL.Panicf("Failed to load state: %+v", err)
	}

	identity, err = xxdk.LoadReceptionIdentity(identityStorageKey, baseClient)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	return baseClient, identity
}

// Initialize an xxdk.E2e for the connection client.
func initializeConnectClient(statePath, regCode string, statePass []byte,
	cmixParms xxdk.CMIXParams, e2eParams xxdk.E2EParams) *xxdk.E2e {
	// Initialize basic client-------------------------------------------------------
	baseClient, identity := initializeBasicConnectionClient(statePath, regCode,
		statePass, cmixParms)

	// Connect Client Specific ------------------------------------------------------

	// Create an E2E client
	// The `connect` packages handles AuthCallbacks,
	// `xxdk.DefaultAuthCallbacks` is fine here
	jww.INFO.Printf("Using E2E parameters: %+v", e2eParams)
	e2eClient, err := xxdk.Login(baseClient, xxdk.DefaultAuthCallbacks{},
		identity, e2eParams)
	if err != nil {
		jww.FATAL.Panicf("Unable to Login: %+v", err)
	}

	return e2eClient
}

///////////////////////////////////////////////////////////////////////////////
// Recreated Callback & Listener for connection testing
///////////////////////////////////////////////////////////////////////////////

//var connAuthCbs *authConnHandler

// listener implements the receive.Listener interface
type listener struct {
	name string
}

// Hear will be called whenever a message matching
// the RegisterListener call is received
// User-defined message handling logic goes here
func (l listener) Hear(item receive.Message) {
	fmt.Printf("%s heard message \"%s\"\n", l.name, string(item.Payload))
}

// Name is used for debugging purposes
func (l listener) Name() string {
	return l.name
}

///////////////////////////////////////////////////////////////////////////////
// Command Line Flags                                                         /
///////////////////////////////////////////////////////////////////////////////

// init initializes commands and flags for Cobra.
func init() {
	connectionCmd.Flags().String(connectionFlag, "",
		"This flag is a client side operation. "+
			"This flag expects a path to a contact file (similar "+
			"to destfile). It will parse this into an contact object,"+
			" referred to as a server contact. The client will "+
			"establish a connection with the server contact. "+
			"If a connection already exists between "+
			"the client and the server, this will be used instead of "+
			"resending a connection request to the server.")
	bindFlagHelper(connectionFlag, connectionCmd)

	connectionCmd.Flags().Bool(connectionStartServerFlag, false,
		"This flag is a server-side operation and takes no arguments. "+
			"This initiates a connection server. "+
			"Calling this flag will have this process call "+
			"connection.StartServer().")
	bindFlagHelper(connectionStartServerFlag, connectionCmd)

	connectionCmd.Flags().Duration(connectionServerTimeoutFlag, time.Duration(0),
		"This flag is a connection parameter. "+
			"This takes as an argument a time.Duration. "+
			"This duration specifies how long a server will run before "+
			"closing. Without this flag present, a server will be "+
			"long-running.")
	bindFlagHelper(connectionServerTimeoutFlag, connectionCmd)

	connectionCmd.Flags().Bool(connectionDisconnectFlag, false,
		"This flag is available to both server and client. "+
			"This uses a contact object from a file specified by --destfile."+
			"This will close the connection with the given contact "+
			"if it exists.")
	bindFlagHelper(connectionDisconnectFlag, connectionCmd)

	connectionCmd.Flags().Bool(connectionAuthenticatedFlag, false,
		"This flag is available to both server and client. "+
			"This flag operates as a switch for the authenticated code-path. "+
			"With this flag present, any additional connection related flags"+
			" will call the applicable authenticated counterpart")
	bindFlagHelper(connectionAuthenticatedFlag, connectionCmd)

	rootCmd.AddCommand(connectionCmd)
}
