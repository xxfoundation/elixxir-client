////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/connect"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/client/xxdk"
	connCrypto "gitlab.com/elixxir/crypto/connect"
	"gitlab.com/elixxir/crypto/contact"
	"time"
)

// connectionCmd handles the operation of connection operations within the CLI.
var connectionCmd = &cobra.Command{
	Use:   "connection",
	Short: "Runs clients and servers in the connections paradigm.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if viper.GetBool(authenticatedFlag) {
			authenticatedConnections()
		} else {
			connections()
		}

		// Handle server closing
		serverTimeout := viper.GetDuration(serverTimeoutFlag)
		if viper.GetBool(startServerFlag) {
			// If server timeout is specified, close out on timeout
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
		}

	},
}

// connections is the CLI handler for un-authenticated connect.Connection's.
func connections() {
	// NOTE: for now this supports one connection for servers, for integration
	//  testing.
	connChan := make(chan connect.Connection, 1)
	var conn connect.Connection
	var err error
	var client *xxdk.E2e
	connectionParam := connect.GetDefaultParams()

	// Start connection server
	if viper.GetBool(startServerFlag) {
		// Construct connection callback
		cb := connect.Callback(func(connection connect.Connection) {
			partnerId := connection.GetPartner().PartnerId()
			jww.INFO.Printf("[CONN] Received connection from %s",
				partnerId)

			connChan <- connection
		})

		// Construct a client
		client = initializeConnectionServerE2e()
		u := client.GetUser()
		ri := xxdk.ReceptionIdentity{
			ID:            u.ReceptionID,
			RSAPrivatePem: u.ReceptionRSA,
			Salt:          u.ReceptionSalt,
			DHKeyPrivate:  client.GetE2E().GetHistoricalDHPrivkey(),
		}

		// Start a connection server
		client, err = connect.StartServer(ri,
			cb, client.Cmix, connectionParam)
		if err != nil {
			jww.FATAL.Panicf("[CONN] Failed to start connection server: %v", err)
		}

		connAuthCbs.connCb = cb

		// Start network follower
		err = client.StartNetworkFollower(5 * time.Second)
		if err != nil {
			jww.FATAL.Panicf("[CONN] Failed to start network follower: %+v", err)
		}

		// Wait until connected or crash on timeout
		connected := make(chan bool, 10)
		client.GetCmix().AddHealthCallback(
			func(isconnected bool) {
				connected <- isconnected
			})
		waitUntilConnected(connected)

		// Print user's reception ID and save contact file
		user := client.GetUser()
		jww.INFO.Printf("User: %s", user.ReceptionID)
		writeContact(user.GetContact())

		jww.INFO.Printf("[CONN] Established connection server, begin listening...")
		fmt.Println("Established connection server, begin listening...")
	} else {
		// Initialize client for client-side connection operations
		client = initializeConnectionClient()
	}

	// Have client connect to connection server
	contactPath := viper.GetString(connectionFlag)
	if contactPath != "" {
		serverContact := getContactFromFile(contactPath)
		jww.INFO.Printf("[CONN] Sending connection request to %s",
			serverContact.ID)

		// Establish connection with partner
		conn, err = connect.Connect(serverContact, client,
			connectionParam)
		if err != nil {
			jww.FATAL.Panicf("[CONN] Failed to build connection with %s: %v",
				serverContact.ID, err)
		}

		connChan <- conn
	}

	// Wait for connection to be established
	connectionTimeout := time.NewTimer(240 * time.Second)
	select {
	case conn = <-connChan:
	case <-connectionTimeout.C:
		connectionTimeout.Stop()
		jww.FATAL.Panicf("[CONN] Failed to establish connection within " +
			"default time period, closing process")

	}

	jww.INFO.Printf("[CONN] Established connection with %s",
		conn.GetPartner().PartnerId())

	// Send message
	msgBody := viper.GetString(messageFlag)
	if msgBody != "" {
		jww.INFO.Printf("[CONN] Sending message to %s",
			conn.GetPartner().PartnerId())
		payload := []byte(msgBody)

		for {
			paramsE2E := e2e.GetDefaultParams()

			roundIDs, _, _, err := conn.SendE2E(catalog.XxMessage, payload,
				paramsE2E)
			if err != nil {
				jww.FATAL.Panicf("[CONN] Failed to send E2E message: %v", err)
			}

			// Verify message sends were successful when there is a flag
			// asserting verification
			if viper.GetBool(verifySendFlag) {
				if !verifySendSuccess(client, paramsE2E, roundIDs,
					conn.GetPartner().PartnerId(), payload) {
					continue
				}

			}

			break
		}
		jww.INFO.Printf("[CONN] Sent message %q to %s", msgBody,
			conn.GetPartner().PartnerId())
		fmt.Printf("Sent message %q to connection partner.\n", msgBody)

	}

	// Disconnect from partner
	if viper.GetBool(disconnectFlag) {
		jww.INFO.Printf("[CONN] Disconnecting from %s",
			conn.GetPartner().PartnerId())
		fmt.Println("Disconnecting from connection partner")
		if err = conn.Close(); err != nil {
			jww.FATAL.Panicf("Failed to disconnect with %s: %v",
				conn.GetPartner().PartnerId(), err)
		}
		jww.INFO.Printf("[CONN] Disconnected from %s",
			conn.GetPartner().PartnerId())
		fmt.Println("Disconnected from partner")
	}

}

// authenticatedConnections is the CLI handler for
// connect.AuthenticatedConnection's.
func authenticatedConnections() {
	// NOTE: for now this supports one connection for servers, for integration
	//  testing.
	connChan := make(chan connect.AuthenticatedConnection, 1)
	var conn connect.AuthenticatedConnection
	var err error
	var client *xxdk.E2e
	connectionParam := connect.GetDefaultParams()
	// Start authentication connection server
	if viper.GetBool(startServerFlag) {
		// Construct connection callback
		cb := connect.AuthenticatedCallback(
			func(connection connect.AuthenticatedConnection) {
				partnerId := connection.GetPartner().PartnerId()
				fmt.Printf("Received authenticated connection from client\n")
				jww.INFO.Printf("[CONN] Received authenticated connection "+
					"from %s", partnerId)

				connChan <- connection
			})

		// Construct a client
		client = initializeConnectionServerE2e()
		u := client.GetUser()
		ri := xxdk.ReceptionIdentity{
			ID:            u.ReceptionID,
			RSAPrivatePem: u.ReceptionRSA,
			Salt:          u.ReceptionSalt,
			DHKeyPrivate:  client.GetE2E().GetHistoricalDHPrivkey(),
		}

		// Start a authenticated server
		client, err = connect.StartAuthenticatedServer(
			ri, cb, client.Cmix, connectionParam)
		if err != nil {
			jww.FATAL.Panicf("Failed to start authenticated "+
				"connection server: %v", err)
		}

		connAuthCbs.authConnCb = cb

		// Start network follower
		err = client.StartNetworkFollower(5 * time.Second)
		if err != nil {
			jww.FATAL.Panicf("Failed to start network follower: %+v", err)
		}

		// Wait until connected or crash on timeout
		connected := make(chan bool, 10)
		client.GetCmix().AddHealthCallback(
			func(isconnected bool) {
				connected <- isconnected
			})
		waitUntilConnected(connected)

		// Print user's reception ID and save contact file
		user := client.GetUser()
		jww.INFO.Printf("User: %s", user.ReceptionID)
		writeContact(user.GetContact())
		jww.INFO.Printf("[CONN] Established connection server, " +
			"begin listening for authentication...")
		fmt.Println("Established connection server, " +
			"begin listening for authentication...")

	} else {
		// Initialize client for client-side connection operations
		client = initializeConnectionClient()
	}

	// Have client connect to connection server
	contactPath := viper.GetString(connectionFlag)
	if contactPath != "" {
		serverContact := getContactFromFile(contactPath)

		// Establish connection with partner
		conn, err = connect.ConnectWithAuthentication(serverContact, client,
			connectionParam)
		if err != nil {
			jww.FATAL.Panicf("[CONN] Failed to build connection with %s",
				serverContact.ID)
		}

		connChan <- conn
	}

	// Wait for connection to be established
	var connectionTimeout *time.Timer
	if viper.GetBool(authenticatedFlag) {
		connectionTimeout = time.NewTimer(20 * time.Second)
	} else {
		connectionTimeout = time.NewTimer(connectionParam.Timeout)
	}
	select {
	case conn = <-connChan:
	case <-connectionTimeout.C:
		connectionTimeout.Stop()
		jww.FATAL.Panicf("[CONN] Failed to establish connection within " +
			"default time period, closing process")

	}

	jww.INFO.Printf("[CONN] Established connection with %s, "+
		"awaiting authentication...",
		conn.GetPartner().PartnerId())
	fmt.Printf("Established connection with partner " +
		"awaiting authentication...\n")

	// Send message
	msgBody := viper.GetString(messageFlag)
	if msgBody != "" {
		jww.INFO.Printf("[CONN] Sending message to %s",
			conn.GetPartner().PartnerId())
		payload := []byte(msgBody)
		for {
			paramsE2E := e2e.GetDefaultParams()
			roundIDs, _, _, err := conn.SendE2E(catalog.XxMessage, payload,
				paramsE2E)
			if err != nil {
				jww.FATAL.Panicf("[CONN] Failed to send E2E message: %v", err)
			}

			// Verify message sends were successful when there is a flag
			// asserting verification
			if viper.GetBool(verifySendFlag) {
				if !verifySendSuccess(client, paramsE2E, roundIDs,
					conn.GetPartner().PartnerId(), payload) {
					continue
				}
			}
			break
		}
		jww.INFO.Printf("[CONN] Sent message %q to %s", msgBody,
			conn.GetPartner().PartnerId())
		fmt.Printf("Sent message %q to connection partner.\n", msgBody)

	}

	// Disconnect from partner
	if viper.GetBool(disconnectFlag) {
		jww.INFO.Printf("[CONN] Disconnecting from %s",
			conn.GetPartner().PartnerId())
		fmt.Println("Disconnecting from connection partner")

		if err = conn.Close(); err != nil {
			jww.FATAL.Panicf("[CONN] Failed to disconnect with %s: %v",
				conn.GetPartner().PartnerId(), err)
		}
		jww.INFO.Printf("[CONN] Disconnected from %s",
			conn.GetPartner().PartnerId())
		fmt.Println("Disconnected from partner")
	}

}

// Initialize a xxdk.E2e for use for the connection's server.
// The server will create a new instance of an xxdk.E2e after a
// call to connect.StartServer, however this initializes keys and
// sets up local auth callbacks.
func initializeConnectionServerE2e() *xxdk.E2e {
	initClient()

	pass := parsePassword(viper.GetString(passwordFlag))
	storeDir := viper.GetString(sessionFlag)
	jww.DEBUG.Printf("sessionDur: %v", storeDir)

	params := initParams()

	// "oad the client
	baseClient, err := xxdk.LoadCmix(storeDir, pass, params)

	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	// Create auth callbacks
	connAuthCbs = makeAuthConnHandler(viper.GetBool(authenticatedFlag))

	// Login to client
	client, err := xxdk.LoginLegacy(baseClient, connAuthCbs)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	connAuthCbs.client = client

	return client

}

// Initialize an xxdk.E2e for the connection client.
func initializeConnectionClient() *xxdk.E2e {
	// Initialise a new client
	client := initClient()

	err := client.StartNetworkFollower(5 * time.Second)
	if err != nil {
		jww.FATAL.Panicf("[CONN] Failed to start network follower: %+v", err)
	}

	// Wait until connected or crash on timeout
	connected := make(chan bool, 10)
	client.GetCmix().AddHealthCallback(
		func(isconnected bool) {
			connected <- isconnected
		})
	waitUntilConnected(connected)

	return client
}

///////////////////////////////////////////////////////////////////////////////
// Recreated Callback & Listener for cmd
///////////////////////////////////////////////////////////////////////////////

var connAuthCbs *authConnHandler

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

func (a *authConnHandler) Hear(item receive.Message) {
	if item.MessageType == catalog.XxMessage {
		fmt.Printf("Received message: %s\n", string(item.Payload))

	} else if item.MessageType == catalog.ConnectionAuthenticationRequest {
		// Process the message data into a protobuf
		iar := &connect.IdentityAuthentication{}
		err := proto.Unmarshal(item.Payload, iar)
		if err != nil {
			jww.FATAL.Panicf("Failed to unmarshal message: %s", err)
		}

		// Get the new partner
		newPartner := a.conn.GetPartner()
		connectionFp := newPartner.ConnectionFingerprint().Bytes()

		// Verify the signature within the message
		err = connCrypto.Verify(newPartner.PartnerId(),
			iar.Signature, connectionFp, iar.RsaPubKey, iar.Salt)
		if err != nil {
			jww.FATAL.Panicf("Failed to verify message: %v", err)
		}

		// If successful, pass along the established authenticated connection
		// via the callback
		jww.DEBUG.Printf("AuthenticatedConnection auth request "+
			"for %s confirmed",
			item.Sender.String())
		fmt.Println("Authenticated connection established")
		authConn := connect.BuildAuthenticatedConnection(a.conn)
		go a.authConnCb(authConn)
	}

}

func (a *authConnHandler) Name() string {
	return "authConnHandler"
}

func (a *authConnHandler) Request(partner contact.Contact,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	partnerId := partner.ID

	// Accept channel and send confirmation message
	if viper.GetBool(verifySendFlag) {
		// Verify message sends were successful
		acceptChannelVerified(a.client, partnerId)
	} else {
		acceptChannel(a.client, partnerId)
	}

	// After confirmation, get the new partner
	newPartner, err := a.client.GetE2E().GetPartner(partner.ID)
	if err != nil {
		jww.ERROR.Printf("[CONN] Unable to build connection with "+
			"partner %s: %+v", partner.ID, err)
		// Send a nil connection to avoid hold-ups down the line
		if a.connCb != nil {
			a.connCb(nil)
		}
		return
	}

	a.conn = connect.BuildConnection(newPartner, a.client.GetE2E(),
		a.client.GetAuth(), connect.GetDefaultParams())

	if a.connCb != nil {
		// Return the new Connection object
		a.connCb(a.conn)
	}

	a.client.GetE2E().RegisterListener(partnerId, catalog.XxMessage, a)

	if a.isAuth {
		a.conn.RegisterListener(catalog.ConnectionAuthenticationRequest, a)
	}

}

func (a *authConnHandler) Confirm(partner contact.Contact,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	// After confirmation, get the new partner
	newPartner, err := a.client.GetE2E().GetPartner(partner.ID)
	if err != nil {
		jww.ERROR.Printf("[CONN] Unable to build connection with "+
			"partner %s: %+v", partner.ID, err)
		// Send a nil connection to avoid hold-ups down the line
		if a.connCb != nil {
			a.connCb(nil)
		}

		if a.authConnCb != nil {
			a.authConnCb(nil)
		}

		return
	}

	// Return the new Connection object
	if a.connCb != nil {
		a.connCb(connect.BuildConnection(newPartner, a.client.GetE2E(),
			a.client.GetAuth(), connect.GetDefaultParams()))
	}
}

func (a authConnHandler) Reset(partner contact.Contact,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	return
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
