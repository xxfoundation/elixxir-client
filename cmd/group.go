///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// The group subcommand allows creation and sending messages to groups

package cmd

import (
	cmdUtils "gitlab.com/elixxir/client/cmdUtils"
	"time"

	"github.com/spf13/cobra"
	grpCmd "gitlab.com/elixxir/client/groupChat/cmd"
)

// groupCmd represents the base command when called without any subcommands
var groupCmd = &cobra.Command{
	Use:   "group",
	Short: "Group commands for cMix client",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		grpCmd.Start()
	},
}

func init() {
	groupCmd.Flags().String(grpCmd.GroupCreateFlag, "",
		"Create a group with from the list of contact file paths.")
	cmdUtils.BindFlagHelper(grpCmd.GroupCreateFlag, groupCmd)

	groupCmd.Flags().String(grpCmd.GroupNameFlag, "Group Name",
		"The name of the new group to create.")
	cmdUtils.BindFlagHelper(grpCmd.GroupNameFlag, groupCmd)

	groupCmd.Flags().String(grpCmd.GroupResendFlag, "",
		"Resend invites for all users in this group ID.")
	cmdUtils.BindFlagHelper(grpCmd.GroupResendFlag, groupCmd)

	groupCmd.Flags().Bool(grpCmd.GroupJoinFlag, false,
		"Waits for group request joins the group.")
	cmdUtils.BindFlagHelper(grpCmd.GroupJoinFlag, groupCmd)

	groupCmd.Flags().String(grpCmd.GroupLeaveFlag, "",
		"Leave this group ID.")
	cmdUtils.BindFlagHelper(grpCmd.GroupLeaveFlag, groupCmd)

	groupCmd.Flags().String(grpCmd.GroupSendMessageFlag, "",
		"Send message to this group ID.")
	cmdUtils.BindFlagHelper(grpCmd.GroupSendMessageFlag, groupCmd)

	groupCmd.Flags().Uint(grpCmd.GroupWaitFlag, 0,
		"Waits for number of messages to be received.")
	cmdUtils.BindFlagHelper(grpCmd.GroupWaitFlag, groupCmd)

	groupCmd.Flags().Duration(grpCmd.GroupReceiveTimeoutFlag, time.Minute,
		"Amount of time to wait for a group request or message before timing out.")
	cmdUtils.BindFlagHelper(grpCmd.GroupReceiveTimeoutFlag, groupCmd)

	groupCmd.Flags().Bool(grpCmd.GroupListFlag, false,
		"Prints list all groups to which this client belongs.")
	cmdUtils.BindFlagHelper(grpCmd.GroupListFlag, groupCmd)

	groupCmd.Flags().String(grpCmd.GroupShowFlag, "",
		"Prints the members of this group ID.")
	cmdUtils.BindFlagHelper(grpCmd.GroupShowFlag, groupCmd)

	rootCmd.AddCommand(groupCmd)
}
