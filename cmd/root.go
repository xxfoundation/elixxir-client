///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Package cmd initializes the CLI and config parsers as well as the logger.
package cmd

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/interfaces/contact"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/switchboard"
	"gitlab.com/xx_network/primitives/id"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

// Execute adds all child commands to the root command and sets flags
// appropriately.  This is called by main.main(). It only needs to
// happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "client",
	Short: "Runs a client for cMix anonymous communication platform",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {

		client := initClient()

		user := client.GetUser()
		jww.INFO.Printf("User: %s", user.ReceptionID)
		writeContact(user.GetContact())

		// Set up reception handler
		swboard := client.GetSwitchboard()
		recvCh := make(chan message.Receive, 10000)
		listenerID := swboard.RegisterChannel("DefaultCLIReceiver",
			switchboard.AnyUser(), message.Text, recvCh)
		jww.INFO.Printf("Message ListenerID: %v", listenerID)

		// Set up auth request handler, which simply prints the
		// user id of the requestor.
		authMgr := client.GetAuthRegistrar()
		authMgr.AddGeneralRequestCallback(printChanRequest)

		// If unsafe channels, add auto-acceptor
		if viper.GetBool("unsafe-channel-creation") {
			authMgr.AddGeneralRequestCallback(func(
				requestor contact.Contact, message string) {
				jww.INFO.Printf("Got Request: %s", requestor.ID)
				err := client.ConfirmAuthenticatedChannel(
					requestor)
				if err != nil {
					jww.FATAL.Panicf("%+v", err)
				}
			})
		}

		err := client.StartNetworkFollower()
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		// Wait until connected or crash on timeout
		connected := make(chan bool, 10)
		client.GetHealth().AddChannel(connected)
		waitUntilConnected(connected)

		// Send Messages
		msgBody := viper.GetString("message")

		isPrecanPartner := false
		recipientContact := readContact()
		recipientID := recipientContact.ID

		// Try to get recipientID from destid
		if recipientID == nil {
			recipientID, isPrecanPartner = parseRecipient(
				viper.GetString("destid"))
		}

		// Set it to myself
		if recipientID == nil {
			jww.INFO.Printf("sending message to self")
			recipientID = user.ReceptionID
			recipientContact = user.GetContact()
		}

		time.Sleep(10 * time.Second)

		// Accept auth request for this recipient
		if viper.GetBool("accept-channel") {
			acceptChannel(client, recipientID)
		}

		// Send unsafe messages or not?
		unsafe := viper.GetBool("unsafe")
		if !unsafe {
			addAuthenticatedChannel(client, recipientID,
				recipientContact, isPrecanPartner)
		}

		msg := message.Send{
			Recipient:   recipientID,
			Payload:     []byte(msgBody),
			MessageType: message.Text,
		}
		paramsE2E := params.GetDefaultE2E()
		paramsUnsafe := params.GetDefaultUnsafe()

		sendCnt := int(viper.GetUint("sendCount"))
		sendDelay := time.Duration(viper.GetUint("sendDelay"))
		for i := 0; i < sendCnt; i++ {
			fmt.Printf("Sending to %s: %s\n", recipientID, msgBody)
			var roundIDs []id.Round
			var roundTimeout time.Duration
			if unsafe {
				roundIDs, err = client.SendUnsafe(msg,
					paramsUnsafe)
				roundTimeout = paramsUnsafe.Timeout
			} else {
				roundIDs, _, err = client.SendE2E(msg,
					paramsE2E)
				roundTimeout = paramsE2E.Timeout
			}
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}

			// Construct the callback function which prints out the rounds' results
			f := func(allRoundsSucceeded, timedOut bool,
				rounds map[id.Round]api.RoundResult) {
				printRoundResults(allRoundsSucceeded, timedOut, rounds, roundIDs, msg)
			}

			// Have the client report back the round results
			err = client.GetRoundResults(roundIDs, roundTimeout, f)
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}

			jww.INFO.Printf("RoundIDs: %+v\n", roundIDs)
			time.Sleep(sendDelay * time.Millisecond)
		}

		// Wait until message timeout or we receive enough then exit
		// TODO: Actually check for how many messages we've received
		expectedCnt := viper.GetUint("receiveCount")
		receiveCnt := uint(0)
		waitTimeout := time.Duration(viper.GetUint("waitTimeout"))
		timeoutTimer := time.NewTimer(waitTimeout * time.Second)
		done := false
		for !done && expectedCnt != 0 {
			select {
			case <-timeoutTimer.C:
				fmt.Println("Timed out!")
				done = true
				break
			case m := <-recvCh:
				fmt.Printf("Message received: %s\n", string(
					m.Payload))
				//fmt.Printf("%s", m.Timestamp)
				receiveCnt++
				if receiveCnt == expectedCnt {
					done = true
				}
				break
			}
		}
		fmt.Printf("Received %d\n", receiveCnt)
		// client.StopNetworkFollower(1 * time.Second)
		/*if err!=nil{
			fmt.Printf("Failed to cleanly close threads: %+v\n", err)
		}*/
		time.Sleep(10 * time.Second)
	},
}

// Helper function which prints the round resuls
func printRoundResults(allRoundsSucceeded, timedOut bool,
	rounds map[id.Round]api.RoundResult, roundIDs []id.Round, msg message.Send) {

	// Done as string slices for easy and human readable printing
	successfulRounds := make([]string, 0)
	failedRounds := make([]string, 0)
	timedOutRounds := make([]string, 0)

	for _, r := range roundIDs {
		// Group all round reports into a category based on their
		// result (successful, failed, or timed out)
		if result, exists := rounds[r]; exists {
			if result == api.Succeeded {
				successfulRounds = append(successfulRounds, strconv.Itoa(int(r)))
			} else if result == api.Failed {
				failedRounds = append(failedRounds, strconv.Itoa(int(r)))
			} else {
				timedOutRounds = append(timedOutRounds, strconv.Itoa(int(r)))
			}
		}
	}

	jww.INFO.Printf("Result of sending message \"%s\" to \"%v\":",
		msg.Payload, msg.Recipient)

	// Print out all rounds results, if they are populated
	if len(successfulRounds) > 0 {
		jww.INFO.Printf("\tRound(s) %v successful", strings.Join(successfulRounds, ","))
	}
	if len(failedRounds) > 0 {
		jww.ERROR.Printf("\tRound(s) %v failed", strings.Join(failedRounds, ","))
	}
	if len(timedOutRounds) > 0 {
		jww.ERROR.Printf("\tRound(s) %v timed " +
			"\n\tout (no network resolution could be found)", strings.Join(timedOutRounds, ","))
	}

}

func createClient() *api.Client {
	initLog(viper.GetBool("verbose"), viper.GetString("log"))
	jww.INFO.Printf(Version())

	pass := viper.GetString("password")
	storeDir := viper.GetString("session")
	regCode := viper.GetString("regcode")
	precannedID := viper.GetUint("sendid")

	//create a new client if none exist
	if _, err := os.Stat(storeDir); os.IsNotExist(err) {
		// Load NDF
		ndfPath := viper.GetString("ndf")
		ndfJSON, err := ioutil.ReadFile(ndfPath)
		if err != nil {
			jww.FATAL.Panicf(err.Error())
		}

		if precannedID != 0 {
			err = api.NewPrecannedClient(precannedID,
				string(ndfJSON), storeDir, []byte(pass))
		} else {
			err = api.NewClient(string(ndfJSON), storeDir,
				[]byte(pass), regCode)
		}

		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
	}

	netParams := params.GetDefaultNetwork()
	netParams.E2EParams.MinKeys = uint16(viper.GetUint("e2eMinKeys"))
	netParams.E2EParams.MaxKeys = uint16(viper.GetUint("e2eMaxKeys"))
	netParams.E2EParams.NumRekeys = uint16(
		viper.GetUint("e2eNumReKeys"))
	netParams.ForceHistoricalRounds = viper.GetBool("forceHistoricalRounds")

	client, err := api.OpenClient(storeDir, []byte(pass), netParams)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	return client
}

func initClient() *api.Client {
	createClient()

	pass := viper.GetString("password")
	storeDir := viper.GetString("session")

	netParams := params.GetDefaultNetwork()
	netParams.E2EParams.MinKeys = uint16(viper.GetUint("e2eMinKeys"))
	netParams.E2EParams.MaxKeys = uint16(viper.GetUint("e2eMaxKeys"))
	netParams.E2EParams.NumRekeys = uint16(
		viper.GetUint("e2eNumReKeys"))
	netParams.ForceHistoricalRounds = viper.GetBool("forceHistoricalRounds")

	//load the client
	client, err := api.Login(storeDir, []byte(pass), netParams)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	return client
}

func writeContact(c contact.Contact) {
	outfilePath := viper.GetString("writeContact")
	if outfilePath == "" {
		return
	}
	err := ioutil.WriteFile(outfilePath, c.Marshal(), 0644)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
}

func readContact() contact.Contact {
	inputFilePath := viper.GetString("destfile")
	if inputFilePath == "" {
		return contact.Contact{}
	}
	data, err := ioutil.ReadFile(inputFilePath)
	jww.INFO.Printf("Contact file size read in: %d", len(data))
	if err != nil {
		jww.FATAL.Panicf("Failed to read contact file: %+v", err)
	}
	c, err := contact.Unmarshal(data)
	if err != nil {
		jww.FATAL.Panicf("Failed to unmarshal contact: %+v", err)
	}
	return c
}

func acceptChannel(client *api.Client, recipientID *id.ID) {
	recipientContact, err := client.GetAuthenticatedChannelRequest(
		recipientID)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	err = client.ConfirmAuthenticatedChannel(
		recipientContact)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
}

func printChanRequest(requestor contact.Contact, message string) {
	msg := fmt.Sprintf("Authentication channel request from: %s\n",
		requestor.ID)
	jww.INFO.Printf(msg)
	fmt.Printf(msg)
	msg = fmt.Sprintf("Authentication channel request message: %s\n", message)
	jww.INFO.Printf(msg)
	fmt.Printf(msg)
}

func addAuthenticatedChannel(client *api.Client, recipientID *id.ID,
	recipient contact.Contact, isPrecanPartner bool) {
	if client.HasAuthenticatedChannel(recipientID) {
		jww.INFO.Printf("Authenticated channel already in place for %s",
			recipientID)
		return
	}

	var allowed bool
	if viper.GetBool("unsafe-channel-creation") {
		msg := "unsafe channel creation enabled\n"
		jww.WARN.Printf(msg)
		fmt.Printf("WARNING: %s", msg)
		allowed = true
	} else {
		allowed = askToCreateChannel(recipientID)
	}
	if !allowed {
		jww.FATAL.Panicf("User did not allow channel creation!")
	}

	// Check if a channel exists for this recipientID
	recipientContact, err := client.GetAuthenticatedChannelRequest(
		recipientID)
	if err == nil {
		jww.INFO.Printf("Accepting existing channel request for %s",
			recipientID)
		err := client.ConfirmAuthenticatedChannel(recipientContact)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
		return
	} else {
		recipientContact = recipient
	}

	msg := fmt.Sprintf("Adding authenticated channel for: %s\n",
		recipientID)
	jww.INFO.Printf(msg)
	fmt.Printf(msg)

	if isPrecanPartner {
		jww.WARN.Printf("Precanned user id detected: %s",
			recipientID)
		preUsr, err := client.MakePrecannedAuthenticatedChannel(
			getPrecanID(recipientID))
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
		// Sanity check, make sure user id's haven't changed
		preBytes := preUsr.ID.Bytes()
		idBytes := recipientID.Bytes()
		for i := 0; i < len(preBytes); i++ {
			if idBytes[i] != preBytes[i] {
				jww.FATAL.Panicf("no id match: %v %v",
					preBytes, idBytes)
			}
		}
	} else if recipientContact.ID != nil && recipientContact.DhPubKey != nil {
		me := client.GetUser().GetContact()
		jww.INFO.Printf("Requesting auth channel from: %s",
			recipientID)
		err := client.RequestAuthenticatedChannel(recipientContact,
			me, msg)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
	} else {
		jww.ERROR.Printf("Could not add auth channel for %s",
			recipientID)
	}
}

func waitUntilConnected(connected chan bool) {
	waitTimeout := time.Duration(viper.GetUint("waitTimeout"))
	timeoutTimer := time.NewTimer(waitTimeout * time.Second)
	isConnected := false
	//Wait until we connect or panic if we can't by a timeout
	for !isConnected {
		select {
		case isConnected = <-connected:
			jww.INFO.Printf("Network Status: %v\n",
				isConnected)
			break
		case <-timeoutTimer.C:
			jww.FATAL.Panic("timeout on connection")
		}
	}

	// Now start a thread to empty this channel and update us
	// on connection changes for debugging purposes.
	go func() {
		prev := true
		for {
			select {
			case isConnected = <-connected:
				if isConnected != prev {
					prev = isConnected
					jww.INFO.Printf(
						"Network Status Changed: %v\n",
						isConnected)
				}
				break
			}
		}
	}()
}

func getPrecanID(recipientID *id.ID) uint {
	return uint(recipientID.Bytes()[7])
}

func parseRecipient(idStr string) (*id.ID, bool) {
	if idStr == "0" {
		return nil, false
	}

	var recipientID *id.ID
	if strings.HasPrefix(idStr, "0x") {
		recipientID = getUIDFromHexString(idStr[2:])
	} else if strings.HasPrefix(idStr, "b64:") {
		recipientID = getUIDFromb64String(idStr[4:])
	} else {
		recipientID = getUIDFromString(idStr)
	}
	// check if precanned
	rBytes := recipientID.Bytes()
	for i := 0; i < 32; i++ {
		if i != 7 && rBytes[i] != 0 {
			return recipientID, false
		}
	}
	if rBytes[7] != byte(0) && rBytes[7] <= byte(40) {
		return recipientID, true
	}
	jww.FATAL.Panicf("error recipient id parse failure: %+v", recipientID)
	return recipientID, false
}

func getUIDFromHexString(idStr string) *id.ID {
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

func getUIDFromb64String(idStr string) *id.ID {
	idBytes, err := base64.StdEncoding.DecodeString(idStr)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	ID, err := id.Unmarshal(idBytes)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	return ID
}

func getUIDFromString(idStr string) *id.ID {
	idInt, err := strconv.Atoi(idStr)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	if idInt > 255 {
		jww.FATAL.Panicf("cannot convert integers above 255. Use 0x " +
			"or b64: representation")
	}
	idBytes := make([]byte, 33)
	binary.BigEndian.PutUint64(idBytes, uint64(idInt))
	idBytes[32] = byte(id.User)
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

func askToCreateChannel(recipientID *id.ID) bool {
	for {
		fmt.Printf("This is the first time you have messaged %v, "+
			"are you sure? (yes/no) ", recipientID)
		var input string
		fmt.Scanln(&input)
		if input == "yes" {
			return true
		}
		if input == "no" {
			return false
		}
		fmt.Printf("Please answer 'yes' or 'no'\n")
	}
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
	rootCmd.PersistentFlags().BoolP("verbose", "v", false,
		"Verbose mode for debugging")
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

	rootCmd.PersistentFlags().StringP("session", "s",
		"", "Sets the initial storage directory for "+
			"client session data")
	viper.BindPFlag("session", rootCmd.PersistentFlags().Lookup("session"))

	rootCmd.PersistentFlags().StringP("writeContact", "w",
		"-", "Write contact information, if any, to this file, "+
			" defaults to stdout")
	viper.BindPFlag("writeContact", rootCmd.PersistentFlags().Lookup(
		"writeContact"))

	rootCmd.PersistentFlags().StringP("password", "p", "",
		"Password to the session file")
	viper.BindPFlag("password", rootCmd.PersistentFlags().Lookup(
		"password"))

	rootCmd.PersistentFlags().StringP("ndf", "n", "ndf.json",
		"Path to the network definition JSON file")
	viper.BindPFlag("ndf", rootCmd.PersistentFlags().Lookup("ndf"))

	rootCmd.PersistentFlags().StringP("log", "l", "-",
		"Path to the log output path (- is stdout)")
	viper.BindPFlag("log", rootCmd.PersistentFlags().Lookup("log"))

	rootCmd.Flags().StringP("regcode", "", "",
		"Identity code (optional)")
	viper.BindPFlag("regcode", rootCmd.Flags().Lookup("regcode"))

	rootCmd.PersistentFlags().StringP("message", "m", "",
		"Message to send")
	viper.BindPFlag("message", rootCmd.PersistentFlags().Lookup("message"))

	rootCmd.Flags().UintP("sendid", "", 0,
		"Use precanned user id (must be between 1 and 40, inclusive)")
	viper.BindPFlag("sendid", rootCmd.Flags().Lookup("sendid"))

	rootCmd.Flags().StringP("destid", "d", "0",
		"ID to send message to (if below 40, will be precanned. Use "+
			"'0x' or 'b64:' for hex and base64 representations)")
	viper.BindPFlag("destid", rootCmd.Flags().Lookup("destid"))

	rootCmd.Flags().StringP("destfile", "",
		"", "Read this contact file for the destination id")
	viper.BindPFlag("destfile", rootCmd.Flags().Lookup("destfile"))

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

	rootCmd.Flags().BoolP("unsafe", "", false,
		"Send raw, unsafe messages without e2e encryption.")
	viper.BindPFlag("unsafe", rootCmd.Flags().Lookup("unsafe"))

	rootCmd.Flags().BoolP("unsafe-channel-creation", "", false,
		"Turns off the user identity authenticated channel check, "+
			"automatically approving authenticated channels")
	viper.BindPFlag("unsafe-channel-creation",
		rootCmd.Flags().Lookup("unsafe-channel-creation"))

	rootCmd.Flags().BoolP("accept-channel", "", false,
		"Accept the channel request for the corresponding recipient ID")
	viper.BindPFlag("accept-channel",
		rootCmd.Flags().Lookup("accept-channel"))

	rootCmd.Flags().BoolP("forceHistoricalRounds", "", false,
		"Force all rounds to be sent to historical round retrieval")
	viper.BindPFlag("forceHistoricalRounds",
		rootCmd.Flags().Lookup("forceHistoricalRounds"))

	// E2E Params
	defaultE2EParams := params.GetDefaultE2ESessionParams()
	rootCmd.Flags().UintP("e2eMinKeys",
		"", uint(defaultE2EParams.MinKeys),
		"Minimum number of keys used before requesting rekey")
	viper.BindPFlag("e2eMinKeys", rootCmd.Flags().Lookup("e2eMinKeys"))
	rootCmd.Flags().UintP("e2eMaxKeys",
		"", uint(defaultE2EParams.MaxKeys),
		"Max keys used before blocking until a rekey completes")
	viper.BindPFlag("e2eMaxKeys", rootCmd.Flags().Lookup("e2eMaxKeys"))
	rootCmd.Flags().UintP("e2eNumReKeys",
		"", uint(defaultE2EParams.NumRekeys),
		"Number of rekeys reserved for rekey operations")
	viper.BindPFlag("e2eNumReKeys", rootCmd.Flags().Lookup("e2eNumReKeys"))
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
