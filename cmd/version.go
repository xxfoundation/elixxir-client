///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Handles command-line version functionality

package cmd

import (
	"fmt"
	cmdUtils "gitlab.com/elixxir/client/cmdUtils"

	"github.com/spf13/cobra"
	"gitlab.com/xx_network/primitives/utils"
)

// Change this value to set the version for this build
const currentVersion = "4.2.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version and dependency information for the Elixxir binary",
	Long:  `Print the version and dependency information for the Elixxir binary`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf(cmdUtils.Version())
	},
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generates version and dependency information for the Elixxir binary",
	Long:  `Generates version and dependency information for the Elixxir binary`,
	Run: func(cmd *cobra.Command, args []string) {
		utils.GenerateVersionFile(currentVersion)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(generateCmd)
}
