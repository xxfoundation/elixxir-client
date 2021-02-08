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
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
)

// initCmd creates a new user object with the given NDF
var initCmd = &cobra.Command{
	Use:   "init",
	Short: ("Initialize a user ID but do not connect to the network"),
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		client := createClient()
		user := client.GetUser()
		jww.INFO.Printf("User: %s", user.ID)
		writeContact(user.GetContact())
		fmt.Printf("%s\n", user.ID)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
