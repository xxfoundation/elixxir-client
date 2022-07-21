package cmd

import (
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	cmdUtils "gitlab.com/elixxir/client/cmdUtils"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/single"
	"time"
)

// Start is the ingress point for this package. This will handle CLI input and operations
// for the single subcommand.
func Start() {
	// Initialize log
	cmdUtils.InitLog(viper.GetUint(cmdUtils.LogLevelFlag), viper.GetString(cmdUtils.LogFlag))

	// Initialize messenger
	cmixParams, e2eParams := cmdUtils.InitParams()
	authCbs := cmdUtils.MakeAuthCallbacks(
		viper.GetBool(cmdUtils.UnsafeChannelCreationFlag), e2eParams)
	messenger := cmdUtils.InitE2e(cmixParams, e2eParams, authCbs)

	// Write user contact to file
	user := messenger.GetReceptionIdentity()
	jww.INFO.Printf("User: %s", user.ID)
	cmdUtils.WriteContact(user.GetContact())

	err := messenger.StartNetworkFollower(5 * time.Second)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	// Wait until connected or crash on timeout
	connected := make(chan bool, 10)
	messenger.GetCmix().AddHealthCallback(
		func(isconnected bool) {
			connected <- isconnected
		})
	cmdUtils.WaitUntilConnected(connected)

	// get the tag
	tag := viper.GetString(SingleTagFlag)

	// Register the callback
	receiver := &receiver{
		recvCh: make(chan struct {
			request *single.Request
			ephID   receptionID.EphemeralIdentity
			round   []rounds.Round
		}),
	}

	dhKeyPriv, err := user.GetDHKeyPrivate()
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	myID := user.ID
	listener := single.Listen(tag, myID,
		dhKeyPriv,
		messenger.GetCmix(),
		messenger.GetStorage().GetE2EGroup(),
		receiver)

	for numReg, total := 1, 100; numReg < (total*3)/4; {
		time.Sleep(1 * time.Second)
		numReg, total, err = messenger.GetNodeRegistrationStatus()
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
		jww.INFO.Printf("Registering with nodes (%d/%d)...",
			numReg, total)
	}

	timeout := viper.GetDuration(SingleTimeoutFlag)

	// If the send flag is set, then send a message
	if viper.GetBool(SingleSendFlag) {
		// get message details
		payload := []byte(viper.GetString(cmdUtils.MessageFlag))
		partner := readSingleUseContact(SingleContactFlag)
		maxMessages := uint8(viper.GetUint(SingleMaxMessagesFlag))

		sendSingleUse(messenger.Cmix, partner, payload,
			maxMessages, timeout, tag)
	}

	// If the reply flag is set, then start waiting for a
	// message and reply when it is received
	if viper.GetBool(SingleReplyFlag) {
		replySingleUse(timeout, receiver)
	}
	listener.Stop()

}
