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
	cmdUtils "gitlab.com/elixxir/client/cmdUtils"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/xxdk"
	"io/ioutil"
)

// initCmd creates a new user object with the given NDF
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a user ID but do not connect to the network",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Handle userid-prefix argument
		storePassword := cmdUtils.ParsePassword(viper.GetString(cmdUtils.PasswordFlag))
		storeDir := viper.GetString(cmdUtils.SessionFlag)
		regCode := viper.GetString(cmdUtils.RegCodeFlag)

		// Initialize from scratch
		ndfJson, err := ioutil.ReadFile(viper.GetString(cmdUtils.NdfFlag))
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
		cmdUtils.WriteContact(identity.GetContact())

		// NOTE: DO NOT REMOVE THIS LINE. YOU WILL BREAK INTEGRATION
		fmt.Printf("%s\n", identity.ID)
	},
}

func init() {
	initCmd.Flags().StringP(cmdUtils.UserIdPrefixFlag, "", "",
		"Desired prefix of userID to brute force when running init command. "+
			"Prepend (?i) for case-insensitive. Only Base64 characters are valid.")
	cmdUtils.BindFlagHelper(cmdUtils.UserIdPrefixFlag, initCmd)

	rootCmd.AddCommand(initCmd)
}
