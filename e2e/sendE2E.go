////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package e2e

import (
	"sync"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/catalog"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/e2e/parse"
	"gitlab.com/elixxir/client/v4/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/v4/e2e/rekey"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

// SendE2E send a message containing the payload to the
// recipient of the passed message type, per the given
// parameters - encrypted with end-to-end encryption.
// Default parameters can be retrieved through
// GetDefaultParams()
// If too long, it will chunk a message up into its messages
// and send each as a separate cmix message. It will return
// the list of all rounds sent on, a unique ID for the
// message, and the timestamp sent on.
// the recipient must already have an e2e relationship,
// otherwise an error will be returned.
// Will return an error if the network is not healthy or in
// the event of a failed send
func (m *manager) SendE2E(mt catalog.MessageType, recipient *id.ID,
	payload []byte, params Params) (e2e.SendReport, error) {

	if !m.net.IsHealthy() {
		return e2e.SendReport{},
			errors.New("cannot sendE2E when network is not healthy")
	}

	handleCritical := params.Critical
	if handleCritical {
		m.crit.AddProcessing(mt, recipient, payload, params)
		// Set critical to false so that the network layer does not make the
		// messages critical as well
		params.Critical = false
	}

	sendReport, err := m.sendE2E(mt, recipient, payload, params)

	if handleCritical {
		m.crit.handle(mt, recipient, payload, sendReport.RoundList, err)
	}
	return sendReport, err

}

// sendE2eFn contains a prepared sendE2E operation and sends an E2E message when
// called, returning the results of the send.
type sendE2eFn func() (e2e.SendReport, error)

// prepareSendE2E makes a prepared function that does the e2e send.
// This is so that when doing deletePartner we can prepare the send before
// deleting and then send after deleting to ensure there is correctness.
//
// Note: the timestamp in the send is recorded in this call, not when the
// sendE2e function is called.
func (m *manager) prepareSendE2E(mt catalog.MessageType, recipient *id.ID,
	payload []byte, params Params) (sendE2E sendE2eFn, err error) {
	ts := netTime.Now()

	sendFuncs := make([]func(), 0)

	partitions, internalMsgId, err := m.partitioner.Partition(recipient,
		mt, ts, payload)
	if err != nil {
		return nil, errors.WithMessage(err,
			"failed to send unsafe message")
	}

	jww.INFO.Printf("E2E sending %d messages to %s",
		len(partitions), recipient)

	// When sending E2E messages, we first partition into cMix
	// packets and then send each partition over cMix
	roundIds := make([]id.Round, len(partitions))
	errCh := make(chan error, len(partitions))

	// The Key manager for the partner (recipient) ensures single
	// use of each key negotiated for the ratchet
	partner, err := m.Ratchet.GetPartner(recipient)
	if err != nil {
		return nil, errors.WithMessagef(err,
			"cannot send E2E message no relationship found with %s",
			recipient)
	}

	msgID := e2e.NewMessageID(
		partner.SendRelationshipFingerprint(), internalMsgId)

	wg := sync.WaitGroup{}
	var keyResidue e2e.KeyResidue
	for i, p := range partitions {
		if mt != catalog.KeyExchangeTrigger {
			// Check if any rekeys need to happen and trigger them
			rekeySendFunc := func(mt catalog.MessageType,
				recipient *id.ID,
				payload []byte, cmixParams cmix.CMIXParams) (
				e2e.SendReport, error) {
				par := params
				par.CMIXParams = cmixParams
				return m.SendE2E(mt, recipient, payload, par)
			}
			rekey.CheckKeyExchanges(m.net.GetInstance(), m.grp,
				rekeySendFunc, m.events, partner,
				m.rekeyParams, 1*time.Minute)
		}

		var keyGetter func() (session.Cypher, error)
		if params.Rekey {
			keyGetter = partner.PopRekeyCypher
		} else {
			keyGetter = partner.PopSendCypher
		}

		// FIXME: remove this wait, it is weird. Why is it
		// here? we cant remember.
		key, err := waitForKey(
			keyGetter, params.KeyGetRetryCount,
			params.KeyGeRetryDelay,
			params.Stop, recipient, format.DigestContents(p), i)
		if err != nil {
			return nil, errors.WithMessagef(err,
				"Failed to get key for end-to-end encryption")
		}

		// This does not encrypt for cMix but instead
		// end-to-end encrypts the cMix message
		contentsEnc, mac, residue := key.Encrypt(p)
		// Carry the first key residue to the top level
		if i == 0 {
			keyResidue = residue
		}

		jww.INFO.Printf("E2E sending %d/%d to %s with key fp: %s, "+
			"msgID: %s (msgDigest %s)",
			i+i, len(partitions), recipient, key.Fingerprint(),
			msgID, format.DigestContents(p))

		var s message.Service
		if i == len(partitions)-1 {
			s = partner.MakeService(params.LastServiceTag)
		} else {
			s = partner.MakeService(params.ServiceTag)
		}

		// We send each partition in its own thread here; some
		// may send in round X, others in X+1 or X+2, and so
		// on
		localI := i
		thisSendFunc := func() {
			wg.Add(1)
			go func(i int) {
				r, _, err := m.net.Send(recipient,
					key.Fingerprint(), s, contentsEnc, mac,
					params.CMIXParams)
				if err != nil {
					jww.DEBUG.Printf("[E2E] cMix error on "+
						"Send: %+v", err)
					errCh <- err
				}
				roundIds[i] = r.ID
				wg.Done()
			}(localI)
		}
		sendFuncs = append(sendFuncs, thisSendFunc)
	}

	sendE2E = func() (e2e.SendReport, error) {
		for i := range sendFuncs {
			sendFuncs[i]()
		}

		wg.Wait()

		numFail, errRtn := getSendErrors(errCh)
		if numFail > 0 {
			jww.INFO.Printf("Failed to E2E send %d/%d to %s",
				numFail, len(partitions), recipient)
			return e2e.SendReport{}, errors.Errorf(
				"Failed to E2E send %v/%v sub payloads: %s",
				numFail, len(partitions), errRtn)
		} else {
			jww.INFO.Printf("Successfully E2E sent %d/%d to %s",
				len(partitions)-numFail, len(partitions),
				recipient)
		}

		jww.INFO.Printf("Successful E2E Send of %d messages to %s "+
			"with msgID %s", len(partitions), recipient, msgID)

		return e2e.SendReport{
			RoundList:  roundIds,
			MessageId:  msgID,
			SentTime:   ts,
			KeyResidue: keyResidue,
		}, nil
	}
	return sendE2E, nil
}

func (m *manager) sendE2E(mt catalog.MessageType, recipient *id.ID,
	payload []byte, params Params) (e2e.SendReport, error) {
	sendFunc, err := m.prepareSendE2E(mt, recipient, payload, params)
	if err != nil {
		return e2e.SendReport{}, err
	}
	return sendFunc()
}

// waitForKey waits the designated amount of time for a key to become available
// with the partner.
func waitForKey(keyGetter func() (session.Cypher, error), numAttempts uint,
	wait time.Duration, stop *stoppable.Single, recipient *id.ID, digest string,
	partition int) (session.Cypher, error) {
	key, err := keyGetter()
	if err == nil {
		return key, nil
	}

	ticker := time.NewTicker(wait)
	defer ticker.Stop()

	for keyTries := uint(1); err != nil && keyTries < numAttempts; keyTries++ {
		jww.WARN.Printf(
			"Out of sending keys for %s (digest: %s, partition: %d), this can "+
				"happen when sending messages faster than the client can "+
				"negotiate keys. Please adjust your e2e key parameters.",
			recipient, digest, partition)

		select {
		case <-ticker.C:
			key, err = keyGetter()
		case <-stop.Quit():
			stop.ToStopped()
			return nil, errors.Errorf("Stopped by stopper")
		}
	}

	return key, err
}

// getSendErrors returns a string of all error received on the error channel and
// a count of the number of errors.
func getSendErrors(c chan error) (numFail int, errRtn string) {
	for {
		select {
		case err := <-c:
			errRtn += err.Error()
			numFail++
		default:
			return numFail, errRtn
		}
	}
}

// FirstPartitionSize returns the max partition payload size for the
// first payload
func (m *manager) FirstPartitionSize() uint {
	return m.partitioner.FirstPartitionSize()
}

// SecondPartitionSize returns the max partition payload size for all
// payloads after the first payload
func (m *manager) SecondPartitionSize() uint {
	return m.partitioner.SecondPartitionSize()
}

// PartitionSize returns the partition payload size for the given
// payload index. The first payload is index 0.
func (m *manager) PartitionSize(payloadIndex uint) uint {
	if payloadIndex == 0 {
		return m.FirstPartitionSize()
	}
	if payloadIndex > parse.MaxMessageParts {
		return 0
	}
	return m.SecondPartitionSize()
}

// PayloadSize Returns the max payload size for a partitionable E2E
// message
func (m *manager) PayloadSize() uint {
	return m.partitioner.PayloadSize()
}
