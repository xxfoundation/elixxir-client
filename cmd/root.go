////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package cmd initializes the CLI and config parsers as well as the logger.
package cmd

import (
	"encoding/base64"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/bots"
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/crypto/certs"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/switchboard"
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
var registrationCode string
var userEmail string
var client *api.Client

// Execute adds all child commands to the root command and sets flags
// appropriately.  This is called by main.main(). It only needs to
// happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		jww.ERROR.Println(err)
		os.Exit(1)
	}
}

func sessionInitialization() *id.User {
	var err error
	register := false

	//If no session file is passed initialize with RAM Storage
	if sessionFile == "" {
		client, err = api.NewClient(&globals.RamStorage{}, "")
		if err != nil {
			fmt.Printf("Could Not Initialize Ram Storage: %s\n",
				err.Error())
			return id.ZeroID
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
		client, err = api.NewClient(nil, sessionFile)

		if err != nil {
			fmt.Printf("Could Not Initialize OS Storage: %s\n", err.Error())
			return id.ZeroID
		}
	}

	if noBlockingTransmission {
		client.DisableBlockingTransmission()
	}

	client.SetRateLimiting(rateLimiting)

	// Handle parsing gateway addresses from the config file
	gateways := viper.GetStringSlice("gateways")
	if len(gwAddresses) < 1 {
		// If gwAddr was not passed via command line, check config file
		if len(gateways) < 1 {
			// No gateways in config file or passed via command line
			fmt.Printf("Error: No gateway specified! Add to" +
				" configuration file or pass via command line using -g!")
			return id.ZeroID
		} else {
			// List of gateways found in config file
			gwAddresses = gateways
		}
	}

	// Holds the User ID
	var uid *id.User

	// Register a new user if requested
	if register {
		grpJSON := viper.GetString("group")

		// Unmarshal group JSON
		var grp cyclic.Group
		err := grp.UnmarshalJSON([]byte(grpJSON))
		if err != nil {
			return id.ZeroID
		}

		regCode := registrationCode
		// If precanned user, use generated code instead
		if userId != 0 {
			regCode = new(id.User).SetUints(&[4]uint64{0, 0, 0, userId}).RegistrationCode()
		}

		globals.Log.INFO.Printf("Attempting to register with code %s...", regCode)

		uid, err = client.Register(userId != 0, regCode, "",
			registrationAddr, gwAddresses, mint, &grp)
		if err != nil {
			fmt.Printf("Could Not Register User: %s\n", err.Error())
			return id.ZeroID
		}

		globals.Log.INFO.Printf("Successfully registered user %v!", uid)

	} else {
		// hack for session persisting with cmd line
		// doesn't support non pre canned users
		uid = new(id.User).SetUints(&[4]uint64{0, 0, 0, userId})
		// clear userEmail if it was defined, since login was previously done
		userEmail = ""
	}

	// Log the user in, for now using the first gateway specified
	// This will also register the user email with UDB
	_, err = client.Login(uid, userEmail,
		gwAddresses[0], certs.GatewayTLS)
	if err != nil {
		fmt.Printf("Could Not Log In: %s\n", err)
		return id.ZeroID
	}

	return uid
}

type FallbackListener struct {
	messagesReceived int64
}

func (l *FallbackListener) Hear(item switchboard.Item, isHeardElsewhere bool) {
	if !isHeardElsewhere {
		message := item.(*parse.Message)
		sender, ok := user.Users.GetUser(message.Sender)
		var senderNick string
		if !ok {
			globals.Log.ERROR.Printf("Couldn't get sender %v", message.Sender)
		} else {
			senderNick = sender.Nick
		}
		atomic.AddInt64(&l.messagesReceived, 1)
		fmt.Printf("Message of type %v from %q, %v received with fallback: %s\n",
			message.MessageType, *message.Sender, senderNick,
			string(message.Body))
	}
}

type TextListener struct {
	messagesReceived int64
}

func (l *TextListener) Hear(item switchboard.Item, isHeardElsewhere bool) {
	message := item.(*parse.Message)
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
		globals.Log.INFO.Printf("First message from sender %v", message.Sender)
		u := user.Users.NewUser(message.Sender, base64.StdEncoding.EncodeToString(message.Sender[:]))
		user.Users.UpsertUser(u)
		senderNick = u.Nick
	} else {
		senderNick = sender.Nick
	}
	fmt.Printf("Message from %v, %v Received: %s\n", large.NewIntFromBytes(message.Sender[:]).Text(10),
		senderNick, result.Message)

	atomic.AddInt64(&l.messagesReceived, 1)
}

type ChannelListener struct {
	messagesReceived int64
}

func (l *ChannelListener) Hear(item switchboard.Item, isHeardElsewhere bool) {
	message := item.(*parse.Message)
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
	client.GetSwitchboard().Speak(&parse.Message{
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
		}

		var dummyPeriod time.Duration
		var timer *time.Timer

		// Set the cert paths explicitly to avoid data races
		SetCertPaths(gwCertPath, registrationCertPath)

		userID := sessionInitialization()
		// Set up the listeners for both of the types the client needs for
		// the integration test
		// Normal text messages
		text := TextListener{}
		client.Listen(id.ZeroID, format.None, int32(cmixproto.Type_TEXT_MESSAGE),
			&text)
		// Channel messages
		channel := ChannelListener{}
		client.Listen(id.ZeroID, format.None,
			int32(cmixproto.Type_CHANNEL_MESSAGE), &channel)
		// All other messages
		fallback := FallbackListener{}
		client.Listen(id.ZeroID, format.None, int32(cmixproto.Type_NO_TYPE),
			&fallback)

		// Do calculation for dummy messages if the flag is set
		if dummyFrequency != 0 {
			dummyPeriod = time.Nanosecond *
				(time.Duration(float64(1000000000) * (float64(1.0) / dummyFrequency)))
		}

		// Only send a message if we have a message to send (except dummy messages)
		recipientId := new(id.User).SetUints(&[4]uint64{0, 0, 0, destinationUserId})
		if message != "" {
			// Get the recipient's nick
			recipientNick := ""
			u, ok := user.Users.GetUser(recipientId)
			if ok {
				recipientNick = u.Nick
			}

			// Handle sending to UDB
			if *recipientId == *bots.UdbID {
				parseUdbMessage(message, client)
			} else {
				// Handle sending to any other destination
				wireOut := api.FormatTextMessage(message)

				fmt.Printf("Sending Message to %d, %v: %s\n", destinationUserId,
					recipientNick, message)

				// Send the message
				client.Send(&parse.Message{
					Sender: userID,
					TypedBody: parse.TypedBody{
						MessageType: int32(cmixproto.Type_TEXT_MESSAGE),
						Body:      wireOut,
					},
					CryptoType: format.Unencrypted,
					Receiver: recipientId,
				})
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

				message := &parse.Message{
					Sender: userID,
					TypedBody: parse.TypedBody{
						MessageType: int32(cmixproto.Type_TEXT_MESSAGE),
						Body:      api.FormatTextMessage(message),
					},
					CryptoType: format.Unencrypted,
					Receiver:  recipientId}
				client.Send(message)

				timer = time.NewTimer(dummyPeriod)
			}
		} else {
			// wait 45 seconds since UDB commands are now non-blocking
			// TODO figure out the right way to do this
			timer = time.NewTimer(45 * time.Second)
			<-timer.C
		}

		//Logout
		err := client.Logout()

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

	rootCmd.PersistentFlags().StringVarP(&registrationCode,
		"regcode", "e",
		"",
		"Registration Code")

	rootCmd.PersistentFlags().StringVarP(&userEmail,
			"email", "E",
			"",
			"Email to register for User Discovery")

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
func initConfig() {
	// Temporarily need to get group as JSON data into viper
	json, err := globals.InitCrypto().MarshalJSON()
	if err != nil {
		// panic
	}
	viper.Set("group", string(json))
}

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
