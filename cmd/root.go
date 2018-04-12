////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package cmd initializes the CLI and config parsers as well as the logger.
package cmd

import (
	"encoding/binary"
	"fmt"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/privategrity/client/api"
	"gitlab.com/privategrity/client/bindings"
	"gitlab.com/privategrity/client/channelbot"
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/crypto/format"
	"os"
	"strings"
	"time"
)

var verbose bool
var userId uint64
var destinationUserId uint64
var serverAddr string
var message string
var numNodes uint
var sessionFile string
var noRatchet bool
var dummyFrequency float64
var nick string
var blockingTransmission bool
var rateLimiting uint32

// Execute adds all child commands to the root command and sets flags
// appropriately.  This is called by main.main(). It only needs to
// happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		jww.ERROR.Println(err)
		os.Exit(1)
	}
}

func sessionInitialization() {
	// Disable ratcheting if the flag is set

	if !blockingTransmission {
		if !noRatchet {
			fmt.Printf("Cannot disable Blocking Transmission with" +
				" Ratcheting turned on\n")
		}
		api.DisableBlockingTransmission()
	}

	bindings.SetRateLimiting(int(rateLimiting))

	if noRatchet {
		bindings.DisableRatchet()
	}

	var err error
	register := false

	//If no session file is passed initialize with RAM Storage
	if sessionFile == "" {
		err = bindings.InitClient(&globals.RamStorage{}, "", nil)
		if err != nil {
			fmt.Printf("Could Not Initialize Ram Storage: %s\n",
				err.Error())
			return
		}
		register = true
	} else {

		//If a session file is passed, check if it's valid
		_, err1 := os.Stat(sessionFile)

		if err1 != nil {
			//If the file does not exist, register a new user
			if os.IsNotExist(err1) {
				register = true
			} else {
				//Fail if any other error is received
				fmt.Printf("Error with file path: %s\n", err1.Error())
			}
		}

		//Initialize client with OS Storage
		err = bindings.InitClient(&globals.DefaultStorage{}, sessionFile, nil)

		if err != nil {
			fmt.Printf("Could Not Initialize OS Storage: %s\n", err.Error())
			return
		}
	}

	//Register a new user if requested
	if register {
		_, err := bindings.Register(
			cyclic.NewIntFromUInt(globals.UserHash(userId)).TextVerbose(
				32, 0),
			nick, serverAddr, int(numNodes))
		if err != nil {
			fmt.Printf("Could Not Register User: %s\n", err.Error())
			return
		}
	}

	//log the user in
	_, err = bindings.Login(
		cyclic.NewIntFromUInt(userId).LeftpadBytes(8))

	if err != nil {
		fmt.Printf("Could Not Log In\n")
		return
	}
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "client",
	Short: "Runs a client for cMix anonymous communication platform",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// Main client run function
		blockingTransmission = !blockingTransmission

		var dummyPeriod time.Duration
		var timer *time.Timer

		// Do calculation for dummy messages if the flag is set
		if dummyFrequency != 0 {
			dummyPeriod = time.Nanosecond *
				(time.Duration(1000000000 * (1.0 / dummyFrequency)))
		}

		sessionInitialization()

		//Get contact list (just for testing)
		contact := ""
		api.UpdateContactList()
		users, nicks := api.GetContactList()

		for i := range users {
			if destinationUserId == users[i] {
				contact = nicks[i]
			}
		}

		fmt.Printf("Sending Message to %d, %v: %s\n", destinationUserId,
			contact, message)

		//Send the message
		bindings.Send(api.APIMessage{SenderID: userId, Payload: message, RecipientID: destinationUserId})

		if dummyFrequency != 0 {
			timer = time.NewTimer(dummyPeriod)
		}

		// Loop until we get a message, then print and exit
		for {

			var msg bindings.Message
			msg, err := bindings.TryReceive()

			end := false

			//Report failed message reception
			if err != nil && err != globals.FifoEmptyErr {
				fmt.Printf("Could not Receive Message: %s\n", err.Error())
				break
			}
			sender := binary.BigEndian.Uint64(msg.GetSender())

			contact = ""
			user, ok := globals.Users.GetUser(sender)
			if ok {
				contact = user.Nick
			}

			//Return the received message to console
			if msg.GetPayload() != "" {
				channelMessage, err := channelbot.ParseChannelbotMessage(msg.
					GetPayload())
				if err == nil {
					speakerContact := ""
					user, ok := globals.Users.GetUser(channelMessage.SpeakerID)
					if ok {
						speakerContact = user.Nick
					}
					fmt.Printf("Message from channel %v, %v:\n%v, %v: %v\n",
						sender, contact, channelMessage.SpeakerID,
						speakerContact, channelMessage.Message)
				} else {
					fmt.Printf("Message from %v, %v Received: %s\n", sender,
						contact, msg.GetPayload())
				}
				end = true
			}

			//If dummy messages are enabled, send the next one
			if dummyPeriod != 0 {
				end = false
				<-timer.C

				contact = ""
				user, ok := globals.Users.GetUser(destinationUserId)
				if ok {
					contact = user.Nick
				}
				fmt.Printf("Sending Message to %d, %v: %s\n", destinationUserId,
					contact, message)

				message := api.APIMessage{SenderID: userId, Payload: message, RecipientID: destinationUserId}
				bindings.Send(message)

				timer = time.NewTimer(dummyPeriod)
				//otherwise just wait to check for new messages
			} else {
				time.Sleep(200 * time.Millisecond)
			}

			if end {
				break
			}

		}

		//Logout
		err := bindings.Logout()

		if err != nil {
			fmt.Printf("Could not logout: %s\n", err.Error())
			return
		}

	},
}

var channelbotCmd = &cobra.Command{
	Use:   "channelbot",
	Short: "Run a channel for communications in a group",
	Run: func(cmd *cobra.Command, args []string) {
		// Logs in, starts reception runner, and so on
		sessionInitialization()

		globals.SetReceiver(func(message format.MessageInterface) {
			payload := message.GetPayload()
			if payload != "" && strings.Index(payload, "/") == 0 {
				// this is a command and we should parse it as a command
				sender := cyclic.NewIntFromBytes(message.GetSender()).Uint64()
				err := channelbot.ParseCommand(payload, sender)
				if err != nil {
					// report the error back to the user who's run the command
					bindings.Send(api.APIMessage{err.Error(),
						globals.Session.GetCurrentUser().UserID, sender})
				}
			} else {
				// this is a normal message that should be rebroadcast
				channelbot.BroadcastMessage(message, &api.APISender{},
					globals.Session.GetCurrentUser().UserID)
			}
		})

		// Block forever as a keepalive
		quit := make(chan bool)
		<-quit
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
	rootCmd.PersistentFlags().BoolVarP(&noRatchet, "noratchet", "", false,
		"Avoid ratcheting the keys for forward secrecy")

	rootCmd.Flags().BoolVar(&blockingTransmission, "blockingTransmission",
		false, "Sets if transmitting messages blocks or not.  "+
			"Defaults to true if unset.")

	rootCmd.Flags().Uint32Var(&rateLimiting, "rateLimiting",
		globals.DefaultTransmitDelay, "Sets the amount of time, in ms, "+
			"that the client waits between sending messages.  "+
			"set to zero to disable.  "+
			"Automatically disabled if 'blockingTransmission' is false")

	rootCmd.PersistentFlags().Uint64VarP(&userId, "userid", "i", 0,
		"UserID to sign in as")
	rootCmd.MarkPersistentFlagRequired("userid")
	rootCmd.PersistentFlags().StringVarP(&nick, "nick", "", "",
		"Nickname to register as")
	rootCmd.PersistentFlags().StringVarP(&serverAddr, "serveraddr", "s", "",
		"Server address to send messages to")
	rootCmd.MarkPersistentFlagRequired("serveraddr")
	// TODO: support this negotiating separate keys with different servers
	rootCmd.PersistentFlags().UintVarP(&numNodes, "numnodes", "n", 1,
		"The number of servers in the network that the client is"+
			" connecting to")
	rootCmd.MarkPersistentFlagRequired("numnodes")

	rootCmd.PersistentFlags().StringVarP(&sessionFile, "sessionfile", "f",
		"", "Passes a file path for loading a session.  "+
			"If the file doesn't exist the code will register the user and"+
			" store it there.  If not passed the session will be stored"+
			" to ram and lost when the cli finishes")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().StringVarP(&message, "message", "m", "", "Message to send")
	rootCmd.PersistentFlags().Uint64VarP(&destinationUserId, "destid", "d", 0,
		"UserID to send message to")

	rootCmd.Flags().Float64Var(&dummyFrequency, "dummyfrequency", 0,
		"Frequency of dummy messages in Hz.  If no message is passed, "+
			"will transmit a random message.  Dummies are only sent if this flag is passed")

	rootCmd.AddCommand(channelbotCmd)
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
