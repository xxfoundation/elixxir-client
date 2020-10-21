////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package message

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/keyExchange"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"time"
)

func (m *Manager) SendE2E(msg message.Send, param params.E2E) ([]id.Round, e2e.MessageID, error) {
	if msg.MessageType == message.Raw {
		return nil, e2e.MessageID{}, errors.Errorf("Raw (%d) is a reserved "+
			"message type", msg.MessageType)
	}
	//timestamp the message
	ts := time.Now()

	//partition the message
	partitions, internalMsgId, err := m.partitioner.Partition(msg.Recipient, msg.MessageType, ts,
		msg.Payload)
	if err != nil {
		return nil, e2e.MessageID{}, errors.WithMessage(err, "failed to send unsafe message")
	}

	//encrypt then send the partitions over cmix
	roundIds := make([]id.Round, len(partitions))
	errCh := make(chan error, len(partitions))

	// get the key manager for the partner
	partner, err := m.Session.E2e().GetPartner(msg.Recipient)
	if err != nil {
		return nil, e2e.MessageID{}, errors.WithMessagef(err, "Could not send End to End encrypted "+
			"message, no relationship found with %v", partner)
	}

	wg := sync.WaitGroup{}

	for i, p := range partitions {
		//create the cmix message
		msgCmix := format.NewMessage(m.Session.Cmix().GetGroup().GetP().ByteLen())
		msgCmix.SetContents(p)
		msgCmix.SetRecipientID(msg.Recipient)

		//get a key to end to end encrypt
		key, err := partner.GetKeyForSending(param.Type)
		if err != nil {
			return nil, e2e.MessageID{}, errors.WithMessagef(err, "Failed to get key "+
				"for end to end encryption")
		}

		//end to end encrypt the cmix message
		msgEnc := key.Encrypt(msgCmix)

		//send the cmix message, each partition in its own thread
		wg.Add(1)
		go func(i int) {
			var err error
			roundIds[i], err = m.SendCMIX(msgEnc, param.CMIX)
			if err != nil {
				errCh <- err
			}
			wg.Done()
		}(i)
	}

	// while waiting check if any rekeys need to happen and trigger them. This
	// can happen now because the key popping happens in this thread,
	// only the sending is parallelized
	keyExchange.CheckKeyExchanges(m.Instance, m.SendE2E, m.Session, partner, 1*time.Minute)

	wg.Wait()

	//see if any parts failed to send
	numFail, errRtn := getSendErrors(errCh)
	if numFail > 0 {
		return nil, e2e.MessageID{}, errors.Errorf("Failed to E2E send %v/%v sub payloads:"+
			" %s", numFail, len(partitions), errRtn)
	}

	//return the rounds if everything send successfully
	msgID := e2e.NewMessageID(partner.GetSendRelationshipFingerprint(), internalMsgId)
	return roundIds, msgID, nil
}
