////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package cmd initializes the CLI and config parsers as well as the logger.
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/privategrity/client/api"
	"os"
	"time"
	"gitlab.com/privategrity/client/globals"
)

var verbose bool
var userId uint64
var destinationUserId uint64
var serverAddr string
var message string
var numNodes int
var sessionFile string

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

		var success bool

		register := false

		if sessionFile == ""{
			success = api.InitClient(globals.RamStorage{},"")
			if !success{
				fmt.Println("Could Not Initilize Ram Storage")
				return
			}
			register = true
		}else{

			_, err := os.Stat(sessionFile)

			if err!=nil{
				if os.IsNotExist(err){
					register = true
				} else{
					fmt.Println("Error with file path: %v", err.Error())
				}
			}


			success = api.InitClient(globals.DefaultStorage{},sessionFile)

			if !success{
				fmt.Println("Could Not Initilize OS Storage")
				return
			}
		}

		if register{
			UID := api.Register(globals.UserHash(userId),
				"",serverAddr, 	numNodes)
			if UID==0{
				fmt.Println("Could Not Register User")
				return
			}
		}

		success = api.Login(userId)

		if !success {
			fmt.Println("Could Not Log In ")
			return
		}

		fmt.Printf("Sending Message to %d: %s\n", destinationUserId, message)

		api.Send(api.APIMessage{userId,message,destinationUserId})
		// Loop until we get a message, then print and exit
		for {
			msg := api.TryReceive()
			if msg.Payload != "" {
				fmt.Printf("Message from %v Received: %s\n", msg.Sender, msg.Payload)
				break
			}
			time.Sleep(200 * time.Millisecond)
		}

		success = api.Logout()

		if !success{
			fmt.Printf("Could not logout")
			return
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
	rootCmd.PersistentFlags().Uint64VarP(&userId, "userid", "i", 0,
		"UserID to sign in as")
	rootCmd.MarkPersistentFlagRequired("userid")
	rootCmd.PersistentFlags().StringVarP(&serverAddr, "serveraddr", "s", "",
		"Server address to send messages to")
	rootCmd.MarkPersistentFlagRequired("serveraddr")
	// TODO: support this negotiating separate keys with different servers
	rootCmd.PersistentFlags().IntVarP(&numNodes, "numnodes", "n", 1,
		"The number of servers in the network that the client is"+
			" connecting to")

	rootCmd.PersistentFlags().StringVarP(&sessionFile,"sessionfile", "f",
		"", "Passes a file path for loading a session.  " +
			"If the file doesn't exist the code will register the user and" +
				" store it there.  If not passed the session will be stored" +
					" to ram and lost when the cli finishes")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().StringVarP(&message, "message", "m", "", "Message to send")
	rootCmd.PersistentFlags().Uint64VarP(&destinationUserId, "destid", "d", 0,
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
	if viper.Get("logPath") != nil {
		// Create log file, overwrites if existing
		logPath := viper.GetString("logPath")
		logFile, err := os.Create(logPath)
		if err != nil {
			jww.WARN.Println("Invalid or missing log path, default path used.")
		} else {
			jww.SetLogOutput(logFile)
		}
	}
}
