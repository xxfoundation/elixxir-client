////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package dummy

import (
	"gitlab.com/elixxir/client/cmix/message"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
)

// Error messages.
const (
	numMsgsRngErr     = "failed to generate random number of messages to send: %+v"
	payloadRngErr     = "failed to generate random payload: %+v"
	recipientRngErr   = "failed to generate random recipient: %+v"
	fingerprintRngErr = "failed to generate random fingerprint: %+v"
	macRngErr         = "failed to generate random MAC: %+v"
)

// sendThread is a thread that sends the dummy messages at random intervals.
func (m *Manager) sendThread(stop *stoppable.Single) {
	jww.INFO.Print("Starting dummy traffic sending thread.")

	nextSendChan := make(<-chan time.Time)
	nextSendChanPtr := &(nextSendChan)

	for {
		select {
		case <-stop.Quit():
			m.stopSendThread(stop)
			return
		case status := <-m.statusChan:
			if status {
				atomic.StoreUint32(&m.status, running)
				nextSendChanPtr = &(m.randomTimer().C)
			} else {
				atomic.StoreUint32(&m.status, paused)
				nextSendChan = make(<-chan time.Time)
				nextSendChanPtr = &nextSendChan
			}
		case <-*nextSendChanPtr:
			nextSendChanPtr = &(m.randomTimer().C)

			go func() {
				// get list of random messages and recipients
				rng := m.rng.GetStream()
				defer rng.Close()
				msgs, err := m.newRandomMessages(rng)
				if err != nil {
					jww.ERROR.Printf("Failed to generate dummy messages: %+v", err)
					return
				}

				err = m.sendMessages(msgs, rng)
				if err != nil {
					jww.ERROR.Printf("Failed to send dummy messages: %+v", err)
				}
			}()

		}
	}
}

// stopSendThread is triggered when the stoppable is triggered. It prints a
// debug message, sets the thread status to stopped, and sets the status of the
// stoppable to stopped.
func (m *Manager) stopSendThread(stop *stoppable.Single) {
	jww.DEBUG.Print(
		"Stopping dummy traffic sending thread: stoppable triggered")
	atomic.StoreUint32(&m.status, stopped)
	stop.ToStopped()
}

// sendMessages generates and sends random messages.
func (m *Manager) sendMessages(msgs map[id.ID]format.Message, rng csprng.Source) error {
	var sent, i int64
	var wg sync.WaitGroup

	for recipient, msg := range msgs {
		wg.Add(1)

		go func(i int64, recipient id.ID, msg format.Message) {
			defer wg.Done()

			// Fill the preimage with random data to ensure it is not repeatable
			p := cmix.GetDefaultCMIXParams()
			//Send(recipient *id.ID, fingerprint format.Fingerprint,
			//	service message.Service, payload, mac []byte, cmixParams CMIXParams) (
			//	id.Round, ephemeral.Id, error)
			_, _, err := m.net.GetCmix().Send(&recipient, msg.GetKeyFP(),
				message.GetRandomService(rng), msg.GetContents(), msg.GetMac(), p)
			if err != nil {
				jww.WARN.Printf("Failed to send dummy message %d/%d via "+
					"Send: %+v", i, len(msgs), err)
			} else {
				atomic.AddInt64(&sent, 1)
			}
		}(i, recipient, msg)

		i++
	}

	wg.Wait()

	jww.INFO.Printf("Sent %d/%d dummy messages.", sent, len(msgs))

	return nil
}

// newRandomMessages returns a map of a random recipients and random messages of
// a randomly generated length in [1, Manager.maxNumMessages].
func (m *Manager) newRandomMessages(rng csprng.Source) (
	map[id.ID]format.Message, error) {
	numMessages, err := intRng(m.maxNumMessages+1, rng)
	if err != nil {
		return nil, errors.Errorf(numMsgsRngErr, err)
	}

	msgs := make(map[id.ID]format.Message, numMessages)

	for i := 0; i < numMessages; i++ {
		// Generate random recipient
		recipient, err := id.NewRandomID(rng, id.User)
		if err != nil {
			return nil, errors.Errorf(recipientRngErr, err)
		}

		msgs[*recipient], err = m.newRandomCmixMessage(rng)
		if err != nil {
			return nil, err
		}
	}

	return msgs, nil
}

// newRandomCmixMessage returns a new cMix message filled with a randomly
// generated payload, fingerprint, and MAC.
func (m *Manager) newRandomCmixMessage(rng csprng.Source) (format.Message, error) {
	// Create new empty cMix message
	clientStorage := *m.store
	cMixMsg := format.NewMessage(clientStorage.GetCmixGroup().GetP().ByteLen())

	// Generate random message
	randomMsg, err := newRandomPayload(cMixMsg.ContentsSize(), rng)
	if err != nil {
		return format.Message{}, errors.Errorf(payloadRngErr, err)
	}

	// Generate random fingerprint
	fingerprint, err := newRandomFingerprint(rng)
	if err != nil {
		return format.Message{}, errors.Errorf(fingerprintRngErr, err)
	}

	// Generate random MAC
	mac, err := newRandomMAC(rng)
	if err != nil {
		return format.Message{}, errors.Errorf(macRngErr, err)
	}

	// Set contents, fingerprint, and MAC, of the cMix message
	cMixMsg.SetContents(randomMsg)
	cMixMsg.SetKeyFP(fingerprint)
	cMixMsg.SetMac(mac)

	return cMixMsg, nil
}

// randomTimer generates a timer that will trigger after a random duration.
func (m *Manager) randomTimer() *time.Timer {
	rng := m.rng.GetStream()

	duration, err := durationRng(m.avgSendDelta, m.randomRange, rng)
	if err != nil {
		jww.FATAL.Panicf("Failed to generate random duration to wait to send "+
			"dummy messages: %+v", err)
	}

	return time.NewTimer(duration)
}
