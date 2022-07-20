////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"github.com/spf13/cobra"
	fileTransferCmd "gitlab.com/elixxir/client/fileTransfer/cmd"
)

// ftCmd starts the file transfer manager and allows the sending and receiving
// of files.
var ftCmd = &cobra.Command{
	Use:   "fileTransfer",
	Short: "Send and receive file for cMix client",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fileTransferCmd.Start()
	},
}

////////////////////////////////////////////////////////////////////////////////
// Command Line Flags                                                         //
////////////////////////////////////////////////////////////////////////////////

// init initializes commands and flags for Cobra.
func init() {
	ftCmd.Flags().String(fileTransferCmd.FileSendFlag, "",
		"Sends a file to a recipient with the contact file at this path.")
	BindFlagHelper(fileTransferCmd.FileSendFlag, ftCmd)

	ftCmd.Flags().String(fileTransferCmd.FilePathFlag, "",
		"The path to the file to send. Also used as the file name.")
	BindFlagHelper(fileTransferCmd.FilePathFlag, ftCmd)

	ftCmd.Flags().String(fileTransferCmd.FileTypeFlag, "txt",
		"8-byte file type.")
	BindFlagHelper(fileTransferCmd.FileTypeFlag, ftCmd)

	ftCmd.Flags().String(fileTransferCmd.FilePreviewPathFlag, "",
		"The path to the file preview to send. Set either this flag or "+
			"filePreviewString.")
	BindFlagHelper(fileTransferCmd.FilePreviewPathFlag, ftCmd)

	ftCmd.Flags().String(fileTransferCmd.FilePreviewStringFlag, "",
		"File preview data. Set either this flag or filePreviewPath.")
	BindFlagHelper(fileTransferCmd.FilePreviewStringFlag, ftCmd)

	ftCmd.Flags().Int(fileTransferCmd.FileMaxThroughputFlag, 1000,
		"Maximum data transfer speed to send file parts (in bytes per second)")
	BindFlagHelper(fileTransferCmd.FileMaxThroughputFlag, ftCmd)

	ftCmd.Flags().Float64(fileTransferCmd.FileRetry, 0.5,
		"Retry rate.")
	BindFlagHelper(fileTransferCmd.FileRetry, ftCmd)

	rootCmd.AddCommand(ftCmd)
}
