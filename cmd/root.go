////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
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
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/bindings"
	"gitlab.com/elixxir/client/bots"
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/switchboard"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/primitives/id"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"sync/atomic"
	"time"
)

var verbose bool
var userId uint64
var destinationUserId uint64
var gwAddresses []string
var message string
var sessionFile string
var dummyFrequency float64
var noBlockingTransmission bool
var mint bool
var rateLimiting uint32
var showVer bool
var gwCertPath string
var registrationCertPath string
var registrationAddr string

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
	if len(gwAddresses) < 1 {
		// If gwAddr was not passed via command line, check config file
		if len(gateways) < 1 {
			// No gateways in config file or passed via command line
			fmt.Printf("Error: No gateway specified! Add to" +
				" configuration file or pass via command line using -g!")
			return
		} else {
			// List of gateways found in config file
			gwAddresses = gateways
		}
	}

	// Register a new user if requested
	if register {
		// FIXME Use a different encoding for the user ID command line argument,
		// to allow testing with IDs that are long enough to exercise more than
		// 64 bits
		regCode := new(id.User).SetUints(&[4]uint64{0, 0, 0, userId}).RegistrationCode()
		_, err := bindings.Register(regCode, registrationAddr, gwAddresses, mint)
		if err != nil {
			fmt.Printf("Could Not Register User: %s\n", err.Error())
			return
		}
	}

	// Log the user in, for now using the first gateway specified
	uid := id.NewUserFromUint(userId, nil)
	_, err = bindings.Login(uid[:], gwAddresses[0], "")

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
			globals.Log.ERROR.Printf("Couldn't get sender %v", message.Sender)
		} else {
			senderNick = sender.Nick
		}
		atomic.AddInt64(&l.messagesReceived, 1)
		fmt.Printf("Message of type %v from %q, %v received with fallback: %s\n",
			message.Type, *message.Sender, senderNick, string(message.Body))
	}
}

type TextListener struct {
	messagesReceived int64
}

func (l *TextListener) Hear(message *parse.Message, isHeardElsewhere bool) {
	globals.Log.INFO.Println("Hearing a text message")
	result := cmixproto.TextMessage{}
	err := proto.Unmarshal(message.Body, &result)
	if err != nil {
		globals.Log.ERROR.Printf("Error unmarshaling text message: %v\n",
			err.Error())
	}

	sender, ok := user.Users.GetUser(message.Sender)
	var senderNick string
	if !ok {
		globals.Log.ERROR.Printf("Couldn't get sender %v", message.Sender)
	} else {
		senderNick = sender.Nick
	}
	fmt.Printf("Message from %v, %v Received: %s\n", new(big.Int).SetBytes(message.Sender[:]).Text(10),
		senderNick, result.Message)

	atomic.AddInt64(&l.messagesReceived, 1)
}

type ChannelListener struct {
	messagesReceived int64
}

func (l *ChannelListener) Hear(message *parse.Message, isHeardElsewhere bool) {
	globals.Log.INFO.Println("Hearing a channel message")
	result := cmixproto.ChannelMessage{}
	proto.Unmarshal(message.Body, &result)

	sender, ok := user.Users.GetUser(message.Sender)
	var senderNick string
	if !ok {
		globals.Log.ERROR.Printf("Couldn't get sender %v", message.Sender)
	} else {
		senderNick = sender.Nick
	}

	fmt.Printf("Message from channel %v, %v: ",
		new(big.Int).SetBytes(message.Sender[:]).Text(10), senderNick)
	typedBody, _ := parse.Parse(result.Message)
	speakerId := new(id.User).SetBytes(result.SpeakerID)
	switchboard.Listeners.Speak(&parse.Message{
		TypedBody: *typedBody,
		Sender:    speakerId,
		Receiver:  id.ZeroID,
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
		}

		var dummyPeriod time.Duration
		var timer *time.Timer

		// Set the cert paths explicitly to avoid data races
		SetCertPaths(gwCertPath, registrationCertPath)

		// Set up the listeners for both of the types the client needs for
		// the integration test
		// Normal text messages
		text := TextListener{}
		api.Listen(id.ZeroID, cmixproto.Type_TEXT_MESSAGE, &text, switchboard.Listeners)
		// Channel messages
		channel := ChannelListener{}
		api.Listen(id.ZeroID, cmixproto.Type_CHANNEL_MESSAGE, &channel, switchboard.Listeners)
		// All other messages
		fallback := FallbackListener{}
		api.Listen(id.ZeroID, cmixproto.Type_NO_TYPE, &fallback, switchboard.Listeners)

		// Do calculation for dummy messages if the flag is set
		if dummyFrequency != 0 {
			dummyPeriod = time.Nanosecond *
				(time.Duration(float64(1000000000) * (float64(1.0) / dummyFrequency)))
		}

		sessionInitialization()

		// Only send a message if we have a message to send (except dummy messages)
		recipientId := new(id.User).SetUints(&[4]uint64{0, 0, 0, destinationUserId})
		senderId := new(id.User).SetUints(&[4]uint64{0, 0, 0, userId})
		if message != "" {
			// Get the recipient's nick
			recipientNick := ""
			u, ok := user.Users.GetUser(recipientId)
			if ok {
				recipientNick = u.Nick
			}

			// Handle sending to UDB
			if *recipientId == *bots.UdbID {
				fmt.Println(parseUdbMessage(message))
			} else {
				// Handle sending to any other destination
				wireOut := bindings.FormatTextMessage(message)

				fmt.Printf("Sending Message to %d, %v: %s\n", destinationUserId,
					recipientNick, message)

				// Send the message
				bindings.Send(&parse.BindingsMessageProxy{&parse.Message{
					Sender: senderId,
					TypedBody: parse.TypedBody{
						Type: cmixproto.Type_TEXT_MESSAGE,
						Body: wireOut,
					},
					Receiver: recipientId,
				}})
			}
		}

		if dummyFrequency != 0 {
			timer = time.NewTimer(dummyPeriod)
		}

		if dummyPeriod != 0 {
			for {
				// need to constantly send new messages
				<-timer.C

				contact := ""
				u, ok := user.Users.GetUser(recipientId)
				if ok {
					contact = u.Nick
				}
				fmt.Printf("Sending Message to %d, %v: %s\n", destinationUserId,
					contact, message)

				message := &parse.BindingsMessageProxy{&parse.Message{
					Sender: senderId,
					TypedBody: parse.TypedBody{
						Type: cmixproto.Type_TEXT_MESSAGE,
						Body: bindings.FormatTextMessage(message),
					},
					Receiver: recipientId}}
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
	rootCmd.PersistentFlags().BoolVarP(&mint, "mint", "", false,
		"Mint some coins for testing")

	rootCmd.PersistentFlags().Uint32VarP(&rateLimiting, "rateLimiting", "",
		1000, "Sets the amount of time, in ms, "+
			"that the client waits between sending messages.  "+
			"set to zero to disable.  "+
			"Automatically disabled if 'blockingTransmission' is false")

	rootCmd.PersistentFlags().Uint64VarP(&userId, "userid", "i", 0,
		"ID to sign in as")
	rootCmd.PersistentFlags().StringSliceVarP(&gwAddresses, "gwaddresses",
		"g", make([]string, 0), "Gateway addresses:port for message sending, "+
			"comma-separated")
	rootCmd.PersistentFlags().StringVarP(&gwCertPath, "gwcertpath", "c", "",
		"Path to the certificate file for connecting to gateway using TLS")
	rootCmd.PersistentFlags().StringVarP(&registrationCertPath, "registrationcertpath", "r",
		"",
		"Path to the certificate file for connecting to registration server"+
			" using TLS")
	rootCmd.PersistentFlags().StringVarP(&registrationAddr,
		"registrationaddr", "a",
		"",
		"Address:Port for connecting to registration server"+
			" using TLS")

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

// Sets the cert paths in comms
func SetCertPaths(gwCertPath, registrationCertPath string) {
	connect.GatewayCertPath = gwCertPath
	connect.RegistrationCertPath = registrationCertPath
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {}

// initLog initializes logging thresholds and the log path.
func initLog() {
	globals.Log = jww.NewNotepad(jww.LevelError, jww.LevelWarn, os.Stdout,
		ioutil.Discard, "CLIENT", log.Ldate|log.Ltime)
	// If verbose flag set then log more info for debugging
	if verbose || viper.GetBool("verbose") {
		globals.Log.SetLogThreshold(jww.LevelDebug)
		globals.Log.SetStdoutThreshold(jww.LevelDebug)
		globals.Log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	} else {
		globals.Log.SetLogThreshold(jww.LevelWarn)
		globals.Log.SetStdoutThreshold(jww.LevelWarn)
	}
	if viper.Get("logPath") != nil {
		// Create log file, overwrites if existing
		logPath := viper.GetString("logPath")
		logFile, err := os.Create(logPath)
		if err != nil {
			globals.Log.WARN.Println("Invalid or missing log path, default path used.")
		} else {
			globals.Log.SetLogOutput(logFile)
		}
	}
}
