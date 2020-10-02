////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package cmd initializes the CLI and config parsers as well as the logger.
package cmd

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/xx_network/primitives/id"
	"io/ioutil"
	"os"
	"strconv"
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
		initLog(viper.GetBool("verbose"), viper.GetString("log"))
		jww.INFO.Printf(Version())

		pass := viper.GetString("password")
		storeDir := viper.GetString("session")
		regCode := viper.GetString("regcode")

		var client *api.Client
		if _, err := os.Stat(storeDir); os.IsNotExist(err) {
			// Load NDF
			ndfPath := viper.GetString("ndf")
			ndfJSON, err := ioutil.ReadFile(ndfPath)
			if err != nil {
				jww.FATAL.Panicf(err.Error())
			}

			client, err = api.NewClient(string(ndfJSON), storeDir,
				[]byte(pass), regCode)
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}
		} else {
			client, err = api.LoadClient(storeDir, []byte(pass))
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}
		}

		user := client.GetUser()
		jww.INFO.Printf("%s", user.ID)

		err := client.StartNetworkFollower()
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		time.Sleep(10 * time.Second)

		// Wait until connected or crash on timeout
		connected := make(chan bool, 1)
		client.GetHealth().AddChannel(connected)
		waitTimeout := time.Duration(viper.GetUint("waitTimeout"))
		timeoutTick := time.NewTicker(waitTimeout * time.Second)
		isConnected := false
		for !isConnected {
			select {
			case isConnected = <-connected:
				jww.INFO.Printf("health status: %b\n",
					isConnected)
				break
			case <-timeoutTick.C:
				jww.FATAL.Panic("timeout on connection")
			}
		}

		// Send Messages
		msgBody := viper.GetString("message")
		recipientID := getUIDFromString(viper.GetString("destid"))

		msg := client.NewCMIXMessage(recipientID, []byte(msgBody))
		params := params.GetDefaultCMIX()

		sendCnt := int(viper.GetUint("sendCount"))
		sendDelay := time.Duration(viper.GetUint("sendDelay"))
		for i := 0; i < sendCnt; i++ {
			fmt.Printf("Sending to %s: %s\n", recipientID, msgBody)
			roundID, err := client.SendCMIX(msg, params)
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}
			jww.INFO.Printf("RoundID: %d\n", roundID)
			time.Sleep(sendDelay * time.Millisecond)
		}

		// Wait until message timeout or we receive enough then exit
		// TODO: Actually check for how many messages we've received
		receiveCnt := viper.GetUint("receiveCount")
		timeoutTick = time.NewTicker(waitTimeout * time.Second)
		done := false
		for !done {
			select {
			case <-timeoutTick.C:
				fmt.Println("Timed out!")
				done = true
				break
			}
		}
		fmt.Printf("Received %d", receiveCnt)
	},
}

func getUIDFromString(idStr string) *id.ID {
	idBytes, err := hex.DecodeString(fmt.Sprintf("%0*d%s",
		66-len(idStr), 0, idStr))
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	ID, err := id.Unmarshal(idBytes)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	return ID
}

func initLog(verbose bool, logPath string) {
	if logPath != "-" && logPath != "" {
		// Disable stdout output
		jww.SetStdoutOutput(ioutil.Discard)
		// Use log file
		logOutput, err := os.OpenFile(logPath,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err.Error())
		}
		jww.SetLogOutput(logOutput)
	}

	if verbose {
		jww.SetStdoutThreshold(jww.LevelTrace)
		jww.SetLogThreshold(jww.LevelTrace)
	} else {
		jww.SetStdoutThreshold(jww.LevelInfo)
		jww.SetLogThreshold(jww.LevelInfo)
	}

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
	rootCmd.Flags().BoolP("verbose", "v", false,
		"Verbose mode for debugging")
	viper.BindPFlag("verbose", rootCmd.Flags().Lookup("verbose"))

	rootCmd.Flags().StringP("session", "s",
		"", "Sets the initial username and the directory for "+
			"client storage")
	viper.BindPFlag("session", rootCmd.Flags().Lookup("session"))

	rootCmd.Flags().StringP("password", "p", "",
		"Password to the session file")
	viper.BindPFlag("password", rootCmd.Flags().Lookup("password"))

	rootCmd.Flags().StringP("ndf", "n", "ndf.json",
		"Path to the network definition JSON file")
	viper.BindPFlag("ndf", rootCmd.Flags().Lookup("ndf"))

	rootCmd.Flags().StringP("log", "l", "-",
		"Path to the log output path (- is stdout)")
	viper.BindPFlag("log", rootCmd.Flags().Lookup("log"))

	rootCmd.Flags().StringP("regcode", "", "",
		"Registration code (optional)")
	viper.BindPFlag("regcode", rootCmd.Flags().Lookup("regcode"))

	rootCmd.Flags().StringP("message", "m", "", "Message to send")
	viper.BindPFlag("message", rootCmd.Flags().Lookup("message"))

	rootCmd.Flags().StringP("destid", "d", "0",
		"ID to send message to (hexadecimal string up to 256 bits)")
	viper.BindPFlag("destid", rootCmd.Flags().Lookup("destid"))

	rootCmd.Flags().UintP("sendCount",
		"", 1, "The number of times to send the message")
	viper.BindPFlag("sendCount", rootCmd.Flags().Lookup("sendCount"))
	rootCmd.Flags().UintP("sendDelay",
		"", 500, "The delay between sending the messages in ms")
	viper.BindPFlag("sendDelay", rootCmd.Flags().Lookup("sendDelay"))

	rootCmd.Flags().UintP("receiveCount",
		"", 1, "How many messages we should wait for before quitting")
	viper.BindPFlag("receiveCount", rootCmd.Flags().Lookup("receiveCount"))
	rootCmd.Flags().UintP("waitTimeout", "", 15,
		"The number of seconds to wait for messages to arrive")
	viper.BindPFlag("waitTimeout",
		rootCmd.Flags().Lookup("waitTimeout"))

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().StringVarP(&notificationToken, "nbRegistration", "x", "",
		"Token to register user with notification bot")

	rootCmd.PersistentFlags().BoolVarP(&end2end, "end2end", "", false,
		"Send messages with E2E encryption to destination user. Must have found each other via UDB first")

	rootCmd.PersistentFlags().StringSliceVarP(&keyParams, "keyParams", "",
		make([]string, 0), "Define key generation parameters. Pass values in comma separated list"+
			" in the following order: MinKeys,MaxKeys,NumRekeys,TTLScalar,MinNumKeys")

	rootCmd.Flags().StringVar(&destinationUserIDBase64, "dest64", "",
		"Sets the destination user id encoded in base 64")

	rootCmd.Flags().UintVarP(&waitForMessages, "waitForMessages",
		"w", 1, "Denotes the number of messages the "+
			"client should receive before closing")

	// rootCmd.Flags().StringVarP(&searchForUser, "SearchForUser", "s", "",
	// 	"Sets the email to search for to find a user with user discovery")
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
