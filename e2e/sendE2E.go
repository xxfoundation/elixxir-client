package e2e

import (
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
	"sync"
	"time"
)

func (m *manager) SendE2E(mt catalog.MessageType, recipient *id.ID,
	payload []byte, params Params) ([]id.Round, e2e.MessageID, time.Time, error) {

	//check if the network is healthy
	if !m.net.IsHealthy() {
		return nil, e2e.MessageID{}, time.Time{}, errors.New("cannot " +
			"sendE2E when network is not healthy")
	}

	//handle critical messages
	handleCritical := params.CMIX.Critical
	if handleCritical {
		m.crit.AddProcessing(mt, recipient, payload, params)
		// set critical to false so the network layer doesnt make the messages
		// critical as well
		params.CMIX.Critical = false
	}

	rnds, msgID, t, err := m.sendE2E(mt, recipient, payload, params)

	if handleCritical {
		m.crit.handle(mt, recipient, payload, rnds, err)
	}
	return rnds, msgID, t, err

}

func (m *manager) sendE2E(mt catalog.MessageType, recipient *id.ID,
	payload []byte, params Params) ([]id.Round, e2e.MessageID, time.Time, error) {
	//timestamp the message
	ts := netTime.Now()

	//partition the message
	partitions, internalMsgId, err := m.partitioner.Partition(recipient, mt, ts,
		payload)
	if err != nil {
		return nil, e2e.MessageID{}, time.Time{}, errors.WithMessage(err, "failed to send unsafe message")
	}

	jww.INFO.Printf("E2E sending %d messages to %s",
		len(partitions), recipient)

	//encrypt then send the partitions over cmix
	roundIds := make([]id.Round, len(partitions))
	errCh := make(chan error, len(partitions))

	// get the key manager for the partner
	partner, err := m.Ratchet.GetPartner(recipient, m.myDefaultID)
	if err != nil {
		return nil, e2e.MessageID{}, time.Time{}, errors.WithMessagef(err,
			"Could not send End to End encrypted "+
				"message, no relationship found with %s", recipient)
	}

	//Generate the message ID
	msgID := e2e.NewMessageID(partner.GetSendRelationshipFingerprint(),
		internalMsgId)

	wg := sync.WaitGroup{}

	//handle sending for each partition
	for i, p := range partitions {
		if mt != catalog.KeyExchangeTrigger {
			// check if any rekeys need to happen and trigger them
			rekeySendFunc := func(mt catalog.MessageType, recipient *id.ID, payload []byte,
				cmixParams cmix.CMIXParams) (
				[]id.Round, e2e.MessageID, time.Time, error) {
				par := GetDefaultParams()
				par.CMIX = cmixParams
				return m.SendE2E(mt, recipient, payload, par)
			}
			rekey.CheckKeyExchanges(m.net.GetInstance(), m.grp, rekeySendFunc,
				m.events, partner, m.rekeyParams, 1*time.Minute)
		}

		//get a key to end to end encrypt
		var keyGetter func() (*session.Cypher, error)
		if params.Rekey {
			keyGetter = partner.PopRekeyCypher
		} else {
			keyGetter = partner.PopSendCypher
		}

		//fixme: remove this wait, it is weird. Why is it here? we cant remember.
		key, err := waitForKey(keyGetter, params.KeyGetRetryCount,
			params.KeyGeRetryDelay, params.CMIX.Stop, recipient,
			format.DigestContents(p), i)
		if err != nil {
			return nil, e2e.MessageID{}, time.Time{}, errors.WithMessagef(err,
				"Failed to get key for end to end encryption")
		}

		//end to end encrypt the cmix message
		contentsEnc, mac := key.Encrypt(p)

		jww.INFO.Printf("E2E sending %d/%d to %s with key fp: %s, msgID: %s (msgDigest %s)",
			i+i, len(partitions), recipient,
			key.Fingerprint(), msgID, format.DigestContents(p))

		//set up the service tags
		var s message.Service
		if i == len(partitions)-1 {
			s = partner.MakeService(params.LastServiceTag)
		} else {
			s = partner.MakeService(params.ServiceTag)
		}

		//send the cmix message, each partition in its own thread
		wg.Add(1)
		go func(i int) {
			var err error
			roundIds[i], _, err = m.net.SendCMIX(recipient, key.Fingerprint(),
				s, contentsEnc, mac, params.CMIX)
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
			numFail, len(partitions), recipient)
		return nil, e2e.MessageID{}, time.Time{}, errors.Errorf("Failed to E2E send %v/%v sub payloads:"+
			" %s", numFail, len(partitions), errRtn)
	} else {
		jww.INFO.Printf("Successfully E2E sent %d/%d to %s",
			len(partitions)-numFail, len(partitions), recipient)
	}

	//return the rounds if everything send successfully
	jww.INFO.Printf("Successful E2E Send of %d messages to %s with msgID %s",
		len(partitions), recipient, msgID)
	return roundIds, msgID, ts, nil
}

// waitForKey waits the designated ammount of time for a
func waitForKey(keyGetter func() (*session.Cypher, error), numAttempts uint,
	wait time.Duration, stop *stoppable.Single, recipient *id.ID,
	digest string, partition int) (*session.Cypher, error) {
	key, err := keyGetter()
	if err == nil {
		return key, nil
	}

	ticker := time.NewTicker(wait)
	defer ticker.Stop()

	for keyTries := uint(1); err != nil && keyTries < numAttempts; keyTries++ {
		jww.WARN.Printf("Out of sending keys for %s "+
			"(digest: %s, partition: %d), this can "+
			"happen when sending messages faster than "+
			"the client can negotiate keys. Please "+
			"adjust your e2e key parameters",
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

// getSendErrors returns any errors on the error channel
func getSendErrors(c chan error) (int, string) {
	var errRtn string
	numFail := 0
	done := false
	for !done {
		select {
		case err := <-c:
			errRtn += err.Error()
			numFail++
		default:
			done = true
		}
	}
	return numFail, errRtn
}
