////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles command-line version functionality

package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/primitives/utils"
)

// Change this value to set the version for this build
const currentVersion = "1.3.0"

func printVersion() {
	fmt.Printf("Elixxir Client v%s -- %s\n\n", globals.SEMVER, globals.GITVERSION)
	fmt.Printf("Dependencies:\n\n%s\n", globals.DEPENDENCIES)
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
		printVersion()
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
