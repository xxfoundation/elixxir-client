///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Package cmd initializes the CLI and config parsers as well as the logger.
package cmd

import (
<<<<<<< HEAD
	"fmt"

=======
	"github.com/pkg/errors"
>>>>>>> origin/hotfix/RefactorCMD
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/xxdk"
	"io/fs"
	"io/ioutil"
	"os"
)

// initCmd creates a new user object with the given NDF
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a user ID but do not connect to the network",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Handle userid-prefix argument
		storePassword := parsePassword(viper.GetString("password"))
		storeDir := viper.GetString("session")
		regCode := viper.GetString("regcode")

		// Initialize from scratch
		ndfJson, err := ioutil.ReadFile(viper.GetString("ndf"))
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		err = xxdk.NewCmix(string(ndfJson), storeDir, storePassword, regCode)
		baseClient, err := xxdk.OpenCmix(storeDir, storePassword)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		identity, err := xxdk.MakeReceptionIdentity(baseClient)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		err = xxdk.StoreReceptionIdentity(identityStorageKey, identity, baseClient)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		jww.INFO.Printf("User: %s", identity.ID)
		writeContact(identity.GetContact())

		// NOTE: DO NOT REMOVE THIS LINE. YOU WILL BREAK INTEGRATION
		fmt.Printf("%s\n", receptionIdentity.ID)
	},
}

func init() {
	initCmd.Flags().StringP("userid-prefix", "", "",
		"Desired prefix of userID to brute force when running init command. Prepend (?i) for case-insensitive. Only Base64 characters are valid.")
	_ = viper.BindPFlag("userid-prefix", initCmd.Flags().Lookup("userid-prefix"))

	rootCmd.AddCommand(initCmd)
}

// loadOrInitClient will build a new xxdk.E2e from existing storage
// or from a new storage that it will create if none already exists
func loadOrInitClient(password []byte, storeDir, regCode string,
	cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams) *xxdk.E2e {

	// create a new client if none exist
	var baseClient *xxdk.Cmix
	var identity xxdk.ReceptionIdentity
	if _, err := os.Stat(storeDir); errors.Is(err, fs.ErrNotExist) {
		// Initialize from scratch
		ndfJson, err := ioutil.ReadFile(viper.GetString("ndf"))
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		err = xxdk.NewCmix(string(ndfJson), storeDir, password, regCode)
		baseClient, err = xxdk.LoadCmix(storeDir, password, cmixParams)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		identity, err = xxdk.MakeReceptionIdentity(baseClient)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		err = xxdk.StoreReceptionIdentity(identityStorageKey, identity, baseClient)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
	} else {
		// Initialize from storage
		baseClient, err = xxdk.LoadCmix(storeDir, password, cmixParams)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
		identity, err = xxdk.LoadReceptionIdentity(identityStorageKey, baseClient)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
	}

	jww.INFO.Printf("Using Login for normal sender")
	client, err := xxdk.Login(baseClient, authCbs, identity, e2eParams)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	return client
}

// loadOrInitVanity will build a new xxdk.E2e from existing storage
// or from a new storage that it will create if none already exists
func loadOrInitVanity(password []byte, storeDir, regCode, userIdPrefix string,
	cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams) *xxdk.E2e {

	// create a new client if none exist
	var baseClient *xxdk.Cmix
	var identity xxdk.ReceptionIdentity
	if _, err := os.Stat(storeDir); errors.Is(err, fs.ErrNotExist) {
		// Initialize precan from scratch
		ndfJson, err := ioutil.ReadFile(viper.GetString("ndf"))
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		err = xxdk.NewVanityClient(string(ndfJson), storeDir,
			password, regCode, userIdPrefix)
		baseClient, err = xxdk.LoadCmix(storeDir, password, cmixParams)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		// TODO: Get proper identity
		identity, err = xxdk.MakeReceptionIdentity(baseClient)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		err = xxdk.StoreReceptionIdentity(identityStorageKey, identity, baseClient)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
	} else {
		// Initialize precan from storage
		baseClient, err = xxdk.LoadCmix(storeDir, password, cmixParams)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
		identity, err = xxdk.LoadReceptionIdentity(identityStorageKey, baseClient)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
	}

	jww.INFO.Printf("Using LoginLegacy for vanity sender")
	client, err := xxdk.LoginLegacy(baseClient, e2eParams, authCbs)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	return client
}
