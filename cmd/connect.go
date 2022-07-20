package cmd

import (
	"github.com/spf13/cobra"
	cmdUtils "gitlab.com/elixxir/client/cmdUtils"
	connCmd "gitlab.com/elixxir/client/connect/cmd"
	"time"
)

// connectionCmd handles the operation of connection operations within the CLI.
var connectionCmd = &cobra.Command{
	Use:   "connection",
	Short: "Runs clients and servers in the connections paradigm.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		connCmd.Start()
	},
}

// init initializes commands and flags for Cobra.
func init() {
	connectionCmd.Flags().String(connCmd.ConnectionFlag, "",
		"This flag is a client side operation. "+
			"This flag expects a path to a contact file (similar "+
			"to destfile). It will parse this into an contact object,"+
			" referred to as a server contact. The client will "+
			"establish a connection with the server contact. "+
			"If a connection already exists between "+
			"the client and the server, this will be used instead of "+
			"resending a connection request to the server.")
	cmdUtils.BindFlagHelper(connCmd.ConnectionFlag, connectionCmd)

	connectionCmd.Flags().Bool(connCmd.ConnectionStartServerFlag, false,
		"This flag is a server-side operation and takes no arguments. "+
			"This initiates a connection server. "+
			"Calling this flag will have this process call "+
			"connection.StartServer().")
	cmdUtils.BindFlagHelper(connCmd.ConnectionStartServerFlag, connectionCmd)

	connectionCmd.Flags().Duration(connCmd.ConnectionServerTimeoutFlag, time.Duration(0),
		"This flag is a connection parameter. "+
			"This takes as an argument a time.Duration. "+
			"This duration specifies how long a server will run before "+
			"closing. Without this flag present, a server will be "+
			"long-running.")
	cmdUtils.BindFlagHelper(connCmd.ConnectionServerTimeoutFlag, connectionCmd)

	connectionCmd.Flags().Bool(connCmd.ConnectionDisconnectFlag, false,
		"This flag is available to both server and client. "+
			"This uses a contact object from a file specified by --destfile."+
			"This will close the connection with the given contact "+
			"if it exists.")
	cmdUtils.BindFlagHelper(connCmd.ConnectionDisconnectFlag, connectionCmd)

	connectionCmd.Flags().Bool(connCmd.ConnectionAuthenticatedFlag, false,
		"This flag is available to both server and client. "+
			"This flag operates as a switch for the authenticated code-path. "+
			"With this flag present, any additional connection related flags"+
			" will call the applicable authenticated counterpart")
	cmdUtils.BindFlagHelper(connCmd.ConnectionAuthenticatedFlag, connectionCmd)

	connectionCmd.Flags().Bool(connCmd.ConnectionEphemeralFlag, false,
		"This flag is available to both server and client. "+
			"This flag operates as a switch determining the initialization path."+
			"If present, the messenger will be initialized ephemerally. Without this flag, "+
			"the messenger will be initialized as stateful.")
	cmdUtils.BindFlagHelper(connCmd.ConnectionEphemeralFlag, connectionCmd)

	rootCmd.AddCommand(connectionCmd)
}
