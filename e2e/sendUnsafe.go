package e2e

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/e2e/ratchet"
	"gitlab.com/elixxir/client/network/message"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
	"time"
)

func (m *manager) SendUnsafe(mt catalog.MessageType, recipient *id.ID,
	payload []byte, params Params) ([]id.Round, time.Time, error) {

	//check if the network is healthy
	if !m.net.IsHealthy() {
		return nil, time.Time{}, errors.New("cannot " +
			"sendE2E when network is not healthy")
	}

	return m.sendUnsafe(mt, recipient, payload, params)
}

func (m *manager) sendUnsafe(mt catalog.MessageType, recipient *id.ID,
	payload []byte, params Params) ([]id.Round, time.Time, error) {
	//timestamp the message
	ts := netTime.Now()

	//partition the message
	partitions, _, err := m.partitioner.Partition(recipient, mt, ts,
		payload)
	if err != nil {
		return nil, time.Time{}, errors.WithMessage(err, "failed to send unsafe message")
	}

	jww.WARN.Printf("unsafe sending %d messages to %s. Unsafe sends "+
		"are unencrypted, only use for debugging",
		len(partitions), recipient)

	//encrypt then send the partitions over cmix
	roundIds := make([]id.Round, len(partitions))
	errCh := make(chan error, len(partitions))

	wg := sync.WaitGroup{}

	//handle sending for each partition
	for i, p := range partitions {

		//set up the service tags
		srvc := message.Service{
			Identifier: recipient[:],
		}
		if i == len(partitions)-1 {
			srvc.Tag = ratchet.Silent
		} else {
			srvc.Tag = ratchet.E2e
		}

		//send the cmix message, each partition in its own thread
		wg.Add(1)
		go func(i int, payload []byte) {

			unencryptedMAC, fp := e2e.SetUnencrypted(payload, m.myID)

			var err error
			roundIds[i], _, err = m.net.SendCMIX(recipient, fp,
				srvc, payload, unencryptedMAC, params.CMIX)
			if err != nil {
				errCh <- err
			}
			wg.Done()
		}(i, p)
	}

	wg.Wait()

	//see if any parts failed to send
	numFail, errRtn := getSendErrors(errCh)
	if numFail > 0 {
		jww.INFO.Printf("Failed to unsafe send %d/%d to %s",
			numFail, len(partitions), recipient)
		return nil, time.Time{}, errors.Errorf("Failed to unsafe send %v/%v sub payloads:"+
			" %s", numFail, len(partitions), errRtn)
	} else {
		jww.INFO.Printf("Successfully Unsafe Send %d/%d to %s",
			len(partitions)-numFail, len(partitions), recipient)
	}

	//return the rounds if everything send successfully
	jww.INFO.Printf("Successful Unsafe Send of %d messages to %s",
		len(partitions), recipient)
	return roundIds, ts, nil
}
