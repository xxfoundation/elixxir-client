package cmd

import (
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	cmdUtils "gitlab.com/elixxir/client/cmdUtils"
	"time"
)

// Start is the ingress point for this package. This will handle CLI input and operations
// for the group subcommand.
func Start() {
	// Initialize parameters
	cmixParams, e2eParams := cmdUtils.InitParams()

	// Initialize logs
	cmdUtils.InitLog(viper.GetUint(cmdUtils.LogLevelFlag), viper.GetString(cmdUtils.LogFlag))

	// Initialize messenger
	authCbs := cmdUtils.MakeAuthCallbacks(
		viper.GetBool(cmdUtils.UnsafeChannelCreationFlag), e2eParams)
	messenger := cmdUtils.InitE2e(cmixParams, e2eParams, authCbs)

	// Print user's reception ID
	user := messenger.GetReceptionIdentity()
	jww.INFO.Printf("User: %s", user.ID)

	err := messenger.StartNetworkFollower(5 * time.Second)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	// Initialize the group chat manager
	groupManager, recChan, reqChan := initGroupManager(messenger)

	// Wait until connected or crash on timeout
	connected := make(chan bool, 10)
	messenger.GetCmix().AddHealthCallback(
		func(isConnected bool) {
			connected <- isConnected
		})
	cmdUtils.WaitUntilConnected(connected)

	// todo CMDRef: some other cmd paths don't use this, determine if this is necessary across the board
	// After connection, make sure we have registered with at least 85% of
	// the nodes
	for numReg, total := 1, 100; numReg < (total*3)/4; {
		time.Sleep(1 * time.Second)
		numReg, total, err = messenger.GetNodeRegistrationStatus()
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		jww.INFO.Printf("Registering with nodes (%d/%d)...", numReg, total)
	}

	// Get group message and name
	msgBody := []byte(viper.GetString(cmdUtils.MessageFlag))
	name := []byte(viper.GetString(GroupNameFlag))
	timeout := viper.GetDuration(GroupReceiveTimeoutFlag)

	if viper.IsSet(GroupCreateFlag) {
		filePath := viper.GetString(GroupCreateFlag)
		createGroup(name, msgBody, filePath, groupManager)
	}

	if viper.IsSet(GroupResendFlag) {
		groupIdString := viper.GetString(GroupResendFlag)
		resendRequests(groupIdString, groupManager)
	}

	if viper.GetBool(GroupJoinFlag) {
		joinGroup(reqChan, timeout, groupManager)
	}

	if viper.IsSet(GroupLeaveFlag) {
		groupIdString := viper.GetString(GroupLeaveFlag)
		leaveGroup(groupIdString, groupManager)
	}

	if viper.IsSet(GroupSendMessageFlag) {
		groupIdString := viper.GetString(GroupSendMessageFlag)
		sendGroup(groupIdString, msgBody, groupManager)
	}

	if viper.IsSet(GroupWaitFlag) {
		numMessages := viper.GetUint(GroupWaitFlag)
		messageWait(numMessages, timeout, recChan)
	}

	if viper.GetBool(GroupListFlag) {
		listGroups(groupManager)
	}

	if viper.IsSet(GroupShowFlag) {
		groupIdString := viper.GetString(GroupShowFlag)
		showGroup(groupIdString, groupManager)
	}
}
