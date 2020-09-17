////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package network

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/context/message"
	"gitlab.com/elixxir/client/context/params"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"time"
)

// SendE2E sends an end-to-end payload to the provided recipient with
// the provided msgType. Returns the list of rounds in which parts of
// the message were sent or an error if it fails.
func (m *Manager) SendE2E(msg message.Send, e2eP params.E2E) (
	[]id.Round, error) {

	if !m.health.IsRunning() {
		return nil, errors.New("Cannot send e2e message when the " +
			"network is not healthy")
	}

	return m.sendE2E(msg, e2eP)
}

func (m *Manager) sendE2E(msg message.Send, param params.E2E) ([]id.Round, error) {

	//timestamp the message
	ts := time.Now()

	//partition the message
	partitions, err := m.partitioner.Partition(msg.Recipient, msg.MessageType, ts,
		msg.Payload)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to send unsafe message")
	}

	//encrypt then send the partitions over cmix
	roundIds := make([]id.Round, len(partitions))
	errCh := make(chan error, len(partitions))

	// get the key manager for the partner
	partner, err := m.Context.Session.E2e().GetPartner(msg.Recipient)
	if err != nil {
		return nil, errors.WithMessagef(err, "Could not send End to End encrypted "+
			"message, no relationship found with %s", partner)
	}

	wg := sync.WaitGroup{}

	for i, p := range partitions {
		//create the cmix message
		msgCmix := format.NewMessage(m.Context.Session.Cmix().GetGroup().GetP().ByteLen())
		msgCmix.SetContents(p)

		//get a key to end to end encrypt
		key, err := partner.GetKeyForSending(param.Type)
		if err != nil {
			return nil, errors.WithMessagef(err, "Failed to get key "+
				"for end to end encryption")
		}

		//end to end encrypt the cmix message
		msgEnc := key.Encrypt(msgCmix)

		//send the cmix message, each partition in its own thread
		wg.Add(1)
		go func(i int) {
			var err error
			roundIds[i], err = m.sendCMIX(msgEnc, param.CMIX)
			if err != nil {
				errCh <- err
			}
			wg.Done()
		}(i)
	}

	wg.Wait()

	//see if any parts failed to send
	numFail, errRtn := getSendErrors(errCh)
	if numFail > 0 {
		return nil, errors.Errorf("Failed to E2E send %v/%v sub payloads:"+
			" %s", numFail, len(partitions), errRtn)
	}

	//return the rounds if everything send successfully
	return roundIds, nil
}
