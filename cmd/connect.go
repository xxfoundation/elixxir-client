////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/connect"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/xxdk"
	"time"
)

// connectionCmd handles the operation of connection operations within the CLI.
var connectionCmd = &cobra.Command{
	Use:   "fileTransfer",
	Short: "Runs clients and servers in the connections paradigm.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {

		// Initialise a new client
		client := initClient()

		if viper.GetBool(authenticatedFlag) {
			authenticatedConnections(client)
		} else {
			connections(client)
		}

		// Handle server closing
		serverTimeout := viper.GetDuration(serverTimeoutFlag)
		if viper.GetBool(startServerFlag) {
			// If server timeout is specified, close out on timeout
			if viper.GetDuration(serverTimeoutFlag) != 0 {
				timer := time.NewTimer(serverTimeout)
				select {
				case <-timer.C:
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
func connections(client *xxdk.E2e) {
	// fixme: for now this supports one connection for servers, for integration
	//  testing.
	connChan := make(chan connect.Connection, 1)
	var conn connect.Connection
	var err error
	connectionParam := connect.GetDefaultParams()

	// Start connection server
	if viper.GetBool(startServerFlag) {
		cb := connect.Callback(func(connection connect.Connection) {
			connChan <- connection
		})
		client, err = connect.StartServer(client.GetReceptionIdentity(),
			cb, client.Cmix, connectionParam)
		if err != nil {
			jww.FATAL.Panicf("Failed to start connection server: %v", err)
		}
	}

	// Have client connect to connection server
	if viper.GetBool(connectionFlag) {
		serverContact := readContact()
		// Establish connection with partner
		conn, err := connect.Connect(serverContact, client, connectionParam)
		if err != nil {
			jww.FATAL.Panicf("Failed to establish connection with %s",
				serverContact.ID)
		}

		connChan <- conn
	}

	// Wait for connection to be established
	connectionTimeout := time.NewTimer(connectionParam.Timeout)
	select {
	case conn = <-connChan:
	case <-connectionTimeout.C:
		connectionTimeout.Stop()
		jww.FATAL.Panicf("Failed to establish connection within default " +
			"time period, closing process")

	}

	// Send message
	msgBody := viper.GetString(messageFlag)
	if msgBody != "" {

		payload := []byte(msgBody)

		for {
			paramsE2E := e2e.GetDefaultParams()

			roundIDs, _, _, err := conn.SendE2E(catalog.XxMessage, payload,
				paramsE2E)
			if err != nil {
				jww.FATAL.Panicf("Failed to send E2E message: %v", err)
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

	}

	// Disconnect from partner
	if viper.GetBool(disconnectFlag) {
		if err = conn.Close(); err != nil {
			jww.FATAL.Panicf("Failed to disconnect with %s: %v",
				conn.GetPartner().PartnerId(), err)
		}
	}

}

// authenticatedConnections is the CLI handler for
// connect.AuthenticatedConnection's.
func authenticatedConnections(client *xxdk.E2e) {
	// fixme: for now this supports one connection for servers, for integration
	//  testing.
	connChan := make(chan connect.AuthenticatedConnection, 1)
	var err error
	var conn connect.Connection
	connectionParam := connect.GetDefaultParams()

	// Start authentication connection server
	if viper.GetBool(startServerFlag) {
		cb := connect.AuthenticatedCallback(
			func(connection connect.AuthenticatedConnection) {
				connChan <- connection
			},
		)
		client, err = connect.StartAuthenticatedServer(
			client.GetReceptionIdentity(), cb, client.Cmix, connectionParam)
		if err != nil {
			jww.FATAL.Panicf("Failed to start connection server: %v", err)
		}
	}

	// Have client connect to connection server
	if viper.GetBool(connectionFlag) {
		serverContact := readContact()
		// Establish connection with partner
		conn, err := connect.ConnectWithAuthentication(serverContact, client,
			connectionParam)
		if err != nil {
			jww.FATAL.Panicf("Failed to establish connection with %s",
				serverContact.ID)
		}

		connChan <- conn
	}

	// Wait for connection to be established
	connectionTimeout := time.NewTimer(connectionParam.Timeout)
	select {
	case conn = <-connChan:
	case <-connectionTimeout.C:
		connectionTimeout.Stop()
		jww.FATAL.Panicf("Failed to establish connection within default " +
			"time period, closing process")

	}

	// Send message
	msgBody := viper.GetString(messageFlag)
	if msgBody != "" {
		payload := []byte(msgBody)
		for {
			paramsE2E := e2e.GetDefaultParams()
			roundIDs, _, _, err := conn.SendE2E(catalog.XxMessage, payload,
				paramsE2E)
			if err != nil {
				jww.FATAL.Panicf("Failed to send E2E message: %v", err)
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

	}

	// Disconnect from partner
	if viper.GetBool(disconnectFlag) {
		if err = conn.Close(); err != nil {
			jww.FATAL.Panicf("Failed to disconnect with %s: %v",
				conn.GetPartner().PartnerId(), err)
		}
	}

}

///////////////////////////////////////////////////////////////////////////////
// Command Line Flags                                                         /
///////////////////////////////////////////////////////////////////////////////

// init initializes commands and flags for Cobra.
func init() {

	connectionCmd.Flags().String(connectionFlag, "",
		"This flag is a client side operation. "+
			"This takes a contact from a file specified by --destfile."+
			"This contact file will represent the server this "+
			"client attempts to connect to. "+
			"If a connection already exists between the client and "+
			"the server, Connect() will not be called.")
	_ = viper.BindPFlag(connectionFlag, connectionCmd.Flags().Lookup(connectionFlag))

	connectionCmd.Flags().Bool(startServerFlag, false,
		"This flag is a server-side operation and takes no arguments. "+
			"This initiates a connection server. "+
			"Calling this flag will have this process call "+
			"connection.StartServer().")
	_ = viper.BindPFlag(startServerFlag, connectionCmd.Flags().Lookup(startServerFlag))

	connectionCmd.Flags().Duration(serverTimeoutFlag, time.Duration(0),
		"This flag is a connection parameter. "+
			"This takes in a time.Duration as an argument. "+
			"This duration specifies how long a server will run before closing. "+
			"Without this flag present, a server will be long-running.")
	_ = viper.BindPFlag(serverTimeoutFlag, connectionCmd.Flags().Lookup(serverTimeoutFlag))

	connectionCmd.Flags().Bool(disconnectFlag, false,
		"This flag is available to both server and client. "+
			"This takes a contact from a file specified by --destfile."+
			"This will close the connection with the given contact "+
			"if it exists.")
	_ = viper.BindPFlag(disconnectFlag, connectionCmd.Flags().Lookup(disconnectFlag))

	connectionCmd.Flags().Bool(authenticatedFlag, false,
		"This flag is available to both server and client. "+
			"This flag operates as a switch for the authenticated code-path. "+
			"With this flag present, any additional other flags will call the "+
			"applicable authenticated counterpart")
	_ = viper.BindPFlag(authenticatedFlag, connectionCmd.Flags().Lookup(authenticatedFlag))

	rootCmd.AddCommand(connectionCmd)
}
