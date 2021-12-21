///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package message

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/keyExchange"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
	"time"
)

func (m *Manager) SendE2E(msg message.Send, param params.E2E,
	stop *stoppable.Single) ([]id.Round, e2e.MessageID, time.Time, error) {
	if msg.MessageType == message.Raw {
		return nil, e2e.MessageID{}, time.Time{}, errors.Errorf("Raw (%d) is a reserved "+
			"message type", msg.MessageType)
	}
	//timestamp the message
	ts := netTime.Now()

	//partition the message
	partitions, internalMsgId, err := m.partitioner.Partition(msg.Recipient, msg.MessageType, ts,
		msg.Payload)
	if err != nil {
		return nil, e2e.MessageID{}, time.Time{}, errors.WithMessage(err, "failed to send unsafe message")
	}

	//encrypt then send the partitions over cmix
	roundIds := make([]id.Round, len(partitions))
	errCh := make(chan error, len(partitions))

	// get the key manager for the partner
	partner, err := m.Session.E2e().GetPartner(msg.Recipient)
	if err != nil {
		return nil, e2e.MessageID{}, time.Time{}, errors.WithMessagef(err,
			"Could not send End to End encrypted "+
				"message, no relationship found with %s", msg.Recipient)
	}

	wg := sync.WaitGroup{}

	jww.INFO.Printf("E2E sending %d messages to %s",
		len(partitions), msg.Recipient)

	for i, p := range partitions {

		if msg.MessageType != message.KeyExchangeTrigger {
			// check if any rekeys need to happen and trigger them
			keyExchange.CheckKeyExchanges(m.Instance, m.SendE2E,
				m.Session, partner, 1*time.Minute, stop)
		}

		//create the cmix message
		msgLen := m.Session.Cmix().GetGroup().GetP().ByteLen()
		msgCmix := format.NewMessage(msgLen)
		msgCmix.SetContents(p)

		//get a key to end to end encrypt
		key, err := partner.GetKeyForSending(param.Type)
		keyTries := 0
		for err != nil && keyTries < param.RetryCount {
			jww.WARN.Printf("Out of sending keys for %s "+
				"(msgDigest: %s, partition: %d), this can "+
				"happen when sending messages faster than "+
				"the client can negotiate keys. Please "+
				"adjust your e2e key parameters",
				msg.Recipient, msgCmix.Digest(), i)
			keyTries++
			time.Sleep(param.RetryDelay)
			key, err = partner.GetKeyForSending(param.Type)
		}
		if err != nil {
			return nil, e2e.MessageID{}, time.Time{}, errors.WithMessagef(err,
				"Failed to get key for end to end encryption")
		}

		//end to end encrypt the cmix message
		msgEnc := key.Encrypt(msgCmix)

		jww.INFO.Printf("E2E sending %d/%d to %s with msgDigest: %s, key fp: %s",
			i+i, len(partitions), msg.Recipient, msgEnc.Digest(), key.Fingerprint())

		localParam := param

		// set all non last partitions to the silent type so they
		// dont cause notificatons if OnlyNotifyOnLastSend is true
		lastPartition:= len(partitions)-1
		if localParam.OnlyNotifyOnLastSend && i != lastPartition {
			localParam.IdentityPreimage = partner.GetSilentPreimage()
		}else if localParam.IdentityPreimage == nil {
			//set the preimage to the default e2e one if it is not already set
			localParam.IdentityPreimage = partner.GetE2EPreimage()
		}

		//send the cmix message, each partition in its own thread
		wg.Add(1)
		go func(i int) {
			var err error
			roundIds[i], _, err = m.SendCMIX(m.sender, msgEnc, msg.Recipient,
				localParam.CMIX, stop)
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
		jww.INFO.Printf("Failed to E2E send %d/%d to %s",
			numFail, len(partitions), msg.Recipient)
		return nil, e2e.MessageID{}, time.Time{}, errors.Errorf("Failed to E2E send %v/%v sub payloads:"+
			" %s", numFail, len(partitions), errRtn)
	} else {
		jww.INFO.Printf("Successfully E2E sent %d/%d to %s",
			len(partitions)-numFail, len(partitions), msg.Recipient)
	}

	//return the rounds if everything send successfully
	msgID := e2e.NewMessageID(partner.GetSendRelationshipFingerprint(), internalMsgId)
	return roundIds, msgID, ts, nil
}
