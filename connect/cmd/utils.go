package cmd

import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/catalog"
	cmdUtils "gitlab.com/elixxir/client/cmdUtils"
	"gitlab.com/elixxir/client/connect"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/client/xxdk"
)

////////////////////////////////////////////////////////////////////////////////////////////
// Misc Logic (shared between client & server)
////////////////////////////////////////////////////////////////////////////////////////////

// miscConnectionFunctions contains miscellaneous functionality for the subcommand connect.
// This functionality should be shared between client & server.
func miscConnectionFunctions(messenger *xxdk.E2e, conn connect.Connection) {
	// Send a message to connection partner--------------------------------------------
	msgBody := viper.GetString(cmdUtils.MessageFlag)
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
			if viper.GetBool(cmdUtils.VerifySendFlag) {
				if !cmdUtils.VerifySendSuccess(messenger, paramsE2E, roundIDs,
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
	if viper.GetBool(ConnectionDisconnectFlag) {
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
