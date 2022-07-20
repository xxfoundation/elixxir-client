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

	// Pull flags to start a client
	// todo: other cmd paths don't need to pull this out (albeit they call initE2e only).
	//  see if there's a way to cleanly refactor existing code so we don't have to pull these
	statePass := cmdUtils.ParsePassword(viper.GetString(cmdUtils.PasswordFlag))
	statePath := viper.GetString(cmdUtils.SessionFlag)
	regCode := viper.GetString(cmdUtils.RegCodeFlag)
	cmixParams, e2eParams := cmdUtils.InitParams()
	forceLegacy := viper.GetBool(cmdUtils.ForceLegacyFlag)

	// Start connection instanct
	if viper.GetBool(ConnectionStartServerFlag) {
		startServer(forceLegacy, statePass, statePath, regCode,
			cmixParams, e2eParams)
	} else {
		startClient(forceLegacy, statePass, statePath, regCode,
			cmixParams, e2eParams)
	}
}

// startServer is a helper function which initializes a connection server.
func startServer(forceLegacy bool, statePass []byte, statePath string, regCode string,
	cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams) {
	if viper.GetBool(ConnectionAuthenticatedFlag) {
		secureConnServer(forceLegacy, statePass, statePath, regCode,
			cmixParams, e2eParams)
	} else {
		insecureConnServer(forceLegacy, statePass, statePath, regCode,
			cmixParams, e2eParams)
	}
}

// startClient is a helper function which initializes a connection client.
func startClient(forceLegacy bool, statePass []byte, statePath string, regCode string,
	cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams) {
	if viper.GetBool(ConnectionAuthenticatedFlag) {
		secureConnClient(forceLegacy, statePass, statePath, regCode,
			cmixParams, e2eParams)
	} else {
		insecureConnClient(forceLegacy, statePass, statePath, regCode,
			cmixParams, e2eParams)
	}
}
