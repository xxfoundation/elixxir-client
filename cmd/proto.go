///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/xx_network/primitives/utils"
	"io/ioutil"
	"time"
)

var protoCmd = &cobra.Command{
	Use:   "proto",
	Short: "Load client with a proto client JSON file.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// If output path is specified, only write to file
		var client *api.Client
		jww.INFO.Printf("In proto user path")
		protoOutputPath := viper.GetString("protoUserOut")
		if protoOutputPath != "" {
			client = initClient()


			jsonBytes, err := client.ConstructProtoUerFile()
			if err != nil {
				jww.FATAL.Panicf("Failed to construct proto user file: %v", err)
			}

			err = utils.WriteFileDef(protoOutputPath, jsonBytes)
			if err != nil {
				jww.FATAL.Panicf("Failed to write proto user to file: %v", err)
			}
		} else {
			jww.INFO.Printf("Loading proto client")

			client = loadProtoClient()
		}
	},
}

func loadProtoClient() *api.Client {
	protoUserPath := viper.GetString("protoUserPath")
	protoUserFile, err := utils.ReadFile(protoUserPath)
	if err != nil {
		jww.FATAL.Panicf("Failed to read proto user: %v", err)
	}

	pass := viper.GetString("password")
	storeDir := viper.GetString("session")

	netParams := params.GetDefaultNetwork()
	netParams.E2EParams.MinKeys = uint16(viper.GetUint("e2eMinKeys"))
	netParams.E2EParams.MaxKeys = uint16(viper.GetUint("e2eMaxKeys"))
	netParams.E2EParams.NumRekeys = uint16(
		viper.GetUint("e2eNumReKeys"))
	netParams.ForceHistoricalRounds = viper.GetBool("forceHistoricalRounds")
	netParams.FastPolling = viper.GetBool("slowPolling")
	netParams.ForceMessagePickupRetry = viper.GetBool("forceMessagePickupRetry")
	if netParams.ForceMessagePickupRetry {
		period := 3 * time.Second
		jww.INFO.Printf("Setting Uncheck Round Period to %v", period)
		netParams.UncheckRoundPeriod = period
	}
	netParams.VerboseRoundTracking = viper.GetBool("verboseRoundTracking")

	// Load NDF
	ndfPath := viper.GetString("ndf")
	ndfJSON, err := ioutil.ReadFile(ndfPath)
	if err != nil {
		jww.FATAL.Panicf(err.Error())
	}

	jww.INFO.Printf("login with proto")

	client, err := api.LoginWithProtoClient(storeDir, []byte(pass),
		protoUserFile, string(ndfJSON), netParams)
	if err != nil {
		jww.FATAL.Panicf("Failed to login: %v", err)
	}

	return client
}

// init is the initialization function for Cobra which defines commands
// and flags.
func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.AddCommand(protoCmd)
}
