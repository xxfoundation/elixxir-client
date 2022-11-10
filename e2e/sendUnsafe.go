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
	"gitlab.com/elixxir/client/v5/catalog"
	"gitlab.com/elixxir/client/v5/cmix/message"
	"gitlab.com/elixxir/client/v5/e2e/ratchet"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

func (m *manager) SendUnsafe(mt catalog.MessageType, recipient *id.ID,
	payload []byte, params Params) ([]id.Round, time.Time, error) {

	if !m.net.IsHealthy() {
		return nil, time.Time{}, errors.New("cannot " +
			"sendE2E when network is not healthy")
	}

	return m.sendUnsafe(mt, recipient, payload, params)
}

func (m *manager) sendUnsafe(mt catalog.MessageType, recipient *id.ID,
	payload []byte, params Params) ([]id.Round, time.Time, error) {
	ts := netTime.Now()

	partitions, _, err := m.partitioner.Partition(recipient, mt, ts,
		payload)
	if err != nil {
		err = errors.WithMessage(err, "failed to send unsafe message")
		return nil, time.Time{}, err
	}

	jww.WARN.Printf("unsafe sending %d messages to %s. Unsafe sends "+
		"are unencrypted, only use for debugging",
		len(partitions), recipient)

	roundIds := make([]id.Round, len(partitions))
	errCh := make(chan error, len(partitions))

	wg := sync.WaitGroup{}

	for i, p := range partitions {

		srvc := message.Service{
			Identifier: recipient[:],
		}
		if i == len(partitions)-1 {
			srvc.Tag = ratchet.Silent
		} else {
			srvc.Tag = ratchet.E2e
		}

		wg.Add(1)
		go func(i int, payload []byte) {

			unencryptedMAC, fp := e2e.SetUnencrypted(payload,
				m.myID)

			jww.TRACE.Printf("sendUnsafe contents: %v, fp: %v, mac: %v",
				payload, fp, unencryptedMAC)

			r, _, err := m.net.Send(recipient, fp,
				srvc, payload, unencryptedMAC,
				params.CMIXParams)
			if err != nil {
				errCh <- err
			}
			roundIds[i] = r.ID
			wg.Done()
		}(i, p)
	}

	wg.Wait()

	//see if any parts failed to send
	numFail, errRtn := getSendErrors(errCh)
	if numFail > 0 {
		jww.INFO.Printf("Failed to unsafe send %d/%d to %s",
			numFail, len(partitions), recipient)
		err = errors.Errorf("Failed to unsafe send %v/%v sub payloads:"+
			" %s", numFail, len(partitions), errRtn)
		return nil, time.Time{}, err
	} else {
		jww.INFO.Printf("Successfully Unsafe Send %d/%d to %s",
			len(partitions)-numFail, len(partitions), recipient)
	}

	//return the rounds if everything send successfully
	jww.INFO.Printf("Successful Unsafe Send of %d messages to %s",
		len(partitions), recipient)
	return roundIds, ts, nil
}
