///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Package cmd initializes the CLI and config parsers as well as the logger.
package cmd

import (
	"fmt"

	"io/fs"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/xxdk"
)

// initCmd creates a new user object with the given NDF
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a user ID but do not connect to the network",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Handle userid-prefix argument
		storePassword := parsePassword(viper.GetString(passwordFlag))
		storeDir := viper.GetString(sessionFlag)
		regCode := viper.GetString(regCodeFlag)

		// Initialize from scratch
		ndfJson, err := ioutil.ReadFile(viper.GetString(ndfFlag))
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		err = xxdk.NewCmix(string(ndfJson), storeDir, storePassword, regCode)
		net, err := xxdk.OpenCmix(storeDir, storePassword)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		identity, err := xxdk.MakeReceptionIdentity(net)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		err = xxdk.StoreReceptionIdentity(identityStorageKey, identity, net)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		jww.INFO.Printf("User: %s", identity.ID)
		writeContact(identity.GetContact())

		// NOTE: DO NOT REMOVE THIS LINE. YOU WILL BREAK INTEGRATION
		fmt.Printf("%s\n", identity.ID)
	},
}

func init() {
	initCmd.Flags().StringP(userIdPrefixFlag, "", "",
		"Desired prefix of userID to brute force when running init command. Prepend (?i) for case-insensitive. Only Base64 characters are valid.")
	bindFlagHelper(userIdPrefixFlag, initCmd)

	rootCmd.AddCommand(initCmd)
}

// loadOrInitCmix will build a new xxdk.Cmix from existing storage
// or from a new storage that it will create if none already exists
func loadOrInitCmix(password []byte, storeDir, regCode string,
	cmixParams xxdk.CMIXParams) *xxdk.Cmix {
	// create a new cMix if none exist
	if _, err := os.Stat(storeDir); errors.Is(err, fs.ErrNotExist) {
		// Initialize from scratch
		ndfJson, err := ioutil.ReadFile(viper.GetString(ndfFlag))
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		err = xxdk.NewCmix(string(ndfJson), storeDir, password, regCode)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
	}

	// Initialize from storage
	net, err := xxdk.LoadCmix(storeDir, password, cmixParams)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	return net
}

// loadOrInitReceptionIdentity will build a new xxdk.ReceptionIdentity from existing storage
// or from a new storage that it will create if none already exists
func loadOrInitReceptionIdentity(forceLegacy bool, net *xxdk.Cmix) xxdk.ReceptionIdentity {
	// Load or initialize xxdk.ReceptionIdentity storage
	identity, err := xxdk.LoadReceptionIdentity(identityStorageKey, net)
	if err != nil {
		if forceLegacy {
			jww.INFO.Printf("Forcing legacy sender")
			identity, err = xxdk.MakeLegacyReceptionIdentity(net)
		} else {
			identity, err = xxdk.MakeReceptionIdentity(net)
		}
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		err = xxdk.StoreReceptionIdentity(identityStorageKey, identity, net)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
	}
	return identity
}

// loadOrInitUser will build a new xxdk.E2e from existing storage
// or from a new storage that it will create if none already exists
func loadOrInitUser(forceLegacy bool, password []byte, storeDir, regCode string,
	cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams, cbs xxdk.AuthCallbacks) *xxdk.E2e {
	jww.INFO.Printf("Using normal sender")

	net := loadOrInitCmix(password, storeDir, regCode, cmixParams)
	identity := loadOrInitReceptionIdentity(forceLegacy, net)

	user, err := xxdk.Login(net, cbs, identity, e2eParams)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	return user
}

// loadOrInitEphemeral will build a new ephemeral xxdk.E2e.
func loadOrInitEphemeral(forceLegacy bool, password []byte, storeDir, regCode string,
	cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams, cbs xxdk.AuthCallbacks) *xxdk.E2e {
	jww.INFO.Printf("Using ephemeral sender")

	net := loadOrInitCmix(password, storeDir, regCode, cmixParams)
	identity := loadOrInitReceptionIdentity(forceLegacy, net)

	user, err := xxdk.LoginEphemeral(net, cbs, identity, e2eParams)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	return user
}

// loadOrInitVanity will build a new xxdk.E2e from existing storage
// or from a new storage that it will create if none already exists
func loadOrInitVanity(password []byte, storeDir, regCode, userIdPrefix string,
	cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams, cbs xxdk.AuthCallbacks) *xxdk.E2e {
	jww.INFO.Printf("Using Vanity sender")

	// create a new cMix if none exist
	if _, err := os.Stat(storeDir); errors.Is(err, fs.ErrNotExist) {
		// Initialize from scratch
		ndfJson, err := ioutil.ReadFile(viper.GetString(ndfFlag))
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		err = xxdk.NewVanityCmix(string(ndfJson), storeDir,
			password, regCode, userIdPrefix)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
	}
	// Initialize from storage
	net, err := xxdk.LoadCmix(storeDir, password, cmixParams)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	// Load or initialize xxdk.ReceptionIdentity storage
	identity, err := xxdk.LoadReceptionIdentity(identityStorageKey, net)
	if err != nil {
		identity, err = xxdk.MakeLegacyReceptionIdentity(net)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		err = xxdk.StoreReceptionIdentity(identityStorageKey, identity, net)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
	}

	user, err := xxdk.Login(net, cbs, identity, e2eParams)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	return user
}
