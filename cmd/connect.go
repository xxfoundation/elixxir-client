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

		// Handle either authenticated or standard connection path
		if viper.GetBool(authenticatedFlag) {
			authenticatedConnections()
		} else {
			connections()
		}

	},
}

// connections is the CLI handler for un-authenticated connect.Connection's.
func connections() {
	// NOTE: for now this supports one connection for servers, for integration
	//  testing.
	connChan := make(chan connect.Connection, 1)
	var err error
	e2eParams := xxdk.GetDefaultE2EParams()
	statePass := parsePassword(viper.GetString(passwordFlag))
	statePath := viper.GetString(sessionFlag)

	// Connection Server path--------------------------------------------------------
	if viper.GetBool(startServerFlag) {

		// Load client state and identity------------------------------------------
		baseClient, identity := initializeBasicConnectionClient(statePath, statePass)

		// Save contact file-------------------------------------------------------
		writeContact(identity.GetContact())

		// Handle incoming connections---------------------------------------------
		cb := connect.Callback(func(connection connect.Connection) {
			partnerId := connection.GetPartner().PartnerId()
			jww.INFO.Printf("[CONN] Received connection request from %s", partnerId)
			fmt.Println("Established connection with client")

			_, err = connection.RegisterListener(catalog.XxMessage, listener{"ConnectionServer"})
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
		serverTimeout := viper.GetDuration(serverTimeoutFlag)
		if viper.GetDuration(serverTimeoutFlag) != 0 {
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

	} else {
		// Connection Client path--------------------------------------------------------

		// Load client ------------------------------------------------------------------
		e2eClient := initializeConnectClient(statePath, statePass)

		// Start network threads---------------------------------------------------------

		// Set networkFollowerTimeout to a value of your choice (seconds)
		networkFollowerTimeout := 5 * time.Second
		err = e2eClient.StartNetworkFollower(networkFollowerTimeout)
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
}

// authenticatedConnections is the CLI handler for
// connect.AuthenticatedConnection's.
func authenticatedConnections() {
	// NOTE: for now this supports one connection for servers, for integration
	//  testing.
	connChan := make(chan connect.Connection, 1)
	var err error
	e2eParams := xxdk.GetDefaultE2EParams()
	statePass := parsePassword(viper.GetString(passwordFlag))
	statePath := viper.GetString(sessionFlag)

	// Connection Server path--------------------------------------------------------
	if viper.GetBool(startServerFlag) {
		// Load client state and identity------------------------------------------
		baseClient, identity := initializeBasicConnectionClient(statePath, statePass)

		// Save contact file-------------------------------------------------------
		writeContact(identity.GetContact())

		// Handle incoming connections---------------------------------------------
		authCb := connect.AuthenticatedCallback(
			func(connection connect.AuthenticatedConnection) {
				partnerId := connection.GetPartner().PartnerId()
				jww.INFO.Printf("[CONN] Received authenticated connection from %s", partnerId)
				fmt.Println("Established authenticated connection with client")

				_, err = connection.RegisterListener(catalog.XxMessage, listener{"AuthServer"})
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
		serverTimeout := viper.GetDuration(serverTimeoutFlag)
		if viper.GetDuration(serverTimeoutFlag) != 0 {
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

	} else {
		// Load client ------------------------------------------------------------------
		e2eClient := initializeConnectClient(statePath, statePass)

		// Start network threads---------------------------------------------------------

		// Set networkFollowerTimeout to a value of your choice (seconds)
		networkFollowerTimeout := 5 * time.Second
		err = e2eClient.StartNetworkFollower(networkFollowerTimeout)
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

}

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
	if viper.GetBool(disconnectFlag) {
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

// Initialize a xxdk.Cmix client. Basic client may be used for server initialization
// or for use in building a connection client.
func initializeBasicConnectionClient(statePath string, statePass []byte) (*xxdk.Cmix,
	xxdk.ReceptionIdentity) {

	// Check if state exists
	if _, err := os.Stat(statePath); errors.Is(err, fs.ErrNotExist) {

		// Load NDF----------------------------------------------------------------------
		ndfJSON, err := ioutil.ReadFile(viper.GetString(ndfFlag))
		if err != nil {
			jww.FATAL.Panicf(err.Error())
		}

		// Initialize the state----------------------------------------------------------
		err = xxdk.NewCmix(string(ndfJSON), statePath, statePass, "")
		if err != nil {
			jww.FATAL.Panicf("Failed to initialize state: %+v", err)
		}
	}

	// Load with the same sessionPath and sessionPass used to call NewClient()
	baseClient, err := xxdk.LoadCmix(statePath, statePass,
		xxdk.GetDefaultCMixParams())
	if err != nil {
		jww.FATAL.Panicf("Failed to load state: %+v", err)
	}

	return baseClient, loadOrMakeIdentity(baseClient)
}

// Initialize an xxdk.E2e for the connection client.
func initializeConnectClient(statePath string, statePass []byte) *xxdk.E2e {
	// Initialize basic client-------------------------------------------------------
	baseClient, identity := initializeBasicConnectionClient(statePath, statePass)

	// Connect Client Specific ------------------------------------------------------

	// Create an E2E client
	// The `connect` packages handles AuthCallbacks,
	// `xxdk.DefaultAuthCallbacks` is fine here
	params := xxdk.GetDefaultE2EParams()
	jww.INFO.Printf("Using E2E parameters: %+v", params)
	e2eClient, err := xxdk.Login(baseClient, xxdk.DefaultAuthCallbacks{},
		identity, params)
	if err != nil {
		jww.FATAL.Panicf("Unable to Login: %+v", err)
	}

	return e2eClient
}

///////////////////////////////////////////////////////////////////////////////
// Recreated Callback & Listener for cmd
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

// todo: figure out if I need all this by testing in integration
func makeAuthConnHandler(isAuthenticated bool) *authConnHandler {
	return &authConnHandler{
		isAuth: isAuthenticated,
	}
}

type authConnHandler struct {
	client *xxdk.E2e
	conn   connect.Connection
	connCb connect.Callback

	authConnCb connect.AuthenticatedCallback
	isAuth     bool
}

//func (a *authConnHandler) Hear(item receive.Message) {
//	if item.MessageType == catalog.XxMessage {
//		fmt.Printf("Received message: %s\n", string(item.Payload))
//
//	} else if item.MessageType == catalog.ConnectionAuthenticationRequest {
//		// Process the message data into a protobuf
//		iar := &connect.IdentityAuthentication{}
//		err := proto.Unmarshal(item.Payload, iar)
//		if err != nil {
//			jww.FATAL.Panicf("Failed to unmarshal message: %s", err)
//		}
//
//		// Get the new partner
//		newPartner := a.conn.GetPartner()
//		connectionFp := newPartner.ConnectionFingerprint().Bytes()
//
//		// Verify the signature within the message
//		err = connCrypto.Verify(newPartner.PartnerId(),
//			iar.Signature, connectionFp, iar.RsaPubKey, iar.Salt)
//		if err != nil {
//			jww.FATAL.Panicf("Failed to verify message: %v", err)
//		}
//
//		// If successful, pass along the established authenticated connection
//		// via the callback
//		jww.DEBUG.Printf("AuthenticatedConnection auth request "+
//			"for %s confirmed",
//			item.Sender.String())
//		authConn := connect.buildAuthenticatedConnection(a.conn)
//		go a.authConnCb(authConn)
//	}
//
//}
//
//func (a *authConnHandler) Name() string {
//	return "authConnHandler"
//}
//
//func (a *authConnHandler) Request(partner contact.Contact,
//	receptionID receptionID.EphemeralIdentity,
//	round rounds.Round, e2e *xxdk.E2e) {
//	partnerId := partner.ID
//
//	// Accept channel and send confirmation message
//	if viper.GetBool(verifySendFlag) {
//		// Verify message sends were successful
//		acceptChannelVerified(e2e, partnerId, xxdk.GetDefaultE2EParams())
//	} else {
//		acceptChannel(e2e, partnerId)
//	}
//
//	// After confirmation, get the new partner
//	newPartner, err := e2e.GetE2E().GetPartner(partner.ID)
//	if err != nil {
//		jww.ERROR.Printf("[CONN] Unable to build connection with "+
//			"partner %s: %+v", partner.ID, err)
//		// Send a nil connection to avoid hold-ups down the line
//		if a.connCb != nil {
//			a.connCb(nil)
//		}
//		return
//	}
//
//	a.conn = connect.BuildConnection(newPartner, e2e.GetE2E(),
//		e2e.GetAuth(), connect.GetDefaultParams())
//
//	if a.connCb != nil {
//		// Return the new Connection object
//		a.connCb(a.conn)
//	}
//
//	e2e.GetE2E().RegisterListener(partnerId, catalog.XxMessage, a)
//
//	if a.isAuth {
//		a.conn.RegisterListener(catalog.ConnectionAuthenticationRequest, a)
//	}
//
//}
//
//func (a *authConnHandler) Confirm(partner contact.Contact,
//	receptionID receptionID.EphemeralIdentity, round rounds.Round,
//	e2e *xxdk.E2e) {
//
//	_, e2eParams := initParams()
//	// After confirmation, get the new partner
//	newPartner, err := e2e.GetE2E().GetPartner(partner.ID)
//	if err != nil {
//		jww.ERROR.Printf("[CONN] Unable to build connection with "+
//			"partner %s: %+v", partner.ID, err)
//		// Send a nil connection to avoid hold-ups down the line
//		if a.connCb != nil {
//			a.connCb(nil)
//		}
//
//		if a.authConnCb != nil {
//			a.authConnCb(nil)
//		}
//
//		return
//	}
//
//	// Return the new Connection object
//	if a.connCb != nil {
//		a.connCb(connect.BuildConnection(newPartner, e2e.GetE2E(),
//			e2e.GetAuth(), e2eParams))
//	}
//}
//
//func (a authConnHandler) Reset(partner contact.Contact,
//	receptionID receptionID.EphemeralIdentity, round rounds.Round,
//	e *xxdk.E2e) {
//	return
//}

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
	_ = viper.BindPFlag(connectionFlag, connectionCmd.Flags().
		Lookup(connectionFlag))

	connectionCmd.Flags().Bool(startServerFlag, false,
		"This flag is a server-side operation and takes no arguments. "+
			"This initiates a connection server. "+
			"Calling this flag will have this process call "+
			"connection.StartServer().")
	_ = viper.BindPFlag(startServerFlag, connectionCmd.Flags().
		Lookup(startServerFlag))

	connectionCmd.Flags().Duration(serverTimeoutFlag, time.Duration(0),
		"This flag is a connection parameter. "+
			"This takes as an argument a time.Duration. "+
			"This duration specifies how long a server will run before "+
			"closing. Without this flag present, a server will be "+
			"long-running.")
	_ = viper.BindPFlag(serverTimeoutFlag, connectionCmd.Flags().
		Lookup(serverTimeoutFlag))

	connectionCmd.Flags().Bool(disconnectFlag, false,
		"This flag is available to both server and client. "+
			"This uses a contact object from a file specified by --destfile."+
			"This will close the connection with the given contact "+
			"if it exists.")
	_ = viper.BindPFlag(disconnectFlag, connectionCmd.Flags().
		Lookup(disconnectFlag))

	connectionCmd.Flags().Bool(authenticatedFlag, false,
		"This flag is available to both server and client. "+
			"This flag operates as a switch for the authenticated code-path. "+
			"With this flag present, any additional connection related flags"+
			" will call the applicable authenticated counterpart")
	_ = viper.BindPFlag(authenticatedFlag, connectionCmd.Flags().
		Lookup(authenticatedFlag))

	rootCmd.AddCommand(connectionCmd)
}
