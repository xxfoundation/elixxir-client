///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// precan.go handles functions for precan users, which are not usable
// unless you are on a localized test network.

package cmd

import (
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/xx_network/primitives/id"
	"io/ioutil"
)

// precanInitCmd creates a new precanned client object.
var precanInitCmd = &cobra.Command{
	Use:   "precan",
	Short: "Initialize a precanned client",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		precanId := viper.GetUint(sendIdFlag)
		storePassword := parsePassword(viper.GetString(passwordFlag))
		storeDir := viper.GetString(sessionFlag)
		initLog(viper.GetUint(logLevelFlag), viper.GetString(logFlag))
		initPrecan(precanId, storePassword, storeDir)
	},
}

// precanCmd loads an existing precanned client from storage. This command will fail
// if precanInitCmd was not called previously to initialize state.
var precanCmd = &cobra.Command{
	Use:   "precan",
	Short: "Initialize a precanned client",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, _ []string) {
		storePassword := parsePassword(viper.GetString(passwordFlag))
		storeDir := viper.GetString(sessionFlag)
		cmixParams, e2eParams := initParams()
		authCbs := makeAuthCallbacks(
			viper.GetBool(unsafeChannelCreationFlag), e2eParams)
		initLog(viper.GetUint(logLevelFlag), viper.GetString(logFlag))
		loadPrecan(storePassword, storeDir, cmixParams, e2eParams, authCbs)
		// todo: do precan specific operations (take precan responsibility away from root.go)
	},
}

// initPrecan initializes a
func initPrecan(precanId uint, password []byte, storeDir string) (*xxdk.Cmix, xxdk.ReceptionIdentity) {
	// Initialize from scratch
	ndfJson, err := ioutil.ReadFile(viper.GetString(ndfFlag))
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	err = xxdk.NewPrecannedClient(precanId, string(ndfJson), storeDir, password)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	// Initialize from storage
	net, err := xxdk.OpenCmix(storeDir, password)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	identity, err := xxdk.MakeLegacyReceptionIdentity(net)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	err = xxdk.StoreReceptionIdentity(identityStorageKey, identity, net)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	return net, identity
}

func loadPrecan(password []byte, storeDir string, cmixParams xxdk.CMIXParams,
	e2eParams xxdk.E2EParams, cbs xxdk.AuthCallbacks) *xxdk.E2e {
	jww.INFO.Printf("Using Precanned sender")

	// Initialize from storage
	net, err := xxdk.LoadCmix(storeDir, password, cmixParams)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	// Load or initialize xxdk.ReceptionIdentity storage
	identity, err := xxdk.LoadReceptionIdentity(identityStorageKey, net)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	messenger, err := xxdk.Login(net, cbs, identity, e2eParams)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	return messenger
}

func isPrecanID(id *id.ID) bool {
	// check if precanned
	rBytes := id.Bytes()
	for i := 0; i < 32; i++ {
		if i != 7 && rBytes[i] != 0 {
			return false
		}
	}
	if rBytes[7] != byte(0) && rBytes[7] <= byte(40) {
		return true
	}
	return false
}

func getPrecanID(recipientID *id.ID) uint {
	return uint(recipientID.Bytes()[7])
}

func addPrecanAuthenticatedChannel(messenger *xxdk.E2e, recipientID *id.ID,
	recipient contact.Contact) {
	jww.WARN.Printf("Precanned user id detected: %s", recipientID)
	preUsr, err := messenger.MakePrecannedAuthenticatedChannel(
		getPrecanID(recipientID))
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	// Sanity check, make sure user id's haven't changed
	preBytes := preUsr.ID.Bytes()
	idBytes := recipientID.Bytes()
	for i := 0; i < len(preBytes); i++ {
		if idBytes[i] != preBytes[i] {
			jww.FATAL.Panicf("no id match: %v %v",
				preBytes, idBytes)
		}
	}
}

func init() {
	initCmd.AddCommand(precanInitCmd)
	rootCmd.AddCommand(precanCmd)

	// todo: sendIdFlag can probably be brought into this subcommand.
	//  This is blocked until once root.go has been refactored to no longer
	//  have usages of sendId and sendId is self-contained in this file.
}
