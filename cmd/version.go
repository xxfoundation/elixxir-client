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

	"github.com/spf13/cobra"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/xx_network/primitives/utils"
)

// Change this value to set the version for this build
const currentVersion = "4.2.0"

func Version() string {
	out := fmt.Sprintf("Elixxir Client v%s -- %s\n\n", api.SEMVER,
		api.GITVERSION)
	out += fmt.Sprintf("Dependencies:\n\n%s\n", api.DEPENDENCIES)
	return out
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(generateCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version and dependency information for the Elixxir binary",
	Long:  `Print the version and dependency information for the Elixxir binary`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf(Version())
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
