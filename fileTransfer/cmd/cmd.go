package cmd

import (
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	cmdUtils "gitlab.com/elixxir/client/cmdUtils"
	"time"
)

// Start is the ingress point for this package. This will handle CLI input and operations
// for the fileTransfer subcommand.
func Start() {
	// Initialize messenger
	cmixParams, e2eParams := cmdUtils.InitParams()
	authCbs := cmdUtils.MakeAuthCallbacks(
		viper.GetBool(cmdUtils.UnsafeChannelCreationFlag), e2eParams)
	cmdUtils.InitLog(viper.GetUint(cmdUtils.LogLevelFlag), viper.GetString(cmdUtils.LogFlag))
	messenger := cmdUtils.InitE2e(cmixParams, e2eParams, authCbs)

	// Print user's reception ID and save contact file
	user := messenger.GetReceptionIdentity()
	jww.INFO.Printf("User: %s", user.ID)
	cmdUtils.WriteContact(user.GetContact())

	// Start the network follower
	err := messenger.StartNetworkFollower(5 * time.Second)
	if err != nil {
		jww.FATAL.Panicf("Failed to start the network follower: %+v", err)
	}

	// Initialize the file transfer manager
	maxThroughput := viper.GetInt(FileMaxThroughputFlag)
	m, receiveChan := initFileTransferManager(messenger, maxThroughput)

	// Wait until connected or crash on timeout
	connected := make(chan bool, 10)
	messenger.GetCmix().AddHealthCallback(
		func(isConnected bool) {
			connected <- isConnected
		})
	cmdUtils.WaitUntilConnected(connected)

	// Start thread that receives new file transfers and prints them to log
	receiveQuit := make(chan struct{})
	receiveDone := make(chan struct{})
	go receiveNewFileTransfers(receiveChan, receiveDone, receiveQuit, m)

	// If set, send the file to the recipient
	sendDone := make(chan struct{})
	if viper.IsSet(FileSendFlag) {
		recipientContactPath := viper.GetString(FileSendFlag)
		filePath := viper.GetString(FilePathFlag)
		fileType := viper.GetString(FileTypeFlag)
		filePreviewPath := viper.GetString(FilePreviewPathFlag)
		filePreviewString := viper.GetString(FilePreviewStringFlag)
		retry := float32(viper.GetFloat64(FileRetry))

		sendFile(filePath, fileType, filePreviewPath, filePreviewString,
			recipientContactPath, retry, m, sendDone)
	}

	// Wait until either the file finishes sending or the file finishes
	// being received, stop the receiving thread, and exit
	select {
	case <-sendDone:
		jww.INFO.Printf("[FT] Finished sending file. Stopping threads " +
			"and network follower.")
	case <-receiveDone:
		jww.INFO.Printf("[FT] Finished receiving file. Stopping threads " +
			"and network follower.")
	}

	// Stop reception thread
	receiveQuit <- struct{}{}

	// Stop network follower
	err = messenger.StopNetworkFollower()
	if err != nil {
		jww.WARN.Printf("[FT] Failed to stop network follower: %+v", err)
	}

	jww.INFO.Print("[FT] File transfer finished stopping threads and " +
		"network follower.")
}
