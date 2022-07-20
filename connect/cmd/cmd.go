////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/xxdk"
)

// Start is the ingress point for outside packages which want to initialize a CLI
// connection service; clients or servers. Either service can use authentication
// (known as being "secure") if the authenticated flag is provided.
func Start(forceLegacy bool, statePass []byte, statePath string, regCode string,
	cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams) {
	if viper.GetBool(connectionStartServerFlag) {
		startServer(forceLegacy, statePass, statePath, regCode,
			cmixParams, e2eParams)

	} else {
		startClient(forceLegacy, statePass, statePath, regCode,
			cmixParams, e2eParams)
	}
}

// startServer is a helper function which
func startServer(forceLegacy bool, statePass []byte, statePath string, regCode string,
	cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams) {
	if viper.GetBool(connectionAuthenticatedFlag) {
		secureConnServer(forceLegacy, statePass, statePath, regCode,
			cmixParams, e2eParams)
	} else {
		insecureConnServer(forceLegacy, statePass, statePath, regCode,
			cmixParams, e2eParams)
	}
}

func startClient(forceLegacy bool, statePass []byte, statePath string, regCode string,
	cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams) {
	if viper.GetBool(connectionAuthenticatedFlag) {
		secureConnClient(forceLegacy, statePass, statePath, regCode,
			cmixParams, e2eParams)
	} else {
		insecureConnClient(forceLegacy, statePass, statePath, regCode,
			cmixParams, e2eParams)
	}
}
