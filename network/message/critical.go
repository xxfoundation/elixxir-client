package message

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/interfaces/utility"
	ds "gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/primitives/states"
	"time"
)

// Critical Messages are protocol layer communications that must succeed. These
// are added to the persistent critical messages store.  This thread waits for
// network access to move from unhealthy to healthy and the sends all critical
// messages.
// Health is tracked by registering with the Health
// Tracker (/network/Health/Tracker.g0)

//Thread loop for processing critical messages
func (m *Manager) processCriticalMessages(quitCh <-chan struct{}) {
	done := false
	for !done {
		select {
		case <-quitCh:
			done = true
		case isHealthy := <-m.networkIsHealthy:
			if isHealthy {
				m.criticalMessages()
			}
		}
	}
}

// processes all critical messages
func (m *Manager) criticalMessages() {
	critMsgs := m.Session.GetCriticalMessages()
	//try to send every message in the critical messages buffer in paralell
	for msg, param, has := critMsgs.Next(); has; msg, param, has = critMsgs.Next() {
		go func(msg message.Send, param params.E2E) {
			//send the message
			rounds, _, err := m.SendE2E(msg, param)
			//if the message fail to send, notify the buffer so it can be handled
			//in the future and exit
			if err != nil {
				jww.ERROR.Printf("Failed to send critical message on " +
					"notification of healthy network")
				critMsgs.Failed(msg)
				return
			}
			//wait on the results to make sure the rounds were sucesfull
			sendResults := make(chan ds.EventReturn, len(rounds))
			roundEvents := m.Instance.GetRoundEvents()
			for _, r := range rounds {
				roundEvents.AddRoundEventChan(r, sendResults, 1*time.Minute,
					states.COMPLETED, states.FAILED)
			}
			success, numTimeOut, numRoundFail := utility.TrackResults(sendResults, len(rounds))
			if !success {
				jww.ERROR.Printf("critical message send failed to transmit "+
					"transmit %v/%v paritions: %v round failures, %v timeouts",
					numRoundFail+numTimeOut, len(rounds), numRoundFail, numTimeOut)
				critMsgs.Failed(msg)
				return
			}
			critMsgs.Succeeded(msg)
		}(msg, param)
	}
}
