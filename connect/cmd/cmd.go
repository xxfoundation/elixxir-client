////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"github.com/spf13/viper"
	cmdUtils "gitlab.com/elixxir/client/cmdUtils"
	"gitlab.com/elixxir/client/xxdk"
)

// Start is the ingress point for this package. This allows for the initialization of
// a CLI connection service; clients or servers. Either service can use authentication
// (known as being "secure") if the authenticated flag is provided.
func Start() {
	// Initialize log
	logLevel := viper.GetUint(cmdUtils.LogLevelFlag)
	logPath := viper.GetString(cmdUtils.LogFlag)
	cmdUtils.InitLog(logLevel, logPath)

	// Initialize paramaters
	cmixParams, e2eParams := cmdUtils.InitParams()

	// Start connection instance
	if viper.GetBool(ConnectionStartServerFlag) {
		startServer(cmixParams, e2eParams)
	} else {
		startClient(cmixParams, e2eParams)
	}
}

// startServer is a helper function which initializes a connection server.
func startServer(cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams) {
	// Pull flags for initializing a messenger
	forceLegacy := viper.GetBool(cmdUtils.ForceLegacyFlag)
	statePass := cmdUtils.ParsePassword(viper.GetString(cmdUtils.PasswordFlag))
	statePath := viper.GetString(cmdUtils.SessionFlag)
	regCode := viper.GetString(cmdUtils.RegCodeFlag)

	// Load client state and identity------------------------------------------
	net := cmdUtils.LoadOrInitCmix(statePass, statePath, regCode, cmixParams)
	identity := cmdUtils.LoadOrInitReceptionIdentity(forceLegacy, net)

	// Save contact file-------------------------------------------------------
	cmdUtils.WriteContact(identity.GetContact())

	// Start Server
	if viper.GetBool(ConnectionAuthenticatedFlag) {
		secureConnServer(net, identity, e2eParams)
	} else {
		insecureConnServer(net, identity, e2eParams)
	}
}

// startClient is a helper function which initializes a connection client.
func startClient(cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams) {
	// Start Client
	if viper.GetBool(ConnectionAuthenticatedFlag) {
		secureConnClient(cmixParams, e2eParams)
	} else {
		insecureConnClient(cmixParams, e2eParams)
	}
}
