///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Package cmd initializes the CLI and config parsers as well as the logger.
package cmd

import (
	"github.com/pkg/errors"
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
	jww.INFO.Printf("Using normal sender")

	// create a new client if none exist
	var net *xxdk.Cmix
	var identity xxdk.ReceptionIdentity
	if _, err := os.Stat(storeDir); errors.Is(err, fs.ErrNotExist) {
		// Initialize from scratch
		ndfJson, err := ioutil.ReadFile(viper.GetString("ndf"))
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		err = xxdk.NewCmix(string(ndfJson), storeDir, password, regCode)
		net, err = xxdk.LoadCmix(storeDir, password, cmixParams)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		identity, err = xxdk.MakeReceptionIdentity(net)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		err = xxdk.StoreReceptionIdentity(identityStorageKey, identity, net)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
	} else {
		// Initialize from storage
		net, err = xxdk.LoadCmix(storeDir, password, cmixParams)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
		identity, err = xxdk.LoadReceptionIdentity(identityStorageKey, net)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
	}

	client, err := xxdk.Login(net, authCbs, identity, e2eParams)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	return client
}

// loadOrInitVanity will build a new xxdk.E2e from existing storage
// or from a new storage that it will create if none already exists
func loadOrInitVanity(password []byte, storeDir, regCode, userIdPrefix string,
	cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams) *xxdk.E2e {
	jww.INFO.Printf("Using Vanity sender")

	// create a new client if none exist
	var net *xxdk.Cmix
	var identity xxdk.ReceptionIdentity
	if _, err := os.Stat(storeDir); errors.Is(err, fs.ErrNotExist) {
		// Initialize from scratch
		ndfJson, err := ioutil.ReadFile(viper.GetString("ndf"))
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		err = xxdk.NewVanityClient(string(ndfJson), storeDir,
			password, regCode, userIdPrefix)
		net, err = xxdk.LoadCmix(storeDir, password, cmixParams)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		identity, err = xxdk.MakeLegacyReceptionIdentity(net)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		err = xxdk.StoreReceptionIdentity(identityStorageKey, identity, net)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
	} else {
		// Initialize from storage
		net, err = xxdk.LoadCmix(storeDir, password, cmixParams)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
		identity, err = xxdk.LoadReceptionIdentity(identityStorageKey, net)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
	}

	messenger, err := xxdk.Login(net, authCbs, identity, e2eParams)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	return messenger
}
