////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.com/elixxir/client/globals"
)

//go:generate go run gen.go
// The above generates: GITVERSION, GLIDEDEPS, and SEMVER

func init() {
	rootCmd.AddCommand(versionCmd)
}

func printVersion() {
	fmt.Printf("Elixxir Client v%s -- %s\n\n", globals.SEMVER, globals.GITVERSION)
	fmt.Printf("Dependencies:\n\n%s\n", globals.GLIDEDEPS)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Elixxir Client",
	Long: `Print the version number of Elixxir Client. This also prints
the glide cache versions of all of its dependencies.`,
	Run: func(cmd *cobra.Command, args []string) {
		printVersion()
	},
}
