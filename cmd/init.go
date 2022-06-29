///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Package cmd initializes the CLI and config parsers as well as the logger.
package cmd

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/xxdk"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const identityStorageKey = "identityStorageKey"

// initCmd creates a new user object with the given NDF
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a user ID but do not connect to the network",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		client := createClient()

		identity, err := xxdk.MakeReceptionIdentity(client)
		if err != nil {
			return
		}

		err = xxdk.StoreReceptionIdentity(identityStorageKey, identity, client)
		if err != nil {
			return
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
