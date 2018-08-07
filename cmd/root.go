////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package cmd initializes the CLI and config parsers as well as the logger.
package cmd

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/privategrity/client/api"
	"gitlab.com/privategrity/client/bindings"
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/client/parse"
	"gitlab.com/privategrity/client/switchboard"
	"gitlab.com/privategrity/client/user"
	"gitlab.com/privategrity/crypto/cyclic"
	"os"
	"sync/atomic"
	"time"
)

var verbose bool
var userId uint64
var destinationUserId uint64
var gwAddr string
var message string
var numNodes uint
var sessionFile string
var dummyFrequency float64
var noBlockingTransmission bool
var mint bool
var rateLimiting uint32
var showVer bool

// CmdMessage are an implementation of the interface in bindings and API
// easy to use from Go
type CmdMessage struct {
	Payload     string
	SenderID    user.ID
	RecipientID user.ID
}

func (m CmdMessage) GetSender() []byte {
	return m.SenderID.Bytes()
}

func (m CmdMessage) GetRecipient() []byte {
	return m.RecipientID.Bytes()
}

func (m CmdMessage) GetPayload() string {
	return m.Payload
}

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
	if noBlockingTransmission {
		api.DisableBlockingTransmission()
	}

	bindings.SetRateLimiting(int(rateLimiting))

	var err error
	register := false

	//If no session file is passed initialize with RAM Storage
	if sessionFile == "" {
		err = bindings.InitClient(&globals.RamStorage{}, "")
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
		err = bindings.InitClient(&globals.DefaultStorage{}, sessionFile)

		if err != nil {
			fmt.Printf("Could Not Initialize OS Storage: %s\n", err.Error())
			return
		}
	}

	// Handle parsing gateway addresses from the config file
	gateways := viper.GetStringSlice("gateways")
	if gwAddr == "" {
		// If gwAddr was not passed via command line, check config file
		if len(gateways) < 1 {
			// No gateways in config file or passed via command line
			fmt.Printf("Error: No gateway specified! Add to" +
				" configuration file or pass via command line using -g!")
			return
		} else {
			// List of gateways found in config file, select one to use
			// TODO: For now, just use the first one?
			gwAddr = gateways[0]
		}
	}

	//Register a new user if requested
	if register {
		_, err := bindings.Register(
			cyclic.NewIntFromBytes(user.UserHash(user.ID(userId))).
				TextVerbose(32, 0), gwAddr, int(numNodes))
		if err != nil {
			fmt.Printf("Could Not Register User: %s\n", err.Error())
			return
		}
	}

	// Mint coins only if the flag is set and it's a new session
	doMint := mint && register
	// Log the user in
	_, err = bindings.Login(
		cyclic.NewIntFromUInt(userId).LeftpadBytes(8), gwAddr, doMint)

	if err != nil {
		fmt.Printf("Could Not Log In\n")
		return
	}
}

type FallbackListener struct {
	messagesReceived int64
}

func (l *FallbackListener) Hear(message *parse.Message, isHeardElsewhere bool) {
	if !isHeardElsewhere {
		sender, ok := user.Users.GetUser(message.Sender)
		var senderNick string
		if !ok {
			jww.ERROR.Printf("Couldn't get sender %v", message.Sender)
		} else {
			senderNick = sender.Nick
		}
		atomic.AddInt64(&l.messagesReceived, 1)
		fmt.Printf("Message of type %v from %v, %v received with fallback: %s\n",
			message.Type, message.Sender, senderNick, string(message.Body))
	}
}

type TextListener struct {
	messagesReceived int64
}

func (l *TextListener) Hear(message *parse.Message, isHeardElsewhere bool) {
	jww.INFO.Println("Hearing a text message")
	result := parse.TextMessage{}
	proto.Unmarshal(message.Body, &result)

	sender, ok := user.Users.GetUser(message.Sender)
	var senderNick string
	if !ok {
		jww.ERROR.Printf("Couldn't get sender %v", message.Sender)
	} else {
		senderNick = sender.Nick
	}
	fmt.Printf("Message from %v, %v Received: %s\n", message.Sender,
		senderNick, result.Message)

	atomic.AddInt64(&l.messagesReceived, 1)
}

type ChannelListener struct {
	messagesReceived int64
}

func (l *ChannelListener) Hear(message *parse.Message, isHeardElsewhere bool) {
	jww.INFO.Println("Hearing a channel message")
	result := parse.ChannelMessage{}
	proto.Unmarshal(message.Body, &result)

	sender, ok := user.Users.GetUser(message.Sender)
	var senderNick string
	if !ok {
		jww.ERROR.Printf("Couldn't get sender %v", message.Sender)
	} else {
		senderNick = sender.Nick
	}

	speakerID := user.NewIDFromBytes(result.SpeakerID)

	fmt.Printf("Message from channel %v, %v: ",
		message.Sender, senderNick)
	typedBody, _ := parse.Parse(result.Message)
	switchboard.Listeners.Speak(&parse.Message{
		TypedBody: *typedBody,
		Sender:    speakerID,
		Receiver:  0,
	})
	atomic.AddInt64(&l.messagesReceived, 1)
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "client",
	Short: "Runs a client for cMix anonymous communication platform",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// Main client run function

		if showVer {
			printVersion()
			return
		} else {
			cmd.MarkPersistentFlagRequired("userid")
			cmd.MarkPersistentFlagRequired("numnodes")
		}

		var dummyPeriod time.Duration
		var timer *time.Timer

		// Set up the listeners for both of the types the client needs for
		// the integration test
		// Normal text messages
		text := TextListener{}
		api.Listen(user.ID(0), parse.Type_TEXT_MESSAGE, &text)
		// Channel messages
		channel := ChannelListener{}
		api.Listen(user.ID(0), parse.Type_CHANNEL_MESSAGE, &channel)
		// All other messages
		fallback := FallbackListener{}
		api.Listen(user.ID(0), parse.Type_NO_TYPE, &fallback)

		// Do calculation for dummy messages if the flag is set
		if dummyFrequency != 0 {
			dummyPeriod = time.Nanosecond *
				(time.Duration(float64(1000000000) * (float64(1.0) / dummyFrequency)))
		}

		sessionInitialization()

		// Only send a message if we have a message to send (except dummy messages)
		if message != "" {
			// Get the recipient's nick
			recipientNick := ""
			u, ok := user.Users.GetUser(user.ID(destinationUserId))
			if ok {
				recipientNick = u.Nick
			}
			wireOut := bindings.FormatTextMessage(message)

			fmt.Printf("Sending Message to %d, %v: %s\n", destinationUserId,
				recipientNick, message)

			//Send the message
			bindings.Send(CmdMessage{
				SenderID:    user.ID(userId),
				Payload:     string(wireOut),
				RecipientID: user.ID(destinationUserId),
			})
		}

		if dummyFrequency != 0 {
			timer = time.NewTimer(dummyPeriod)
		}

		if dummyPeriod != 0 {
			for {
				// need to constantly send new messages
				<-timer.C

				contact := ""
				u, ok := user.Users.GetUser(user.ID(destinationUserId))
				if ok {
					contact = u.Nick
				}
				fmt.Printf("Sending Message to %d, %v: %s\n", destinationUserId,
					contact, message)

				message := CmdMessage{
					SenderID:    user.ID(userId),
					Payload:     string(api.FormatTextMessage(message)),
					RecipientID: user.ID(destinationUserId)}
				bindings.Send(message)

				timer = time.NewTimer(dummyPeriod)
			}
		} else {
			// wait 5 seconds to get all the messages off the gateway,
			// unless you're sending to the channelbot, in which case you need
			// to wait longer because channelbot is slow and dumb
			// TODO figure out the right way to do this
			if destinationUserId == 31 {
				timer = time.NewTimer(20 * time.Second)
			} else {
				timer = time.NewTimer(5 * time.Second)
			}
			<-timer.C
		}

		//Logout
		err := bindings.Logout()

		if err != nil {
			fmt.Printf("Could not logout: %s\n", err.Error())
			return
		}

	},
}

// init is the initialization function for Cobra which defines commands
// and flags.
func init() {
	// NOTE: The point of init() is to be declarative.
	// There is one init in each sub command. Do not put variable declarations
	// here, and ensure all the Flags are of the *P variety, unless there's a
	// very good reason not to have them as local params to sub command."
	cobra.OnInitialize(initConfig, initLog)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false,
		"Verbose mode for debugging")

	rootCmd.PersistentFlags().BoolVarP(&noBlockingTransmission, "noBlockingTransmission",
		"", false, "Sets if transmitting messages blocks or not.  "+
			"Defaults to true if unset.")
	rootCmd.PersistentFlags().BoolVarP(&mint, "mint", "m", false,
		"Mint some coins for testing")

	rootCmd.PersistentFlags().Uint32VarP(&rateLimiting, "rateLimiting", "",
		1000, "Sets the amount of time, in ms, "+
			"that the client waits between sending messages.  "+
			"set to zero to disable.  "+
			"Automatically disabled if 'blockingTransmission' is false")

	rootCmd.PersistentFlags().Uint64VarP(&userId, "userid", "i", 0,
		"ID to sign in as")
	rootCmd.PersistentFlags().StringVarP(&gwAddr, "gwaddr", "g", "",
		"Gateway address to send messages to")
	// TODO: support this negotiating separate keys with different servers
	rootCmd.PersistentFlags().UintVarP(&numNodes, "numnodes", "n", 1,
		"The number of servers in the network that the client is"+
			" connecting to")

	rootCmd.PersistentFlags().StringVarP(&sessionFile, "sessionfile", "f",
		"", "Passes a file path for loading a session.  "+
			"If the file doesn't exist the code will register the user and"+
			" store it there.  If not passed the session will be stored"+
			" to ram and lost when the cli finishes")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().StringVarP(&message, "message", "m", "", "Message to send")
	rootCmd.PersistentFlags().Uint64VarP(&destinationUserId, "destid", "d", 0,
		"ID to send message to")
	rootCmd.Flags().BoolVarP(&showVer, "version", "V", false,
		"Show the server version information.")

	rootCmd.Flags().Float64VarP(&dummyFrequency, "dummyfrequency", "", 0,
		"Frequency of dummy messages in Hz.  If no message is passed, "+
			"will transmit a random message.  Dummies are only sent if this flag is passed")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {}

// initLog initializes logging thresholds and the log path.
func initLog() {
	// If verbose flag set then log more info for debugging
	if verbose || viper.GetBool("verbose") {
		jww.SetLogThreshold(jww.LevelInfo)
		jww.SetStdoutThreshold(jww.LevelInfo)
	} else {
		jww.SetLogThreshold(jww.LevelWarn)
		jww.SetStdoutThreshold(jww.LevelWarn)
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
