////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package cmix

import (
	"gitlab.com/elixxir/client/v4/storage/utility"
	"time"

	"gitlab.com/elixxir/client/v4/cmix/rounds"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix/health"
	"gitlab.com/elixxir/client/v4/stoppable"
	ds "gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

const criticalRawMessagesKey = "RawCriticalMessages"

// roundEventRegistrar is an interface for the round events system to allow
// for easy testing.
type roundEventRegistrar interface {
	AddRoundEventChan(rid id.Round, eventChan chan ds.EventReturn,
		timeout time.Duration, validStates ...states.Round) *ds.EventCallback
}

// criticalSender is an anonymous function that takes the data critical knows
// for sending. It should call sendCmixHelper and use scope sharing in an
// anonymous function to include the structures from client that critical is
// not aware of.
type criticalSender func(msg format.Message, recipient *id.ID,
	params CMIXParams) (rounds.Round, ephemeral.Id, error)

// critical is a structure that allows the auto resending of messages that must
// be received.
type critical struct {
	*CmixMessageBuffer
	roundEvents roundEventRegistrar
	trigger     chan bool
	send        criticalSender
}

func newCritical(kv *utility.KV, hm health.Monitor,
	roundEvents roundEventRegistrar, send criticalSender) *critical {
	cm, err := NewOrLoadCmixMessageBuffer(kv, criticalRawMessagesKey)
	if err != nil {
		jww.FATAL.Panicf(
			"Failed to load the buffer for critical messages: %+v", err)
	}

	c := &critical{
		CmixMessageBuffer: cm,
		roundEvents:       roundEvents,
		trigger:           make(chan bool, 100),
		send:              send,
	}

	hm.AddHealthCallback(func(healthy bool) { c.trigger <- healthy })

	return c
}

func (c *critical) startProcessies() *stoppable.Single {
	stop := stoppable.NewSingle("criticalStopper")
	go c.runCriticalMessages(stop)
	return stop
}

func (c *critical) runCriticalMessages(stop *stoppable.Single) {
	for {
		select {
		case <-stop.Quit():
			stop.ToStopped()
			return
		case isHealthy := <-c.trigger:
			if isHealthy {
				c.evaluate(stop)
			}
		}
	}
}

func (c *critical) handle(
	msg format.Message, recipient *id.ID, rid id.Round, rtnErr error) bool {
	if rtnErr != nil {
		c.Failed(msg, recipient)
		return false
	} else {
		sendResults := make(chan ds.EventReturn, 1)

		c.roundEvents.AddRoundEventChan(
			rid, sendResults, 1*time.Minute, states.COMPLETED, states.FAILED)

		success, numTimeOut, _ := TrackResults(sendResults, 1)
		if !success {
			if numTimeOut > 0 {
				jww.ERROR.Printf("Critical raw message resend to %s "+
					"(msgDigest: %s) on round %d failed to transmit due to "+
					"timeout", recipient, msg.Digest(), rid)
			} else {
				jww.ERROR.Printf("Critical raw message resend to %s "+
					"(msgDigest: %s) on round %d failed to transmit due to "+
					"send failure", recipient, msg.Digest(), rid)
			}

			c.Failed(msg, recipient)
			return success
		}

		c.Succeeded(msg, recipient)
		return success
	}

}

// evaluate tries to send every message in the critical messages and the raw
// critical messages buffer in parallel.
func (c *critical) evaluate(stop *stoppable.Single) {
	for msg, recipient, params, has := c.Next(); has; msg, recipient, params, has = c.Next() {
		localRid := recipient.DeepCopy()
		go func(msg format.Message, recipient *id.ID, params CMIXParams) {
			params.Stop = stop
			params.Critical = false
			jww.INFO.Printf("Resending critical raw message to %s "+
				"(msgDigest: %s)", recipient, msg.Digest())

			// Send the message
			round, _, err := c.send(msg.Copy(), recipient, params)

			// Pass to the handler
			if c.handle(msg, recipient, round.ID, err) {
				jww.INFO.Printf("Successful resend of "+
					"critical raw message to "+
					"%s (msgDigest: %s) on round %d",
					recipient, msg.Digest(), round.ID)
			}
		}(msg, localRid, params)
	}
}
