////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
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
	"gitlab.com/xx_network/primitives/id"
	"io/ioutil"
	"os"
	"strconv"
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

// func setKeyParams(client *api.Client) {
// 	jww.DEBUG.Printf("Trying to parse key parameters...")
// 	minKeys, err := strconv.Atoi(keyParams[0])
// 	if err != nil {
// 		return
// 	}

// 	maxKeys, err := strconv.Atoi(keyParams[1])
// 	if err != nil {
// 		return
// 	}

// 	numRekeys, err := strconv.Atoi(keyParams[2])
// 	if err != nil {
// 		return
// 	}

// 	ttlScalar, err := strconv.ParseFloat(keyParams[3], 64)
// 	if err != nil {
// 		return
// 	}

// 	minNumKeys, err := strconv.Atoi(keyParams[4])
// 	if err != nil {
// 		return
// 	}

// 	jww.DEBUG.Printf("Setting key generation parameters: %d, %d, %d, %f, %d",
// 		minKeys, maxKeys, numRekeys, ttlScalar, minNumKeys)

// 	params := client.GetKeyParams()
// 	params.MinKeys = uint16(minKeys)
// 	params.MaxKeys = uint16(maxKeys)
// 	params.NumRekeys = uint16(numRekeys)
// 	params.TTLScalar = ttlScalar
// 	params.MinNumKeys = uint16(minNumKeys)
// }

// type FallbackListener struct {
// 	MessagesReceived int64
// }

// func (l *FallbackListener) Hear(item switchboard.Item, isHeardElsewhere bool, i ...interface{}) {
// 	if !isHeardElsewhere {
// 		message := item.(*parse.Message)
// 		sender, ok := userRegistry.Users.GetUser(message.Sender)
// 		var senderNick string
// 		if !ok {
// 			jww.ERROR.Printf("Couldn't get sender %v", message.Sender)
// 		} else {
// 			senderNick = sender.Username
// 		}
// 		atomic.AddInt64(&l.MessagesReceived, 1)
// 		jww.INFO.Printf("Message of type %v from %q, %v received with fallback: %s\n",
// 			message.MessageType, printIDNice(message.Sender), senderNick,
// 			string(message.Body))
// 	}
// }

// type TextListener struct {
// 	MessagesReceived int64
// }

// func (l *TextListener) Hear(item switchboard.Item, isHeardElsewhere bool, i ...interface{}) {
// 	message := item.(*parse.Message)
// 	jww.INFO.Println("Hearing a text message")
// 	result := cmixproto.TextMessage{}
// 	err := proto.Unmarshal(message.Body, &result)
// 	if err != nil {
// 		jww.ERROR.Printf("Error unmarshaling text message: %v\n",
// 			err.Error())
// 	}

// 	sender, ok := userRegistry.Users.GetUser(message.Sender)
// 	var senderNick string
// 	if !ok {
// 		jww.INFO.Printf("First message from sender %v", printIDNice(message.Sender))
// 		u := userRegistry.Users.NewUser(message.Sender, base64.StdEncoding.EncodeToString(message.Sender[:]))
// 		userRegistry.Users.UpsertUser(u)
// 		senderNick = u.Username
// 	} else {
// 		senderNick = sender.Username
// 	}
// 	logMsg := fmt.Sprintf("Message from %v, %v Received: %s\n",
// 		printIDNice(message.Sender),
// 		senderNick, result.Message)
// 	jww.INFO.Printf("%s -- Timestamp: %s\n", logMsg,
// 		message.Timestamp.String())
// 	fmt.Printf(logMsg)

// 	atomic.AddInt64(&l.MessagesReceived, 1)
// }

// type userSearcher struct {
// 	foundUserChan chan []byte
// }

// func newUserSearcher() api.SearchCallback {
// 	us := userSearcher{}
// 	us.foundUserChan = make(chan []byte)
// 	return &us
// }

// func (us *userSearcher) Callback(userID, pubKey []byte, err error) {
// 	if err != nil {
// 		jww.ERROR.Printf("Could not find searched user: %+v", err)
// 	} else {
// 		us.foundUserChan <- userID
// 	}
// }

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
			jww.SetLogThreshold(jww.LevelTrace)
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
				jww.WARN.Printf("Could not unmarshal user: %s", err)
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
