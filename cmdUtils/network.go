package cmd

import (
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

func WaitUntilConnected(connected chan bool) {
	waitTimeout := time.Duration(viper.GetUint(waitTimeoutFlag))
	timeoutTimer := time.NewTimer(waitTimeout * time.Second)
	isConnected := false
	// Wait until we connect or panic if we can't by a timeout
	for !isConnected {
		select {
		case isConnected = <-connected:
			jww.INFO.Printf("Network Status: %v\n",
				isConnected)
			break
		case <-timeoutTimer.C:
			jww.FATAL.Panicf("timeout on connection after %s", waitTimeout*time.Second)
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

func VerifySendSuccess(client *xxdk.E2e, paramsE2E e2e.Params,
	roundIDs []id.Round, partnerId *id.ID, payload []byte) bool {
	retryChan := make(chan struct{})
	done := make(chan struct{}, 1)

	// Construct the callback function which
	// verifies successful message send or retries
	f := func(allRoundsSucceeded, timedOut bool,
		rounds map[id.Round]cmix.RoundResult) {
		printRoundResults(
			rounds, roundIDs, payload, partnerId)
		if !allRoundsSucceeded {
			retryChan <- struct{}{}
		} else {
			done <- struct{}{}
		}
	}

	// Monitor rounds for results
	err := client.GetCmix().GetRoundResults(
		paramsE2E.CMIXParams.Timeout, f, roundIDs...)
	if err != nil {
		jww.DEBUG.Printf("Could not verify messages were sent " +
			"successfully, resending messages...")
		return false
	}

	select {
	case <-retryChan:
		// On a retry, go to the top of the loop
		jww.DEBUG.Printf("Messages were not sent successfully," +
			" resending messages...")
		return false
	case <-done:
		// Close channels on verification success
		close(done)
		close(retryChan)
		return true
	}
}

func acceptChannelVerified(messenger *xxdk.E2e, recipientID *id.ID,
	params xxdk.E2EParams) {
	for {
		rid := acceptChannel(messenger, recipientID)
		VerifySendSuccess(messenger, params.Base, []id.Round{rid}, recipientID, nil)
	}
}

func acceptChannel(messenger *xxdk.E2e, recipientID *id.ID) id.Round {
	recipientContact, err := messenger.GetAuth().GetReceivedRequest(
		recipientID)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	rid, err := messenger.GetAuth().Confirm(
		recipientContact)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	return rid
}
