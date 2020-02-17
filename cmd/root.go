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
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/switchboard"
	"gitlab.com/elixxir/primitives/utils"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var verbose bool
var userId uint64
var privateKeyPath string
var destinationUserId uint64
var destinationUserIDBase64 string
var message string
var sessionFile string
var noBlockingTransmission bool
var rateLimiting uint32
var showVer bool
var registrationCode string
var username string
var end2end bool
var keyParams []string
var ndfPath string
var skipNDFVerification bool
var ndfPubKey string
var sessFilePassword string
var noTLS bool
var searchForUser string
var waitForMessages uint
var messageTimeout uint
var messageCnt uint
var notificationToken string

// Execute adds all child commands to the root command and sets flags
// appropriately.  This is called by main.main(). It only needs to
// happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func sessionInitialization() (*id.User, string, *api.Client) {
	var err error
	register := false

	var client *api.Client

	// Read in the network definition file and save as string
	ndfBytes, err := utils.ReadFile(ndfPath)
	if err != nil {
		globals.Log.FATAL.Panicf("Could not read network definition file: %v", err)
	}

	// Check if the NDF verify flag is set
	if skipNDFVerification {
		ndfPubKey = ""
		globals.Log.WARN.Println("Skipping NDF verification")
	} else {
		pkFile, err := os.Open(ndfPubKey)
		if err != nil {
			globals.Log.FATAL.Panicf("Could not open cert file: %v",
				err)
		}

		pkBytes, err := ioutil.ReadAll(pkFile)
		if err != nil {
			globals.Log.FATAL.Panicf("Could not read cert file: %v",
				err)
		}
		ndfPubKey = string(pkBytes)
	}

	// Verify the signature
	globals.Log.DEBUG.Println("Verifying NDF...")
	ndfJSON := api.VerifyNDF(string(ndfBytes), ndfPubKey)
	globals.Log.DEBUG.Printf("   NDF Verified")

	//If no session file is passed initialize with RAM Storage
	if sessionFile == "" {
		client, err = api.NewClient(&globals.RamStorage{}, "", "", ndfJSON)
		if err != nil {
			globals.Log.ERROR.Printf("Could Not Initialize Ram Storage: %s\n",
				err.Error())
			return id.ZeroID, "", nil
		}
		globals.Log.INFO.Println("Initialized Ram Storage")
		register = true
	} else {

		var sessionA, sessionB string

		locs := strings.Split(sessionFile, ",")

		if len(locs) == 2 {
			sessionA = locs[0]
			sessionB = locs[1]
		} else {
			sessionA = sessionFile
			sessionB = sessionFile + "-2"
		}

		//If a session file is passed, check if it's valid
		_, err1 := os.Stat(sessionA)
		_, err2 := os.Stat(sessionB)

		if err1 != nil && err2 != nil {
			//If the file does not exist, register a new user
			if os.IsNotExist(err1) && os.IsNotExist(err2) {
				register = true
			} else {
				//Fail if any other error is received
				globals.Log.ERROR.Printf("Error with file paths: %s %s",
					err1, err2)
				return id.ZeroID, "", nil
			}
		}
		//Initialize client with OS Storage
		client, err = api.NewClient(nil, sessionA, sessionB, ndfJSON)
		if err != nil {
			globals.Log.ERROR.Printf("Could Not Initialize OS Storage: %s\n", err.Error())
			return id.ZeroID, "", nil
		}
		globals.Log.INFO.Println("Initialized OS Storage")

	}

	if noBlockingTransmission {
		globals.Log.INFO.Println("Disabling Blocking Transmissions")
		client.DisableBlockingTransmission()
	}

	client.SetRateLimiting(rateLimiting)

	// Handle parsing gateway addresses from the config file

	//REVIEWER NOTE: Possibly need to remove/rearrange this,
	// now that client may not know gw's upon client creation
	/*gateways := client.GetNDF().Gateways
	// If gwAddr was not passed via command line, check config file
	if len(gateways) < 1 {
		// No gateways in config file or passed via command line
		globals.Log.ERROR.Printf("Error: No gateway specified! Add to" +
			" configuration file or pass via command line using -g!\n")
		return id.ZeroID, "", nil
	}*/

	if noTLS {
		client.DisableTls()
	}

	// InitNetwork to gateways, notificationBot and reg server
	err = client.InitNetwork()
	if err != nil {
		globals.Log.FATAL.Panicf("Could not call connect on client: %+v", err)
	}

	// Holds the User ID
	var uid *id.User

	// Register a new user if requested
	if register {
		globals.Log.INFO.Println("Registering...")

		regCode := registrationCode
		// If precanned user, use generated code instead
		if userId != 0 {
			regCode = id.NewUserFromUints(&[4]uint64{0, 0, 0, userId}).RegistrationCode()
		}

		globals.Log.INFO.Printf("Building keys...")

		var privKey *rsa.PrivateKey

		if privateKeyPath != "" {
			privateKeyBytes, err := utils.ReadFile(privateKeyPath)
			if err != nil {
				globals.Log.FATAL.Panicf("Could not load user private key PEM from "+
					"path %s: %+v", privateKeyPath, err)
			}

			privKey, err = rsa.LoadPrivateKeyFromPem(privateKeyBytes)
			if err != nil {
				globals.Log.FATAL.Panicf("Could not load private key from PEM bytes: %+v", err)
			}
		}

		//Generate keys for registration
		err := client.GenerateKeys(privKey, sessFilePassword)
		if err != nil {
			globals.Log.FATAL.Panicf("%+v", err)
		}

		globals.Log.INFO.Printf("Attempting to register with code %s...", regCode)

		errRegister := fmt.Errorf("")
		uid = client.GetCurrentUser()
		//Attempt to register user with same keys until a success occurs
		for errRegister != nil {
			_, errRegister = client.RegisterWithPermissioning(userId != 0, regCode)
			if errRegister != nil {
				globals.Log.FATAL.Panicf("Could Not Register User: %s",
					errRegister.Error())
			}
		}

		err = client.RegisterWithNodes()
		if err != nil {
			globals.Log.FATAL.Panicf("Could Not Register User with nodes: %s",
				err.Error())
		}

		userbase64 := base64.StdEncoding.EncodeToString(uid[:])
		globals.Log.INFO.Printf("Registered as user (uid, the var) %v", uid)
		globals.Log.INFO.Printf("Registered as user (userID, the global) %v", userId)
		globals.Log.INFO.Printf("Successfully registered user %s!", userbase64)

	} else {
		// hack for session persisting with cmd line
		// doesn't support non pre canned users
		uid = id.NewUserFromUints(&[4]uint64{0, 0, 0, userId})
		globals.Log.INFO.Printf("Skipped Registration, user: %v", uid)
	}

	_, err = client.Login(sessFilePassword)

	if err != nil {
		globals.Log.FATAL.Panicf("Could not login: %v", err)
	}
	return uid, client.GetSession().GetCurrentUser().Username, client
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
	MessagesReceived int64
}

func (l *FallbackListener) Hear(item switchboard.Item, isHeardElsewhere bool, i ...interface{}) {
	if !isHeardElsewhere {
		message := item.(*parse.Message)
		sender, ok := user.Users.GetUser(message.Sender)
		var senderNick string
		if !ok {
			globals.Log.ERROR.Printf("Couldn't get sender %v", message.Sender)
		} else {
			senderNick = sender.Username
		}
		atomic.AddInt64(&l.MessagesReceived, 1)
		globals.Log.INFO.Printf("Message of type %v from %q, %v received with fallback: %s\n",
			message.MessageType, *message.Sender, senderNick,
			string(message.Body))
	}
}

type TextListener struct {
	MessagesReceived int64
}

func (l *TextListener) Hear(item switchboard.Item, isHeardElsewhere bool, i ...interface{}) {
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
		senderNick = u.Username
	} else {
		senderNick = sender.Username
	}
	fmt.Printf("Message from %v, %v Received: %s\n Timestamp: %s",
		large.NewIntFromBytes(message.Sender[:]).Text(10),
		senderNick, result.Message, message.Timestamp.String())

	atomic.AddInt64(&l.MessagesReceived, 1)
}

type userSearcher struct {
	foundUserChan chan []byte
}

func newUserSearcher() api.SearchCallback {
	us := userSearcher{}
	us.foundUserChan = make(chan []byte)
	return &us
}

func (us *userSearcher) Callback(userID, pubKey []byte, err error) {
	if err != nil {
		globals.Log.ERROR.Printf("Could not find searched user: %+v", err)
	} else {
		us.foundUserChan <- userID
	}
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

		userID, _, client := sessionInitialization()
		err := client.RegisterWithNodes()
		if err != nil {
			globals.Log.ERROR.Println(err)
		}
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
		// All other messages
		fallback := FallbackListener{}
		client.Listen(id.ZeroID, int32(cmixproto.Type_NO_TYPE),
			&fallback)

		// Log the user in, for now using the first gateway specified
		// This will also register the user email with UDB
		globals.Log.INFO.Println("Logging in...")
		cb := func(err error) {
			globals.Log.ERROR.Print(err)
		}

		err = client.InitListeners()
		if err != nil {
			globals.Log.FATAL.Panicf("Could not initialize receivers: %s\n", err)
		}

		err = client.StartMessageReceiver(cb)

		if err != nil {
			globals.Log.FATAL.Panicf("Could Not start message reciever: %s\n", err)
		}
		globals.Log.INFO.Println("Logged In!")

		if username != "" {
			err := client.RegisterWithUDB(username, 2*time.Minute)
			if err != nil {
				jww.ERROR.Printf("Could not register with UDB: %+v", err)
			}
		}

		cryptoType := parse.Unencrypted
		if end2end {
			cryptoType = parse.E2E
		}

		var recipientId *id.User

		if destinationUserId != 0 && destinationUserIDBase64 != "" {
			globals.Log.FATAL.Panicf("Two destiantions set for the message, can only have one")
		}

		if destinationUserId == 0 && destinationUserIDBase64 == "" {
			recipientId = userID
		} else if destinationUserIDBase64 != "" {
			recipientIdBytes, err := base64.StdEncoding.DecodeString(destinationUserIDBase64)
			if err != nil {
				globals.Log.FATAL.Panic("Could not decode the destination user ID")
			}
			recipientId = id.NewUserFromBytes(recipientIdBytes)

		} else {
			recipientId = id.NewUserFromUints(&[4]uint64{0, 0, 0, destinationUserId})
		}

		if message != "" {
			// Get the recipient's nick
			recipientNick := ""
			u, ok := user.Users.GetUser(recipientId)
			if ok {
				recipientNick = u.Username
			}

			// Handle sending to UDB
			if *recipientId == *bots.UdbID {
				parseUdbMessage(message, client)
			} else {
				// Handle sending to any other destination
				wireOut := api.FormatTextMessage(message)

				for i := uint(0); i < messageCnt; i++ {
					fmt.Printf("Sending Message to %s, %v: %s\n", base64.StdEncoding.EncodeToString(recipientId.Bytes()),
						recipientNick, message)
					if i != 0 {
						time.Sleep(1 * time.Second)
					}
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
		}

		var udbLister api.SearchCallback

		if searchForUser != "" {
			udbLister = newUserSearcher()
			client.SearchForUser(searchForUser, udbLister, 2*time.Minute)
		}

		if message != "" {
			// Wait up to 45s to receive a message
			lastCnt := int64(0)
			for end, timeout := false, time.After(45*time.Second); !end; {
				numMsgReceived := atomic.LoadInt64(&text.MessagesReceived)
				if numMsgReceived == int64(waitForMessages) {
					end = true
				}
				if numMsgReceived != lastCnt {
					lastCnt = numMsgReceived
					timeout = time.After(45 * time.Second)
				}

				select {
				case <-timeout:
					fmt.Printf("Timing out client, %v/%v "+
						"message(s) been received\n",
						numMsgReceived, waitForMessages)
					end = true
				default:
				}
			}
		}

		if searchForUser != "" {
			foundUser := <-udbLister.(*userSearcher).foundUserChan
			if isValidUser(foundUser) {
				userIDBase64 := base64.StdEncoding.EncodeToString(foundUser)
				globals.Log.INFO.Printf("Found User %s at ID: %s",
					searchForUser, userIDBase64)
			} else {
				globals.Log.INFO.Printf("Found User %s is invalid", searchForUser)
			}
		}

		if notificationToken != "" {
			err = client.RegisterForNotifications([]byte(notificationToken))
			if err != nil {
				globals.Log.FATAL.Printf("failed to register for notifications: %+v", err)
			}
		}

		//Logout
		err = client.Logout()

		if err != nil {
			globals.Log.ERROR.Printf("Could not logout: %s\n", err.Error())
			return
		}

	},
}

func isValidUser(usr []byte) bool {
	if len(usr) != id.UserLen {
		return false
	}
	for _, b := range usr {
		if b != 0 {
			return true
		}
	}
	return false
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
		"ID to sign in as. Does not register, must be an available precanned user")

	rootCmd.PersistentFlags().StringVarP(&registrationCode,
		"regcode", "r",
		"",
		"Registration Code with the registration server")

	rootCmd.PersistentFlags().StringVarP(&username,
		"username", "E",
		"",
		"Username to register for User Discovery")

	rootCmd.PersistentFlags().StringVarP(&sessionFile, "sessionfile", "f",
		"", "Passes a file path for loading a session.  "+
			"If the file doesn't exist the code will register the user and"+
			" store it there.  If not passed the session will be stored"+
			" to ram and lost when the cli finishes")

	rootCmd.PersistentFlags().StringVarP(&ndfPubKey,
		"ndfPubKeyCertPath",
		"p",
		"",
		"Path to the certificated containing the public key for the "+
			" network definition JSON file")

	rootCmd.PersistentFlags().StringVarP(&ndfPath,
		"ndf",
		"n",
		"ndf.json",
		"Path to the network definition JSON file")

	rootCmd.PersistentFlags().BoolVar(&skipNDFVerification,
		"skipNDFVerification",
		false,
		"Specifies if the NDF should be loaded without the signature")

	rootCmd.PersistentFlags().StringVarP(&sessFilePassword,
		"password",
		"P",
		"",
		"Password to the session file")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().StringVarP(&message, "message", "m", "", "Message to send")
	rootCmd.PersistentFlags().Uint64VarP(&destinationUserId, "destid", "d", 0,
		"ID to send message to")
	rootCmd.Flags().BoolVarP(&showVer, "version", "V", false,
		"Show the server version information.")

	rootCmd.Flags().StringVarP(&notificationToken, "nbRegistration", "x", "",
		"Register user with notification bot")

	rootCmd.PersistentFlags().BoolVarP(&end2end, "end2end", "", false,
		"Send messages with E2E encryption to destination user. Must have found each other via UDB first")

	rootCmd.PersistentFlags().StringSliceVarP(&keyParams, "keyParams", "",
		make([]string, 0), "Define key generation parameters. Pass values in comma separated list"+
			" in the following order: MinKeys,MaxKeys,NumRekeys,TTLScalar,MinNumKeys")

	rootCmd.Flags().BoolVarP(&noTLS, "noTLS", "", false,
		"Set to ignore tls. Connections will fail if the network requires tls. For debugging")

	rootCmd.Flags().StringVar(&privateKeyPath, "privateKey", "",
		"The path for a PEM encoded private key which will be used "+
			"to create the user")

	rootCmd.Flags().StringVar(&destinationUserIDBase64, "dest64", "",
		"Sets the destination user id encoded in base 64")

	rootCmd.Flags().UintVarP(&waitForMessages, "waitForMessages",
		"w", 1, "Denotes the number of messages the "+
			"client should receive before closing")

	rootCmd.Flags().StringVarP(&searchForUser, "SearchForUser", "s", "",
		"Sets the email to search for to find a user with user discovery")

	rootCmd.Flags().UintVarP(&messageTimeout, "messageTimeout",
		"t", 45, "The number of seconds to wait for "+
			"'waitForMessages' messages to arrive")

	rootCmd.Flags().UintVarP(&messageCnt, "messageCount",
		"c", 1, "The number of times to send the message")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {}

// initLog initializes logging thresholds and the log path.
func initLog() {
	globals.Log = jww.NewNotepad(jww.LevelError, jww.LevelInfo, os.Stdout,
		ioutil.Discard, "CLIENT", log.Ldate|log.Ltime)
	// If verbose flag set then log more info for debugging
	if verbose || viper.GetBool("verbose") {
		globals.Log.SetLogThreshold(jww.LevelDebug)
		globals.Log.SetStdoutThreshold(jww.LevelDebug)
		globals.Log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	} else {
		globals.Log.SetLogThreshold(jww.LevelInfo)
		globals.Log.SetStdoutThreshold(jww.LevelInfo)
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
