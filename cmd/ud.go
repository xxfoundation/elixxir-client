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
	cmdUtils "gitlab.com/elixxir/client/cmdUtils"
	udCommand "gitlab.com/elixxir/client/ud/cmd"
)

// udCmd is the user discovery subcommand, which allows for user lookup,
// registration, and search. This basically runs a client for these functions
// with the UD module enabled. Normally, clients do not need it so it is not
// loaded for the rest of the commands.
var udCmd = &cobra.Command{
	Use:   "ud",
	Short: "Register for and search users using the xx network user discovery service.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		udCommand.Start()
	},
}

func init() {
	// User Discovery subcommand Options
	udCmd.Flags().StringP(udCommand.UdRegisterFlag, "r", "",
		"Register this user with user discovery.")
	cmdUtils.BindFlagHelper(udCommand.UdRegisterFlag, udCmd)

	udCmd.Flags().StringP(udCommand.UdRemoveFlag, "", "",
		"Remove this user with user discovery.")
	cmdUtils.BindFlagHelper(udCommand.UdRemoveFlag, udCmd)

	udCmd.Flags().String(udCommand.UdAddPhoneFlag, "",
		"Add phone number to existing user registration.")
	cmdUtils.BindFlagHelper(udCommand.UdAddPhoneFlag, udCmd)

	udCmd.Flags().StringP(udCommand.UdAddEmailFlag, "e", "",
		"Add email to existing user registration.")
	cmdUtils.BindFlagHelper(udCommand.UdAddEmailFlag, udCmd)

	udCmd.Flags().String(udCommand.UdConfirmFlag, "",
		"Confirm fact with confirmation ID.")
	cmdUtils.BindFlagHelper(udCommand.UdConfirmFlag, udCmd)

	udCmd.Flags().StringP(udCommand.UdLookupFlag, "u", "",
		"Look up user ID. Use '0x' or 'b64:' for hex and base64 representations.")
	cmdUtils.BindFlagHelper(udCommand.UdLookupFlag, udCmd)

	udCmd.Flags().String(udCommand.UdSearchUsernameFlag, "",
		"Search for users with this username.")
	cmdUtils.BindFlagHelper(udCommand.UdSearchUsernameFlag, udCmd)

	udCmd.Flags().String(udCommand.UdSearchEmailFlag, "",
		"Search for users with this email address.")
	cmdUtils.BindFlagHelper(udCommand.UdSearchEmailFlag, udCmd)

	udCmd.Flags().String(udCommand.UdSearchPhoneFlag, "",
		"Search for users with this email address.")
	cmdUtils.BindFlagHelper(udCommand.UdSearchPhoneFlag, udCmd)

	udCmd.Flags().String(udCommand.UdBatchAddFlag, "",
		"Path to JSON marshalled slice of partner IDs that will be looked up on UD.")
	cmdUtils.BindFlagHelper(udCommand.UdBatchAddFlag, udCmd)

	rootCmd.AddCommand(udCmd)
}
