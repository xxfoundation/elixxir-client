package e2e

import (
	"sync"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/e2e/rekey"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

func (m *manager) SendE2E(mt catalog.MessageType, recipient *id.ID,
	payload []byte, params Params) ([]id.Round, e2e.MessageID, time.Time, error) {

	if !m.net.IsHealthy() {
		return nil, e2e.MessageID{}, time.Time{},
			errors.New("cannot sendE2E when network is not healthy")
	}

	handleCritical := params.Critical
	if handleCritical {
		m.crit.AddProcessing(mt, recipient, payload, params)
		// Set critical to false so that the network layer does not make the
		// messages critical as well
		params.Critical = false
	}

	rounds, msgID, t, err := m.sendE2E(mt, recipient, payload, params)

	if handleCritical {
		m.crit.handle(mt, recipient, payload, rounds, err)
	}
	return rounds, msgID, t, err

}

func (m *manager) sendE2E(mt catalog.MessageType, recipient *id.ID,
	payload []byte, params Params) ([]id.Round, e2e.MessageID, time.Time, error) {
	ts := netTime.Now()

	partitions, internalMsgId, err := m.partitioner.Partition(recipient,
		mt, ts, payload)
	if err != nil {
		return nil, e2e.MessageID{}, time.Time{},
			errors.WithMessage(err, "failed to send unsafe message")
	}

	jww.INFO.Printf("E2E sending %d messages to %s", len(partitions), recipient)

	// When sending E2E messages, we first partition into cMix packets and then
	// send each partition over cMix
	roundIds := make([]id.Round, len(partitions))
	errCh := make(chan error, len(partitions))

	// The Key manager for the partner (recipient) ensures single use of each
	// key negotiated for the ratchet
	partner, err := m.Ratchet.GetPartner(recipient)
	if err != nil {
		return nil, e2e.MessageID{}, time.Time{}, errors.WithMessagef(err,
			"cannot send E2E message no relationship found with %s", recipient)
	}

	msgID := e2e.NewMessageID(
		partner.SendRelationshipFingerprint(), internalMsgId)

	wg := sync.WaitGroup{}

	for i, p := range partitions {
		if mt != catalog.KeyExchangeTrigger {
			// Check if any rekeys need to happen and trigger them
			rekeySendFunc := func(mt catalog.MessageType, recipient *id.ID,
				payload []byte, cmixParams cmix.CMIXParams) (
				[]id.Round, e2e.MessageID, time.Time, error) {
				par := GetDefaultParams()
				par.CMIXParams = cmixParams
				return m.SendE2E(mt, recipient, payload, par)
			}
			rekey.CheckKeyExchanges(m.net.GetInstance(), m.grp, rekeySendFunc,
				m.events, partner, m.rekeyParams, 1*time.Minute)
		}

		var keyGetter func() (session.Cypher, error)
		if params.Rekey {
			keyGetter = partner.PopRekeyCypher
		} else {
			keyGetter = partner.PopSendCypher
		}

		// FIXME: remove this wait, it is weird. Why is it here? we cant remember.
		key, err := waitForKey(
			keyGetter, params.KeyGetRetryCount, params.KeyGeRetryDelay,
			params.Stop, recipient, format.DigestContents(p), i)
		if err != nil {
			return nil, e2e.MessageID{}, time.Time{}, errors.WithMessagef(err,
				"Failed to get key for end-to-end encryption")
		}

		// This does not encrypt for cMix but instead end-to-end encrypts the
		// cMix message
		contentsEnc, mac := key.Encrypt(p)

		jww.INFO.Printf(
			"E2E sending %d/%d to %s with key fp: %s, msgID: %s (msgDigest %s)",
			i+i, len(partitions), recipient, key.Fingerprint(), msgID,
			format.DigestContents(p))

		var s message.Service
		if i == len(partitions)-1 {
			s = partner.MakeService(params.LastServiceTag)
		} else {
			s = partner.MakeService(params.ServiceTag)
		}

		// We send each partition in its own thread here; some may send in round
		// X, others in X+1 or X+2, and so on
		wg.Add(1)
		go func(i int) {
			var err error
			roundIds[i], _, err = m.net.Send(recipient,
				key.Fingerprint(), s, contentsEnc, mac, params.CMIXParams)
			if err != nil {
				errCh <- err
			}
			wg.Done()
		}(i)
	}

	wg.Wait()

	numFail, errRtn := getSendErrors(errCh)
	if numFail > 0 {
		jww.INFO.Printf("Failed to E2E send %d/%d to %s",
			numFail, len(partitions), recipient)
		return nil, e2e.MessageID{}, time.Time{}, errors.Errorf(
			"Failed to E2E send %v/%v sub payloads: %s",
			numFail, len(partitions), errRtn)
	} else {
		jww.INFO.Printf("Successfully E2E sent %d/%d to %s",
			len(partitions)-numFail, len(partitions), recipient)
	}

	jww.INFO.Printf("Successful E2E Send of %d messages to %s with msgID %s",
		len(partitions), recipient, msgID)

	return roundIds, msgID, ts, nil
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
