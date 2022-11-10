////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package dummy

import (
	"gitlab.com/elixxir/client/v5/cmix"
	"gitlab.com/xx_network/crypto/csprng"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v5/stoppable"
)

// Error messages for the Manager.sendThread and its helper functions.
const (
	numMsgsRngErr          = "failed to generate random number of messages to send: %+v"
	overrideAvgSendDelta   = 10 * time.Minute
	overrideRandomRange    = 8 * time.Minute
	overrideMaxNumMessages = 2

	numSendsToOverride = 20
)

// sendThread is a thread that sends the dummy messages at random intervals.
func (m *Manager) sendThread(stop *stoppable.Single) {
	jww.INFO.Print("Starting dummy traffic sending thread.")

	nextSendChan := make(<-chan time.Time)
	nextSendChanPtr := &(nextSendChan)

	for {

		if numSent := atomic.LoadUint64(m.totalSent); numSent > numSendsToOverride {
			m.avgSendDelta = overrideAvgSendDelta
			m.randomRange = overrideRandomRange
			m.maxNumMessages = overrideMaxNumMessages
		}

		select {
		case status := <-m.statusChan:
			if status {
				atomic.StoreUint32(&m.status, running)
				// Generate random duration
				rng := m.rng.GetStream()
				duration, err := randomDuration(m.avgSendDelta, m.randomRange, rng)
				if err != nil {
					rng.Close()
					jww.FATAL.Panicf("Failed to generate random sending interval: %+v", err)
				}
				rng.Close()

				// Create timer
				nextSendChanPtr = &(time.NewTimer(duration).C)

			} else {
				atomic.StoreUint32(&m.status, paused)
				nextSendChan = make(<-chan time.Time)
				nextSendChanPtr = &nextSendChan
			}
		case <-*nextSendChanPtr:
			// Generate random duration
			rng := m.rng.GetStream()
			duration, err := randomDuration(m.avgSendDelta, m.randomRange, rng)
			if err != nil {
				rng.Close()
				jww.FATAL.Panicf("Failed to generate random sending interval: %+v", err)
			}
			rng.Close()

			// Create timer
			nextSendChanPtr = &(time.NewTimer(duration).C)

			// Send messages
			go func() {
				err := m.sendMessages()
				if err != nil {
					jww.ERROR.Printf("Failed to send dummy messages: %+v", err)
				} else {
					atomic.AddUint64(m.totalSent, 1)
				}
			}()
		case <-stop.Quit():
			m.stopSendThread(stop)
			return

		}
	}
}

// sendMessages generates and sends random messages.
func (m *Manager) sendMessages() error {
	var sent int64
	var wg sync.WaitGroup

	// Randomly generate amount of messages to send
	rng := m.rng.GetStream()
	defer rng.Close()
	numMessages, err := randomInt(m.maxNumMessages+1, rng)
	if err != nil {
		return errors.Errorf(numMsgsRngErr, err)
	}

	for i := 0; i < numMessages; i++ {
		wg.Add(1)
		go func(localIndex, totalMessages int) {
			defer wg.Done()

			err = m.sendMessage(localIndex, totalMessages, rng)
			if err != nil {
				jww.ERROR.Printf("Failed to send message %d/%d: %+v",
					localIndex, numMessages, err)
			}
			// Add to counter of successful sends
			atomic.AddInt64(&sent, 1)
		}(i, numMessages)
	}

	wg.Wait()
	jww.INFO.Printf("Sent %d/%d dummy messages.", sent, numMessages)
	return nil
}

// sendMessage is a helper function which generates a sends a single random format.Message
// to a random recipient.
func (m *Manager) sendMessage(index, totalMessages int, rng csprng.Source) error {
	// Generate message data
	recipient, fp, service, payload, mac, err := m.newRandomCmixMessage(rng)
	if err != nil {
		return errors.Errorf("Failed to create random data: %+v", err)
	}

	// Send message
	p := cmix.GetDefaultCMIXParams()
	p.Probe = true
	_, _, err = m.net.Send(recipient, fp, service, payload, mac, p)
	if err != nil {
		return errors.Errorf("Failed to send message: %+v", err)
	}

	return nil
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
