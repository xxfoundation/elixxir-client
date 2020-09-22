package message

import (
	"gitlab.com/elixxir/client/context/message"
	"time"
)

func (m *Manager) processGarbledMessages(quitCh <-chan struct{}) {
	done := false
	for !done {
		select {
		case <-quitCh:
			done = true
		case <-m.triggerGarbled:
			m.handleGarbledMessages()
		}
	}
}

func (m *Manager) handleGarbledMessages() {
	garbledMsgs := m.Session.GetGarbledMessages()
	e2eKv := m.Session.E2e()
	//try to decrypt every garbled message, excising those who's counts are too high
	for grbldMsg, count, timestamp, has := garbledMsgs.Next(); has;
	grbldMsg, count, timestamp, has = garbledMsgs.Next() {
		fingerprint := grbldMsg.GetKeyFP()
		// Check if the key is there, process it if it is
		if key, isE2E := e2eKv.PopKey(fingerprint); isE2E {
			// Decrypt encrypted message
			msg, err := key.Decrypt(grbldMsg)
			// get the sender
			sender := key.GetSession().GetPartner()
			if err == nil {
				//remove from the buffer if decryption is successful
				garbledMsgs.Remove(grbldMsg)
				//handle the successfully decrypted message
				xxMsg, ok := m.partitioner.HandlePartition(sender, message.E2E, msg.GetContents())
				if ok {
					m.Switchboard.Speak(xxMsg)
					continue
				}
			}
		}
		// fail the message if any part of the decryption fails,
		// unless it is our of attempts and has been in the buffer long enough,
		// then remove it
		if count == m.param.MaxChecksGarbledMessage &&
			time.Since(timestamp) > m.param.GarbledMessageWait {
			garbledMsgs.Remove(grbldMsg)
		} else {
			garbledMsgs.Failed(grbldMsg)
		}
	}
}
