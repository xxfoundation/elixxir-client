////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package cmd initializes the CLI and config parsers as well as the logger.
package cmd

import (
	"encoding/base64"
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
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/primitives/switchboard"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"strconv"
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
var rateLimiting uint32
var showVer bool
var gwCertPath string
var registrationCertPath string
var registrationAddr string
var registrationCode string
var userEmail string
var userNick string
var end2end bool
var keyParams []string
var ndfPath string
var skipNDFVerification bool
var ndfRegistration []string
var ndfUDB []string
var ndfPubKey string

// Execute adds all child commands to the root command and sets flags
// appropriately.  This is called by main.main(). It only needs to
// happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		jww.ERROR.Println(err)
		os.Exit(1)
	}
}

func sessionInitialization() (*id.User, *api.Client) {
	var err error
	register := false

	var client *api.Client

	// Read in the network definition file and save as string
	ndfBytes, err := ioutil.ReadFile(ndfPath)
	if err != nil {
		globals.Log.FATAL.Panicf("Could not read network definition file: %v", err)
	}

	// Check if the NDF verify flag is set
	if skipNDFVerification {
		ndfPubKey = ""
		globals.Log.WARN.Println("Skipping NDF verification")
	} else if ndfPubKey == "" {
		globals.Log.FATAL.Panicln("No public key for NDF found")
	}

	// Verify the signature
	ndfJSON := api.VerifyNDF(string(ndfBytes), ndfPubKey)

	// Overwrite the network definition with any specified flags
	overwriteNDF(ndfJSON)

	//If no session file is passed initialize with RAM Storage
	if sessionFile == "" {
		client, err = api.NewClient(&globals.RamStorage{}, "", ndfJSON)
		if err != nil {
			globals.Log.ERROR.Printf("Could Not Initialize Ram Storage: %s\n",
				err.Error())
			return id.ZeroID, nil
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
				globals.Log.ERROR.Printf("Error with file path: %s\n", err1.Error())
				return id.ZeroID, nil
			}
		}

		//Initialize client with OS Storage
		client, err = api.NewClient(nil, sessionFile, ndfJSON)

		if err != nil {
			globals.Log.ERROR.Printf("Could Not Initialize OS Storage: %s\n", err.Error())
			return id.ZeroID, nil
		}
	}

	if noBlockingTransmission {
		client.DisableBlockingTransmission()
	}

	client.SetRateLimiting(rateLimiting)

	// Handle parsing gateway addresses from the config file
	gateways := ndfJSON.Gateways
	// If gwAddr was not passed via command line, check config file
	if len(gateways) < 1 {
		// No gateways in config file or passed via command line
		globals.Log.ERROR.Printf("Error: No gateway specified! Add to" +
			" configuration file or pass via command line using -g!\n")
		return id.ZeroID, nil
	}

	// Connect to gateways and reg server
	err = client.Connect()

	if err != nil {
		globals.Log.FATAL.Panicf("Could not call connect on client: %+v", err)
	}

	// Holds the User ID
	var uid *id.User

	// Register a new user if requested
	if register {
		regCode := registrationCode
		// If precanned user, use generated code instead
		if userId != 0 {
			regCode = id.NewUserFromUints(&[4]uint64{0, 0, 0, userId}).RegistrationCode()
		}

		globals.Log.INFO.Printf("Attempting to register with code %s...", regCode)

		uid, err = client.Register(userId != 0, regCode, userNick, userEmail)
		if err != nil {
			globals.Log.ERROR.Printf("Could Not Register User: %s\n", err.Error())
			return id.ZeroID, nil
		}

		globals.Log.INFO.Printf("Successfully registered user %v!", uid)

	} else {
		// hack for session persisting with cmd line
		// doesn't support non pre canned users
		uid = id.NewUserFromUints(&[4]uint64{0, 0, 0, userId})
		// clear userEmail if it was defined, since login was previously done
		userEmail = ""
	}

	// Log the user in, for now using the first gateway specified
	// This will also register the user email with UDB
	_, err = client.Login(uid)
	if err != nil {
		globals.Log.ERROR.Printf("Could Not Log In: %s\n", err)
		return id.ZeroID, nil
	}

	return uid, client
}

func setKeyParams(client *api.Client) {
	globals.Log.DEBUG.Printf("Trying to parse key parameters...")
	minKeys, err := strconv.Atoi(keyParams[0])
	if err != nil {
		return
	}

	maxKeys, err := strconv.Atoi(keyParams[1])
	if err != nil {
		return
	}

	numRekeys, err := strconv.Atoi(keyParams[2])
	if err != nil {
		return
	}

	ttlScalar, err := strconv.ParseFloat(keyParams[3], 64)
	if err != nil {
		return
	}

	minNumKeys, err := strconv.Atoi(keyParams[4])
	if err != nil {
		return
	}

	globals.Log.DEBUG.Printf("Setting key generation parameters: %d, %d, %d, %f, %d",
		minKeys, maxKeys, numRekeys, ttlScalar, minNumKeys)

	params := client.GetKeyParams()
	params.MinKeys = uint16(minKeys)
	params.MaxKeys = uint16(maxKeys)
	params.NumRekeys = uint16(numRekeys)
	params.TTLScalar = ttlScalar
	params.MinNumKeys = uint16(minNumKeys)
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
		globals.Log.INFO.Printf("Message of type %v from %q, %v received with fallback: %s\n",
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
	globals.Log.INFO.Printf("Message from %v, %v Received: %s\n", large.NewIntFromBytes(message.Sender[:]).Text(10),
		senderNick, result.Message)

	atomic.AddInt64(&l.messagesReceived, 1)
}

type ChannelListener struct {
	messagesReceived int64
}

//used to get the client object into hear
var globalClient *api.Client

func (l *ChannelListener) Hear(item switchboard.Item, isHeardElsewhere bool) {
	message := item.(*parse.Message)
	globals.Log.INFO.Println("Hearing a channel message")
	result := cmixproto.ChannelMessage{}
	err := proto.Unmarshal(message.Body, &result)

	if err != nil {
		globals.Log.ERROR.Printf("Could not unmarhsal message, message "+
			"not processed: %+v", err)
	}

	sender, ok := user.Users.GetUser(message.Sender)
	var senderNick string
	if !ok {
		globals.Log.ERROR.Printf("Couldn't get sender %v", message.Sender)
	} else {
		senderNick = sender.Nick
	}

	globals.Log.INFO.Printf("Message from channel %v, %v: ",
		new(big.Int).SetBytes(message.Sender[:]).Text(10), senderNick)
	typedBody, _ := parse.Parse(result.Message)
	speakerId := id.NewUserFromBytes(result.SpeakerID)
	globalClient.GetSwitchboard().Speak(&parse.Message{
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

		userID, client := sessionInitialization()
		globalClient = client
		// Set Key parameters if defined
		if len(keyParams) == 5 {
			setKeyParams(client)
		}
		// Set up the listeners for both of the types the client needs for
		// the integration test
		// Normal text messages
		text := TextListener{}
		client.Listen(id.ZeroID, int32(cmixproto.Type_TEXT_MESSAGE),
			&text)
		// Channel messages
		channel := ChannelListener{}
		client.Listen(id.ZeroID,
			int32(cmixproto.Type_CHANNEL_MESSAGE), &channel)
		// All other messages
		fallback := FallbackListener{}
		client.Listen(id.ZeroID, int32(cmixproto.Type_NO_TYPE),
			&fallback)

		// Do calculation for dummy messages if the flag is set
		if dummyFrequency != 0 {
			dummyPeriod = time.Nanosecond *
				(time.Duration(float64(1000000000) * (float64(1.0) / dummyFrequency)))
		}

		cryptoType := parse.Unencrypted
		if end2end {
			cryptoType = parse.E2E
		}

		// Only send a message if we have a message to send (except dummy messages)
		recipientId := id.NewUserFromUints(&[4]uint64{0, 0, 0, destinationUserId})
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

				globals.Log.INFO.Printf("Sending Message to %d, %v: %s\n", destinationUserId,
					recipientNick, message)

				// Send the message
				err := client.Send(&parse.Message{
					Sender: userID,
					TypedBody: parse.TypedBody{
						MessageType: int32(cmixproto.Type_TEXT_MESSAGE),
						Body:        wireOut,
					},
					InferredType: cryptoType,
					Receiver:     recipientId,
				})
				if err != nil {
					globals.Log.ERROR.Printf("Error sending message: %+v", err)
				}
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
				globals.Log.INFO.Printf("Sending Message to %d, %v: %s\n", destinationUserId,
					contact, message)

				message := &parse.Message{
					Sender: userID,
					TypedBody: parse.TypedBody{
						MessageType: int32(cmixproto.Type_TEXT_MESSAGE),
						Body:        api.FormatTextMessage(message),
					},
					InferredType: cryptoType,
					Receiver:     recipientId}
				err := client.Send(message)
				if err != nil {
					globals.Log.ERROR.Printf("Error sending message: %+v", err)
				}

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
			globals.Log.ERROR.Printf("Could not logout: %s\n", err.Error())
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
		"default@default.com",
		"Email to register for User Discovery")

	rootCmd.PersistentFlags().StringVar(&userNick,
		"nick",
		"Default",
		"Nickname to register for User Discovery")

	rootCmd.PersistentFlags().StringVarP(&sessionFile, "sessionfile", "f",
		"", "Passes a file path for loading a session.  "+
			"If the file doesn't exist the code will register the user and"+
			" store it there.  If not passed the session will be stored"+
			" to ram and lost when the cli finishes")

	rootCmd.PersistentFlags().StringVarP(&ndfPubKey,
		"ndfPubKey",
		"p",
		"",
		"Path to the public key for the network definition JSON file")

	rootCmd.PersistentFlags().StringVarP(&ndfPath,
		"ndf",
		"n",
		"ndf.json",
		"Path to the network definition JSON file")

	rootCmd.PersistentFlags().BoolVar(&skipNDFVerification,
		"skipNDFVerification",
		false,
		"Specifies if the NDF should be loaded without the signature")

	rootCmd.PersistentFlags().StringSliceVar(&ndfRegistration,
		"ndfRegistration",
		nil,
		"Overwrite the Registration values for the NDF")

	rootCmd.PersistentFlags().StringSliceVar(&ndfUDB,
		"ndfUDB",
		nil,
		"Overwrite the UDB values for the NDF")

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

	rootCmd.PersistentFlags().BoolVarP(&end2end, "end2end", "", false,
		"Send messages with E2E encryption to destination user")

	rootCmd.PersistentFlags().StringSliceVarP(&keyParams, "keyParams", "",
		make([]string, 0), "Define key generation parameters. Pass values in comma separated list"+
			" in the following order: MinKeys,MaxKeys,NumRekeys,TTLScalar,MinNumKeys")
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

// overwriteNDF replaces fields in the NetworkDefinition structure with values
// specified from the commandline.
func overwriteNDF(n *ndf.NetworkDefinition) {
	if len(ndfRegistration) == 3 {
		n.Registration.DsaPublicKey = ndfRegistration[0]
		n.Registration.Address = ndfRegistration[1]
		n.Registration.TlsCertificate = ndfRegistration[2]

		globals.Log.WARN.Println("Overwrote Registration values in the " +
			"NetworkDefinition from the commandline")
	}

	if len(ndfUDB) == 2 {
		udbIdString, err := base64.StdEncoding.DecodeString(ndfUDB[0])
		if err != nil {
			globals.Log.WARN.Printf("Could not decode USB ID: %v", err)
		}

		n.UDB.ID = udbIdString
		n.UDB.DsaPublicKey = ndfUDB[1]

		globals.Log.WARN.Println("Overwrote UDB values in the " +
			"NetworkDefinition from the commandline")
	}
}
