////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package cmd initializes the CLI and config parsers as well as the logger.
package cmd

import (
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/privategrity/client/api"
	"os"
)

var verbose bool
var userId int
var destinationUserId int
var serverAddr string
var message string

// Execute adds all child commands to the root command and sets flags
// appropriately.  This is called by main.main(). It only needs to
// happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		jww.ERROR.Println(err)
		os.Exit(1)
	}
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "client",
	Short: "Runs a client for cMix anonymous communication platform",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// Main client run function
		api.Login(userId, serverAddr)
		api.Send(destinationUserId, message)
		// Loop until we get a message, then print and exit
		for {
			msg := api.TryReceive()
			if msg != "" {
				jww.INFO.Printf("Message Received: %s", msg)
				break
			}
		}
	},
}

// init is the initialization function for Cobra which defines commands
// and flags.
func init() {
	cobra.OnInitialize(initConfig, initLog)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false,
		"Verbose mode for debugging")
	rootCmd.PersistentFlags().IntVarP(&userId, "userid", "i", 0,
		"UserID to sign in as")
	rootCmd.MarkPersistentFlagRequired("userid")
	rootCmd.PersistentFlags().StringVarP(&serverAddr, "serveraddr", "s", "",
		"Server address to send messages to")
	rootCmd.MarkPersistentFlagRequired("serveraddr")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.Flags().StringVarP(&message, "message", "m", "", "Message to send")
	rootCmd.PersistentFlags().IntVarP(&destinationUserId, "destid", "d", 0,
		"UserID to send message to")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {}

// initLog initializes logging thresholds and the log path.
func initLog() {
	// If verbose flag set then log more info for debugging
	if verbose || viper.GetBool("verbose") {
		jww.SetLogThreshold(jww.LevelDebug)
		jww.SetStdoutThreshold(jww.LevelDebug)
	} else {
		jww.SetLogThreshold(jww.LevelInfo)
		jww.SetStdoutThreshold(jww.LevelInfo)
	}
	// Create log file, overwrites if existing
	logPath := viper.GetString("logPath")
	logFile, err := os.Create(logPath)
	if err != nil {
		jww.WARN.Println("Invalid or missing log path, default path used.")
	} else {
		jww.SetLogOutput(logFile)
	}
}
