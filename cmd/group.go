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
	"os"
	"time"

	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/primitives/format"

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
		cmixParams, e2eParams := initParams()
		client := initE2e(cmixParams, e2eParams)

		// Print user's reception ID
		user := client.GetReceptionIdentity()
		jww.INFO.Printf("User: %s", user.ID)

		err := client.StartNetworkFollower(5 * time.Second)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		// Initialize the group chat manager
		groupManager, recChan, reqChan := initGroupManager(client)

		// Wait until connected or crash on timeout
		connected := make(chan bool, 10)
		client.GetCmix().AddHealthCallback(
			func(isConnected bool) {
				connected <- isConnected
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

		// Get group message and name
		msgBody := []byte(viper.GetString(messageFlag))
		name := []byte(viper.GetString(groupNameFlag))
		timeout := viper.GetDuration(groupReceiveTimeoutFlag)

		if viper.IsSet(groupCreateFlag) {
			filePath := viper.GetString(groupCreateFlag)
			createGroup(name, msgBody, filePath, groupManager)
		}

		if viper.IsSet(groupResendFlag) {
			groupIdString := viper.GetString(groupResendFlag)
			resendRequests(groupIdString, groupManager)
		}

		if viper.GetBool(groupJoinFlag) {
			joinGroup(reqChan, timeout, groupManager)
		}

		if viper.IsSet(groupLeaveFlag) {
			groupIdString := viper.GetString(groupLeaveFlag)
			leaveGroup(groupIdString, groupManager)
		}

		if viper.IsSet(groupSendMessageFlag) {
			groupIdString := viper.GetString(groupSendMessageFlag)
			sendGroup(groupIdString, msgBody, groupManager)
		}

		if viper.IsSet(groupWaitFlag) {
			numMessages := viper.GetUint(groupWaitFlag)
			messageWait(numMessages, timeout, recChan)
		}

		if viper.GetBool(groupListFlag) {
			listGroups(groupManager)
		}

		if viper.IsSet(groupShowFlag) {
			groupIdString := viper.GetString(groupShowFlag)
			showGroup(groupIdString, groupManager)
		}
	},
}

// initGroupManager creates a new group chat manager and starts the process
// service.
func initGroupManager(client *xxdk.E2e) (groupChat.GroupChat,
	chan groupChat.MessageReceive, chan groupStore.Group) {
	recChan := make(chan groupChat.MessageReceive, 10)

	reqChan := make(chan groupStore.Group, 10)
	requestCb := func(g groupStore.Group) {
		reqChan <- g
	}

	jww.INFO.Print("[GC] Creating new group manager.")
	manager, err := groupChat.NewManager(client.GetCmix(),
		client.GetE2E(), client.GetStorage().GetReceptionID(),
		client.GetRng(), client.GetStorage().GetE2EGroup(),
		client.GetStorage().GetKV(), requestCb, &receiveProcessor{recChan})
	if err != nil {
		jww.FATAL.Panicf("[GC] Failed to initialize group chat manager: %+v", err)
	}

	return manager, recChan, reqChan
}

type receiveProcessor struct {
	recChan chan groupChat.MessageReceive
}

func (r *receiveProcessor) Process(decryptedMsg groupChat.MessageReceive,
	_ format.Message, _ receptionID.EphemeralIdentity, _ rounds.Round) {
	r.recChan <- decryptedMsg
}

func (r *receiveProcessor) String() string {
	return "groupChatReceiveProcessor"
}

// createGroup creates a new group with the provided name and sends out requests
// to the list of user IDs found at the given file path.
func createGroup(name, msg []byte, filePath string, gm groupChat.GroupChat) {
	userIdStrings := ReadLines(filePath)
	userIDs := make([]*id.ID, 0, len(userIdStrings))
	for _, userIdStr := range userIdStrings {
		userID := parseRecipient(userIdStr)
		userIDs = append(userIDs, userID)
	}

	grp, rids, status, err := gm.MakeGroup(userIDs, name, msg)
	if err != nil {
		jww.FATAL.Panicf("[GC] Failed to create new group: %+v", err)
	}

	// Integration grabs the group ID from this line
	jww.INFO.Printf("[GC] NewGroupID: b64:%s", grp.ID)
	jww.INFO.Printf("[GC] Created Group: Requests:%s on rounds %#v, %v",
		status, rids, grp)
	fmt.Printf("Created new group with name %q and message %q\n", grp.Name,
		grp.InitMessage)
}

// resendRequests resends group requests for the group ID.
func resendRequests(groupIdString string, gm groupChat.GroupChat) {
	groupID := parseRecipient(groupIdString)
	rids, status, err := gm.ResendRequest(groupID)
	if err != nil {
		jww.FATAL.Panicf("[GC] Failed to resend requests to group %s: %+v",
			groupID, err)
	}

	jww.INFO.Printf("[GC] Resending requests to group %s: %v, %s",
		groupID, rids, status)
	fmt.Println("Resending group requests to group.")
}

// joinGroup joins a group when a request is received on the group request
// channel.
func joinGroup(reqChan chan groupStore.Group, timeout time.Duration,
	gm groupChat.GroupChat) {
	jww.INFO.Print("[GC] Waiting for group request to be received.")
	fmt.Println("Waiting for group request to be received.")

	select {
	case grp := <-reqChan:
		err := gm.JoinGroup(grp)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		jww.INFO.Printf("[GC] Joined group %s (%+v)", grp.ID, grp)
		fmt.Printf("Joined group with name %q and message %q\n",
			grp.Name, grp.InitMessage)
	case <-time.After(timeout):
		jww.INFO.Printf("[GC] Timed out after %s waiting for group request.", timeout)
		fmt.Println("Timed out waiting for group request.")
		return
	}
}

// leaveGroup leaves the group.
func leaveGroup(groupIdString string, gm groupChat.GroupChat) {
	groupID := parseRecipient(groupIdString)
	jww.INFO.Printf("[GC] Leaving group %s.", groupID)

	err := gm.LeaveGroup(groupID)
	if err != nil {
		jww.FATAL.Panicf("[GC] Failed to leave group %s: %+v", groupID, err)
	}

	jww.INFO.Printf("[GC] Left group: %s", groupID)
	fmt.Println("Left group.")
}

// sendGroup send the message to the group.
func sendGroup(groupIdString string, msg []byte, gm groupChat.GroupChat) {
	groupID := parseRecipient(groupIdString)

	jww.INFO.Printf("[GC] Sending to group %s message %q", groupID, msg)

	rid, timestamp, _, err := gm.Send(groupID, "", msg)
	if err != nil {
		jww.FATAL.Panicf("[GC] Sending message to group %s: %+v", groupID, err)
	}

	jww.INFO.Printf("[GC] Sent to group %s on round %d at %s",
		groupID, rid, timestamp)
	fmt.Printf("Sent message %q to group.\n", msg)
}

// messageWait waits for the given number of messages to be received on the
// groupChat.MessageReceive channel.
func messageWait(numMessages uint, timeout time.Duration,
	recChan chan groupChat.MessageReceive) {
	jww.INFO.Printf("[GC] Waiting for %d group message(s) to be received.", numMessages)
	fmt.Printf("Waiting for %d group message(s) to be received.\n", numMessages)

	for i := uint(0); i < numMessages; {
		select {
		case msg := <-recChan:
			i++
			jww.INFO.Printf("[GC] Received group message %d/%d: %s", i, numMessages, msg)
			fmt.Printf("Received group message: %q\n", msg.Payload)
		case <-time.NewTimer(timeout).C:
			jww.INFO.Printf("[GC] Timed out after %s waiting for group message.", timeout)
			fmt.Printf("Timed out waiting for %d group message(s).\n", numMessages)
			return
		}
	}
}

// listGroups prints a list of all groups.
func listGroups(gm groupChat.GroupChat) {
	for i, gid := range gm.GetGroups() {
		jww.INFO.Printf("[GC] Group %d: %s", i, gid)
	}

	fmt.Printf("Printed list of %d groups.\n", gm.NumGroups())
}

// showGroup prints all the information of the group.
func showGroup(groupIdString string, gm groupChat.GroupChat) {
	groupID := parseRecipient(groupIdString)

	grp, ok := gm.GetGroup(groupID)
	if !ok {
		jww.FATAL.Printf("[GC] Could not find group: %s", groupID)
	}

	jww.INFO.Printf("[GC] Show group %#v", grp)
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
	groupCmd.Flags().String(groupCreateFlag, "",
		"Create a group with from the list of contact file paths.")
	bindFlagHelper(groupCreateFlag, groupCmd)

	groupCmd.Flags().String(groupNameFlag, "Group Name",
		"The name of the new group to create.")
	bindFlagHelper(groupNameFlag, groupCmd)

	groupCmd.Flags().String(groupResendFlag, "",
		"Resend invites for all users in this group ID.")
	bindFlagHelper(groupResendFlag, groupCmd)

	groupCmd.Flags().Bool(groupJoinFlag, false,
		"Waits for group request joins the group.")
	bindFlagHelper(groupJoinFlag, groupCmd)

	groupCmd.Flags().String(groupLeaveFlag, "",
		"Leave this group ID.")
	bindFlagHelper(groupLeaveFlag, groupCmd)

	groupCmd.Flags().String(groupSendMessageFlag, "",
		"Send message to this group ID.")
	bindFlagHelper(groupSendMessageFlag, groupCmd)

	groupCmd.Flags().Uint(groupWaitFlag, 0,
		"Waits for number of messages to be received.")
	bindFlagHelper(groupWaitFlag, groupCmd)

	groupCmd.Flags().Duration(groupReceiveTimeoutFlag, time.Minute,
		"Amount of time to wait for a group request or message before timing out.")
	bindFlagHelper(groupReceiveTimeoutFlag, groupCmd)

	groupCmd.Flags().Bool(groupListFlag, false,
		"Prints list all groups to which this client belongs.")
	bindFlagHelper(groupListFlag, groupCmd)

	groupCmd.Flags().String(groupShowFlag, "",
		"Prints the members of this group ID.")
	bindFlagHelper(groupShowFlag, groupCmd)

	rootCmd.AddCommand(groupCmd)
}
