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
	"gitlab.com/elixxir/client/xxdk"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
)

// initCmd creates a new user object with the given NDF
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a user ID but do not connect to the network",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		client := createClient()
		e2e, err := xxdk.LoadOrInitE2e(client)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
		user := client.GetUser()
		user.E2eDhPublicKey = e2e.GetHistoricalDHPubkey()

		jww.INFO.Printf("User: %s", user.ReceptionID)
		writeContact(user.GetContact())
		fmt.Printf("%s\n", user.ReceptionID)
	},
}

func init() {
	initCmd.Flags().StringP("userid-prefix", "", "",
		"Desired prefix of userID to brute force when running init command. Prepend (?i) for case-insensitive. Only Base64 characters are valid.")
	_ = viper.BindPFlag("userid-prefix", initCmd.Flags().Lookup("userid-prefix"))

	rootCmd.AddCommand(initCmd)
}
