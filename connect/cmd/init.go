package cmd

import (
	jww "github.com/spf13/jwalterweatherman"
	cmd "gitlab.com/elixxir/client/cmdUtils"
	"gitlab.com/elixxir/client/xxdk"
)

// loadOrInitEphemeral will build a new ephemeral xxdk.E2e.
func loadOrInitEphemeral(forceLegacy bool, password []byte, storeDir, regCode string,
	cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams, cbs xxdk.AuthCallbacks) *xxdk.E2e {
	jww.INFO.Printf("Using ephemeral sender")

	net := cmd.LoadOrInitCmix(password, storeDir, regCode, cmixParams)
	identity := cmd.LoadOrInitReceptionIdentity(forceLegacy, net)

	messenger, err := xxdk.LoginEphemeral(net, cbs, identity, e2eParams)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	return messenger
}
