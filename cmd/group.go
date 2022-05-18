///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// The group subcommand allows creation and sending messages to groups

package cmd

import (
	"bufio"
	"fmt"
	"gitlab.com/elixxir/client/api/messenger"
	"os"
	"time"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/groupChat"
	"gitlab.com/elixxir/client/groupChat/groupStore"
	"gitlab.com/xx_network/primitives/id"
)

// groupCmd represents the base command when called without any subcommands
var groupCmd = &cobra.Command{
	Use:   "group",
	Short: "Group commands for cMix client",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {

		client := initClient()

		// Print user's reception ID
		user := client.GetUser()
		jww.INFO.Printf("User: %s", user.ReceptionID)

		err := client.StartNetworkFollower(5 * time.Second)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		// Initialize the group chat manager
		groupManager, recChan, reqChan := initGroupManager(client)

		// Wait until connected or crash on timeout
		connected := make(chan bool, 10)
		client.GetCmix().AddHealthCallback(
			func(isconnected bool) {
				connected <- isconnected
			})
		waitUntilConnected(connected)

		// After connection, make sure we have registered with at least 85% of
		// the nodes
		for numReg, total := 1, 100; numReg < (total*3)/4; {
			time.Sleep(1 * time.Second)
			numReg, total, err = client.GetNodeRegistrationStatus()
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}

			jww.INFO.Printf("Registering with nodes (%d/%d)...", numReg, total)
		}

		// get group message and name
		msgBody := []byte(viper.GetString("message"))
		name := []byte(viper.GetString("name"))
		timeout := viper.GetDuration("receiveTimeout")

		if viper.IsSet("create") {
			filePath := viper.GetString("create")
			createGroup(name, msgBody, filePath, groupManager)
		}

		if viper.IsSet("resend") {
			groupIdString := viper.GetString("resend")
			resendRequests(groupIdString, groupManager)
		}

		if viper.GetBool("join") {
			joinGroup(reqChan, timeout, groupManager)
		}

		if viper.IsSet("leave") {
			groupIdString := viper.GetString("leave")
			leaveGroup(groupIdString, groupManager)
		}

		if viper.IsSet("sendMessage") {
			groupIdString := viper.GetString("sendMessage")
			sendGroup(groupIdString, msgBody, groupManager)
		}

		if viper.IsSet("wait") {
			numMessages := viper.GetUint("wait")
			messageWait(numMessages, timeout, recChan)
		}

		if viper.GetBool("list") {
			listGroups(groupManager)
		}

		if viper.IsSet("show") {
			groupIdString := viper.GetString("show")
			showGroup(groupIdString, groupManager)
		}
	},
}

// initGroupManager creates a new group chat manager and starts the process
// service.
func initGroupManager(client *messenger.Client) (*groupChat.Manager,
	chan groupChat.MessageReceive, chan groupStore.Group) {
	recChan := make(chan groupChat.MessageReceive, 10)
	receiveCb := func(msg groupChat.MessageReceive) {
		recChan <- msg
	}

	reqChan := make(chan groupStore.Group, 10)
	requestCb := func(g groupStore.Group) {
		reqChan <- g
	}

	jww.INFO.Print("Creating new group manager.")
	manager, err := groupChat.NewManager(client.GetCmix(),
		client.GetE2E(), client.GetStorage().GetReceptionID(),
		client.GetRng(), client.GetStorage().GetE2EGroup(),
		client.GetStorage().GetKV(), requestCb, receiveCb)
	if err != nil {
		jww.FATAL.Panicf("Failed to initialize group chat manager: %+v", err)
	}

	return manager, recChan, reqChan
}

// createGroup creates a new group with the provided name and sends out requests
// to the list of user IDs found at the given file path.
func createGroup(name, msg []byte, filePath string, gm *groupChat.Manager) {
	userIdStrings := ReadLines(filePath)
	userIDs := make([]*id.ID, 0, len(userIdStrings))
	for _, userIdStr := range userIdStrings {
		userID, _ := parseRecipient(userIdStr)
		userIDs = append(userIDs, userID)
	}

	grp, rids, status, err := gm.MakeGroup(userIDs, name, msg)
	if err != nil {
		jww.FATAL.Panicf("Failed to create new group: %+v", err)
	}

	// Integration grabs the group ID from this line
	jww.INFO.Printf("NewGroupID: b64:%s", grp.ID)
	jww.INFO.Printf("Created Group: Requests:%s on rounds %#v, %v",
		status, rids, grp)
	fmt.Printf("Created new group with name %q and message %q\n", grp.Name,
		grp.InitMessage)
}

// resendRequests resends group requests for the group ID.
func resendRequests(groupIdString string, gm *groupChat.Manager) {
	groupID, _ := parseRecipient(groupIdString)
	rids, status, err := gm.ResendRequest(groupID)
	if err != nil {
		jww.FATAL.Panicf("Failed to resend requests to group %s: %+v",
			groupID, err)
	}

	jww.INFO.Printf("Resending requests to group %s: %v, %s",
		groupID, rids, status)
	fmt.Println("Resending group requests to group.")
}

// joinGroup joins a group when a request is received on the group request
// channel.
func joinGroup(reqChan chan groupStore.Group, timeout time.Duration,
	gm *groupChat.Manager) {
	jww.INFO.Print("Waiting for group request to be received.")
	fmt.Println("Waiting for group request to be received.")

	select {
	case grp := <-reqChan:
		err := gm.JoinGroup(grp)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		jww.INFO.Printf("Joined group: %s", grp.ID)
		fmt.Printf("Joined group with name %q and message %q\n",
			grp.Name, grp.InitMessage)
	case <-time.NewTimer(timeout).C:
		jww.INFO.Printf("Timed out after %s waiting for group request.", timeout)
		fmt.Println("Timed out waiting for group request.")
		return
	}
}

// leaveGroup leaves the group.
func leaveGroup(groupIdString string, gm *groupChat.Manager) {
	groupID, _ := parseRecipient(groupIdString)
	jww.INFO.Printf("Leaving group %s.", groupID)

	err := gm.LeaveGroup(groupID)
	if err != nil {
		jww.FATAL.Panicf("Failed to leave group %s: %+v", groupID, err)
	}

	jww.INFO.Printf("Left group: %s", groupID)
	fmt.Println("Left group.")
}

// sendGroup send the message to the group.
func sendGroup(groupIdString string, msg []byte, gm *groupChat.Manager) {
	groupID, _ := parseRecipient(groupIdString)

	jww.INFO.Printf("Sending to group %s message %q", groupID, msg)

	rid, timestamp, _, err := gm.Send(groupID, msg)
	if err != nil {
		jww.FATAL.Panicf("Sending message to group %s: %+v", groupID, err)
	}

	jww.INFO.Printf("Sent to group %s on round %d at %s",
		groupID, rid, timestamp)
	fmt.Printf("Sent message %q to group.\n", msg)
}

// messageWait waits for the given number of messages to be received on the
// groupChat.MessageReceive channel.
func messageWait(numMessages uint, timeout time.Duration,
	recChan chan groupChat.MessageReceive) {
	jww.INFO.Printf("Waiting for %d group message(s) to be received.", numMessages)
	fmt.Printf("Waiting for %d group message(s) to be received.\n", numMessages)

	for i := uint(0); i < numMessages; {
		select {
		case msg := <-recChan:
			i++
			jww.INFO.Printf("Received group message %d/%d: %s", i, numMessages, msg)
			fmt.Printf("Received group message: %q\n", msg.Payload)
		case <-time.NewTimer(timeout).C:
			jww.INFO.Printf("Timed out after %s waiting for group message.", timeout)
			fmt.Printf("Timed out waiting for %d group message(s).\n", numMessages)
			return
		}
	}
}

// listGroups prints a list of all groups.
func listGroups(gm *groupChat.Manager) {
	for i, gid := range gm.GetGroups() {
		jww.INFO.Printf("Group %d: %s", i, gid)
	}

	fmt.Printf("Printed list of %d groups.\n", gm.NumGroups())
}

// showGroup prints all the information of the group.
func showGroup(groupIdString string, gm *groupChat.Manager) {
	groupID, _ := parseRecipient(groupIdString)

	grp, ok := gm.GetGroup(groupID)
	if !ok {
		jww.FATAL.Printf("Could not find group: %s", groupID)
	}

	jww.INFO.Printf("Show group %#v", grp)
	fmt.Printf("Got group with name %q and message %q\n",
		grp.Name, grp.InitMessage)
}

// ReadLines returns each line in a file as a string.
func ReadLines(fileName string) []string {
	file, err := os.Open(fileName)
	if err != nil {
		jww.FATAL.Panicf(err.Error())
	}
	defer func(file *os.File) {
		err = file.Close()
		if err != nil {
			jww.FATAL.Panicf("Failed to close file: %+v", err)
		}
	}(file)

	var res []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		res = append(res, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		jww.FATAL.Panicf(err.Error())
	}
	return res
}

func init() {
	groupCmd.Flags().String("create", "",
		"Create a group with from the list of contact file paths.")
	err := viper.BindPFlag("create", groupCmd.Flags().Lookup("create"))
	checkBindErr(err, "create")

	groupCmd.Flags().String("name", "Group Name",
		"The name of the new group to create.")
	err = viper.BindPFlag("name", groupCmd.Flags().Lookup("name"))
	checkBindErr(err, "name")

	groupCmd.Flags().String("resend", "",
		"Resend invites for all users in this group ID.")
	err = viper.BindPFlag("resend", groupCmd.Flags().Lookup("resend"))
	checkBindErr(err, "resend")

	groupCmd.Flags().Bool("join", false,
		"Waits for group request joins the group.")
	err = viper.BindPFlag("join", groupCmd.Flags().Lookup("join"))
	checkBindErr(err, "join")

	groupCmd.Flags().String("leave", "",
		"Leave this group ID.")
	err = viper.BindPFlag("leave", groupCmd.Flags().Lookup("leave"))
	checkBindErr(err, "leave")

	groupCmd.Flags().String("sendMessage", "",
		"Send message to this group ID.")
	err = viper.BindPFlag("sendMessage", groupCmd.Flags().Lookup("sendMessage"))
	checkBindErr(err, "sendMessage")

	groupCmd.Flags().Uint("wait", 0,
		"Waits for number of messages to be received.")
	err = viper.BindPFlag("wait", groupCmd.Flags().Lookup("wait"))
	checkBindErr(err, "wait")

	groupCmd.Flags().Duration("receiveTimeout", time.Minute,
		"Amount of time to wait for a group request or message before timing out.")
	err = viper.BindPFlag("receiveTimeout", groupCmd.Flags().Lookup("receiveTimeout"))
	checkBindErr(err, "receiveTimeout")

	groupCmd.Flags().Bool("list", false,
		"Prints list all groups to which this client belongs.")
	err = viper.BindPFlag("list", groupCmd.Flags().Lookup("list"))
	checkBindErr(err, "list")

	groupCmd.Flags().String("show", "",
		"Prints the members of this group ID.")
	err = viper.BindPFlag("show", groupCmd.Flags().Lookup("show"))
	checkBindErr(err, "show")

	rootCmd.AddCommand(groupCmd)
}

func checkBindErr(err error, key string) {
	if err != nil {
		jww.ERROR.Printf("viper.BindPFlag failed for %s: %+v", key, err)
	}
}
