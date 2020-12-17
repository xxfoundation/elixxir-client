///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Package cmd initializes the CLI and config parsers as well as the logger.
package cmd

import (
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
)

// udCmd user discovery subcommand, allowing user lookup and registration for
// allowing others to search.
// This basically runs a client for these functions with the UD module enabled.
// Normally, clients don't need it so it is not loaded for the rest of the
// commands.
var udCmd = &cobra.Command{
	Use: "ud",
	Short: ("Register for & search users using the xxnet user discovery " +
		"service"),
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// UD Command does nothing right now.
		jww.INFO.Printf("Hello, World")
	},
}

func init() {
	// User Discovery subcommand Options
	udCmd.Flags().StringP("register", "r", "",
		"Register this user with user discovery")
	viper.BindPFlag("register",
		udCmd.Flags().Lookup("register"))
	udCmd.Flags().StringP("addphone", "", "",
		"Add phone number to existing user registration.")
	viper.BindPFlag("addphone", udCmd.Flags().Lookup("addphone"))
	udCmd.Flags().StringP("addemail", "e", "",
		"Add email to existing user registration.")
	viper.BindPFlag("addemail", udCmd.Flags().Lookup("addemail"))

	udCmd.Flags().StringP("lookup", "u", "",
		"Look up user ID. Use '0x' or 'b64:' for hex and base64 "+
			"representations")
	viper.BindPFlag("lookup", udCmd.Flags().Lookup("lookup"))
	udCmd.Flags().StringP("searchusername", "", "",
		"Search for users with this username")
	viper.BindPFlag("searchusername",
		udCmd.Flags().Lookup("searchusername"))
	udCmd.Flags().StringP("searchemail", "", "",
		"Search for users with this email address")
	viper.BindPFlag("searchemail",
		udCmd.Flags().Lookup("searchemail"))
	udCmd.Flags().StringP("searchphone", "", "",
		"Search for users with this email address")
	viper.BindPFlag("searchphone",
		udCmd.Flags().Lookup("searchphone"))

	rootCmd.AddCommand(udCmd)
}
