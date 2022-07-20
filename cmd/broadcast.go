package cmd

import (
	"github.com/spf13/cobra"
	broadCmd "gitlab.com/elixxir/client/broadcast/cmd"
	cmdUtils "gitlab.com/elixxir/client/cmdUtils"
)

// broadcastCmd is the broadcast subcommand that allows for sending and receiving
// to broadcast messages.
var broadcastCmd = &cobra.Command{
	Use:   "broadcast",
	Short: "Send broadcast messages",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		broadCmd.Start()
	},
}

func init() {
	// Broadcast subcommand options
	broadcastCmd.Flags().StringP(broadCmd.BroadcastNameFlag, "", "",
		"Symmetric channel name")
	cmdUtils.BindFlagHelper(broadCmd.BroadcastNameFlag, broadcastCmd)

	broadcastCmd.Flags().StringP(broadCmd.BroadcastRsaPubFlag, "", "",
		"Broadcast channel rsa pub key")
	cmdUtils.BindFlagHelper(broadCmd.BroadcastRsaPubFlag, broadcastCmd)

	broadcastCmd.Flags().StringP(broadCmd.BroadcastSaltFlag, "", "",
		"Broadcast channel salt")
	cmdUtils.BindFlagHelper(broadCmd.BroadcastSaltFlag, broadcastCmd)

	broadcastCmd.Flags().StringP(broadCmd.BroadcastDescriptionFlag, "", "",
		"Broadcast channel description")
	cmdUtils.BindFlagHelper(broadCmd.BroadcastDescriptionFlag, broadcastCmd)

	broadcastCmd.Flags().StringP(broadCmd.BroadcastChanPathFlag, "", "",
		"Broadcast channel output path")
	cmdUtils.BindFlagHelper(broadCmd.BroadcastChanPathFlag, broadcastCmd)

	broadcastCmd.Flags().StringP(broadCmd.BroadcastKeyPathFlag, "", "",
		"Broadcast channel private key output path")
	cmdUtils.BindFlagHelper(broadCmd.BroadcastKeyPathFlag, broadcastCmd)

	broadcastCmd.Flags().BoolP(broadCmd.BroadcastNewFlag, "", false,
		"Create new broadcast channel")
	cmdUtils.BindFlagHelper(broadCmd.BroadcastNewFlag, broadcastCmd)

	broadcastCmd.Flags().StringP(broadCmd.BroadcastSymmetricFlag, "", "",
		"Send symmetric broadcast message")
	cmdUtils.BindFlagHelper(broadCmd.BroadcastSymmetricFlag, broadcastCmd)

	broadcastCmd.Flags().StringP(broadCmd.BroadcastAsymmetricFlag, "", "",
		"Send asymmetric broadcast message (must be used with keyPath)")
	cmdUtils.BindFlagHelper(broadCmd.BroadcastAsymmetricFlag, broadcastCmd)

	rootCmd.AddCommand(broadcastCmd)
}
