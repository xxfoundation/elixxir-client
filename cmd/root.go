////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package cmd initializes the CLI and config parsers as well as the logger.
package cmd

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/primitives/switchboard"
	"gitlab.com/elixxir/primitives/utils"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"io/ioutil"
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
var precanned = false
var logPath string = ""
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

func sessionInitialization() (*id.ID, string, *api.Client) {
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
			return &id.ZeroUser, "", nil
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
				return &id.ZeroUser, "", nil
			}
		}
		//Initialize client with OS Storage
		client, err = api.NewClient(nil, sessionA, sessionB, ndfJSON)
		if err != nil {
			globals.Log.ERROR.Printf("Could Not Initialize OS Storage: %s\n", err.Error())
			return &id.ZeroUser, "", nil
		}
		globals.Log.INFO.Println("Initialized OS Storage")

	}

	if noBlockingTransmission {
		globals.Log.INFO.Println("Disabling Blocking Transmissions")
		client.DisableBlockingTransmission()
	}

	// Handle parsing gateway addresses from the config file

	//REVIEWER NOTE: Possibly need to remove/rearrange this,
	// now that client may not know gw's upon client creation
	/*gateways := client.GetNDF().Gateways
	// If gwAddr was not passed via command line, check config file
	if len(gateways) < 1 {
		// No gateways in config file or passed via command line
		globals.Log.ERROR.Printf("Error: No gateway specified! Add to" +
			" configuration file or pass via command line using -g!\n")
		return &id.ZeroUser, "", nil
	}*/

	if noTLS {
		client.DisableTls()
	}

	// InitNetwork to gateways, notificationBot and reg server
	err = client.InitNetwork()
	if err != nil {
		globals.Log.FATAL.Panicf("Could not call connect on client: %+v", err)
	}

	client.SetRateLimiting(rateLimiting)

	// Holds the User ID
	var uid *id.ID

	// Register a new user if requested
	if register {
		globals.Log.INFO.Println("Registering...")

		regCode := registrationCode
		// If precanned user, use generated code instead
		if userId != 0 {
			precanned = true
			uid := new(id.ID)
			binary.BigEndian.PutUint64(uid[:], userId)
			uid.SetType(id.User)
			regCode = userRegistry.RegistrationCode(uid)
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

		//Attempt to register user with same keys until a success occurs
		for errRegister != nil {
			_, errRegister = client.RegisterWithPermissioning(precanned, regCode)
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

		uid = client.GetCurrentUser()

		userbase64 := base64.StdEncoding.EncodeToString(uid[:])
		globals.Log.INFO.Printf("Registered as user (uid, the var) %v", uid)
		globals.Log.INFO.Printf("Registered as user (userID, the global) %v", userId)
		globals.Log.INFO.Printf("Successfully registered user %s!", userbase64)

	} else {
		// hack for session persisting with cmd line
		// doesn't support non pre canned users
		uid := new(id.ID)
		binary.BigEndian.PutUint64(uid[:], userId)
		uid.SetType(id.User)
		globals.Log.INFO.Printf("Skipped Registration, user: %v", uid)
	}

	if !precanned {
		// If we are sending to a non precanned user we retrieve the uid from the session returned by client.login
		uid, err = client.Login(sessFilePassword)
	} else {
		_, err = client.Login(sessFilePassword)
	}

	if err != nil {
		globals.Log.FATAL.Panicf("Could not login: %v", err)
	}
	return uid, client.GetUsername(), client
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
		sender, ok := userRegistry.Users.GetUser(message.Sender)
		var senderNick string
		if !ok {
			globals.Log.ERROR.Printf("Couldn't get sender %v", message.Sender)
		} else {
			senderNick = sender.Username
		}
		atomic.AddInt64(&l.MessagesReceived, 1)
		globals.Log.INFO.Printf("Message of type %v from %q, %v received with fallback: %s\n",
			message.MessageType, printIDNice(message.Sender), senderNick,
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

	sender, ok := userRegistry.Users.GetUser(message.Sender)
	var senderNick string
	if !ok {
		globals.Log.INFO.Printf("First message from sender %v", printIDNice(message.Sender))
		u := userRegistry.Users.NewUser(message.Sender, base64.StdEncoding.EncodeToString(message.Sender[:]))
		userRegistry.Users.UpsertUser(u)
		senderNick = u.Username
	} else {
		senderNick = sender.Username
	}
	logMsg := fmt.Sprintf("Message from %v, %v Received: %s\n",
		printIDNice(message.Sender),
		senderNick, result.Message)
	globals.Log.INFO.Printf("%s -- Timestamp: %s\n", logMsg,
		message.Timestamp.String())
	fmt.Printf(logMsg)

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
		if !verbose && viper.Get("verbose") != nil {
			verbose = viper.GetBool("verbose")
		}
		if logPath == "" && viper.Get("logPath") != nil {
			logPath = viper.GetString("logPath")
		}
		// Disable stdout output
		jww.SetStdoutOutput(ioutil.Discard)
		// Use log file
		logOutput, err := os.OpenFile(logPath,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err.Error())
		}
		jww.SetLogOutput(logOutput)
		if verbose {
			jww.SetLogThreshold(verbose)
		}
	},
}

func isValidUser(usr []byte) (bool, *id.ID) {
	if len(usr) != id.ArrIDLen {
		return false, nil
	}
	for _, b := range usr {
		if b != 0 {
			uid, err := id.Unmarshal(usr)
			if err != nil {
				globals.Log.WARN.Printf("Could not unmarshal user: %s", err)
				return false, nil
			}
			return true, uid
		}
	}
	return false, nil
}

// init is the initialization function for Cobra which defines commands
// and flags.
func init() {
	// NOTE: The point of init() is to be declarative.
	// There is one init in each sub command. Do not put variable declarations
	// here, and ensure all the Flags are of the *P variety, unless there's a
	// very good reason not to have them as local params to sub command."
	cobra.OnInitialize(initConfig)

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

	rootCmd.Flags().StringVarP(&notificationToken, "nbRegistration", "x", "",
		"Token to register user with notification bot")

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

	rootCmd.Flags().StringVarP(&logPath, "log", "l", "",
		"Print logs to specified log file, not stdout")

	rootCmd.Flags().UintVarP(&messageTimeout, "messageTimeout",
		"t", 45, "The number of seconds to wait for "+
			"'waitForMessages' messages to arrive")

	rootCmd.Flags().UintVarP(&messageCnt, "messageCount",
		"c", 1, "The number of times to send the message")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {}

// returns a simple numerical id if the user is a precanned user, otherwise
// returns the normal string of the userID
func printIDNice(uid *id.ID) string {

	for index, puid := range precannedIDList {
		if uid.Cmp(puid) {
			return strconv.Itoa(index + 1)
		}
	}

	return uid.String()
}

// build a list of precanned ids to use for comparision for nicer user id output
var precannedIDList = buildPrecannedIDList()

func buildPrecannedIDList() []*id.ID {

	idList := make([]*id.ID, 40)

	for i := 0; i < 40; i++ {
		uid := new(id.ID)
		binary.BigEndian.PutUint64(uid[:], uint64(i+1))
		uid.SetType(id.User)
		idList[i] = uid
	}

	return idList
}
