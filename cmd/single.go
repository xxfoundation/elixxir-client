///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Package cmd initializes the CLI and config parsers as well as the logger.
package cmd

import (
	cmdUtils "gitlab.com/elixxir/client/cmdUtils"
	"time"

	"github.com/spf13/cobra"
	singleCommand "gitlab.com/elixxir/client/single/cmd"
)

// singleCmd is the single-use subcommand that allows for sending and responding
// to single-use messages.
var singleCmd = &cobra.Command{
	Use:   "single",
	Short: "Send and respond to single-use messages.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		singleCommand.Start()
	},
}

func init() {
	// Single-use subcommand options
	singleCmd.Flags().Bool(singleCommand.SingleSendFlag, false, "Sends a single-use message.")
	cmdUtils.BindFlagHelper(singleCommand.SingleSendFlag, singleCmd)

	singleCmd.Flags().Bool(singleCommand.SingleReplyFlag, false,
		"Listens for a single-use message and sends a reply.")
	cmdUtils.BindFlagHelper(singleCommand.SingleSendFlag, singleCmd)

	singleCmd.Flags().StringP(singleCommand.SingleContactFlag, "c", "",
		"Path to contact file to send message to.")
	cmdUtils.BindFlagHelper(singleCommand.SingleContactFlag, singleCmd)

	singleCmd.Flags().StringP(singleCommand.SingleTagFlag, "", "testTag",
		"The tag that specifies the callback to trigger on reception.")
	cmdUtils.BindFlagHelper(singleCommand.SingleTagFlag, singleCmd)

	singleCmd.Flags().Uint8(singleCommand.SingleMaxMessagesFlag, 1,
		"The max number of single-use response messages.")
	cmdUtils.BindFlagHelper(singleCommand.SingleMaxMessagesFlag, singleCmd)

	singleCmd.Flags().DurationP(singleCommand.SingleTimeoutFlag, "t", 30*time.Second,
		"Duration before stopping to wait for single-use message.")
	cmdUtils.BindFlagHelper(singleCommand.SingleMaxMessagesFlag, singleCmd)

	rootCmd.AddCommand(singleCmd)
}
