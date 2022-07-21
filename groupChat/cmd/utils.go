package cmd

import (
	"bufio"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	cmdUtils "gitlab.com/elixxir/client/cmdUtils"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/groupChat"
	"gitlab.com/elixxir/client/groupChat/groupStore"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"os"
	"time"
)

// initGroupManager creates a new group chat manager and starts the process
// service.
func initGroupManager(messenger *xxdk.E2e) (groupChat.GroupChat,
	chan groupChat.MessageReceive, chan groupStore.Group) {
	recChan := make(chan groupChat.MessageReceive, 10)

	reqChan := make(chan groupStore.Group, 10)
	requestCb := func(g groupStore.Group) {
		reqChan <- g
	}

	jww.INFO.Print("[GC] Creating new group manager.")
	manager, err := groupChat.NewManager(messenger, requestCb,
		&receiveProcessor{recChan})
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
	userIdStrings := readLines(filePath)
	userIDs := make([]*id.ID, 0, len(userIdStrings))
	for _, userIdStr := range userIdStrings {
		userID := cmdUtils.ParseRecipient(userIdStr)
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
	groupID := cmdUtils.ParseRecipient(groupIdString)
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
	groupID := cmdUtils.ParseRecipient(groupIdString)
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
	groupID := cmdUtils.ParseRecipient(groupIdString)

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
	groupID := cmdUtils.ParseRecipient(groupIdString)

	grp, ok := gm.GetGroup(groupID)
	if !ok {
		jww.FATAL.Printf("[GC] Could not find group: %s", groupID)
	}

	jww.INFO.Printf("[GC] Show group %#v", grp)
	fmt.Printf("Got group with name %q and message %q\n",
		grp.Name, grp.InitMessage)
}

// readLines returns each line in a file as a string.
func readLines(fileName string) []string {
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
