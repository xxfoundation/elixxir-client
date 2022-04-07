package e2e

import (
	"time"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage/versioned"
	ds "gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
)

const e2eCriticalMessagesKey = "E2ECriticalMessages"

// roundEventRegistrar is an interface for the round events system to allow
// for easy testing.
type roundEventRegistrar interface {
	AddRoundEventChan(rid id.Round, eventChan chan ds.EventReturn,
		timeout time.Duration, validStates ...states.Round) *ds.EventCallback
}

// criticalSender is an anonymous function that takes the data critical knows
// for sending. It should call sendCmixHelper and use scope sharing in an
// anonymous function to include the structures from manager that critical is
// not aware of.
type criticalSender func(mt catalog.MessageType, recipient *id.ID,
	payload []byte, params Params) ([]id.Round, e2e.MessageID, time.Time, error)

// critical is a structure that allows the auto resending of messages that must
// be received.
type critical struct {
	*E2eMessageBuffer
	roundEvents roundEventRegistrar
	trigger     chan bool
	send        criticalSender
}

func newCritical(kv *versioned.KV, hm func(f func(bool)) uint64,
	roundEvents roundEventRegistrar, send criticalSender) *critical {
	cm, err := NewOrLoadE2eMessageBuffer(kv, e2eCriticalMessagesKey)
	if err != nil {
		jww.FATAL.Panicf("cannot load the critical messages buffer: "+
			"%+v", err)
	}

	c := &critical{
		E2eMessageBuffer: cm,
		roundEvents:      roundEvents,
		trigger:          make(chan bool, 100),
		send:             send,
	}

	hm(func(healthy bool) { c.trigger <- healthy })

	return c
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

func (c *critical) handle(mt catalog.MessageType, recipient *id.ID,
	payload []byte, rids []id.Round, rtnErr error) {
	if rtnErr != nil {
		c.Failed(mt, recipient, payload)
	} else {
		sendResults := make(chan ds.EventReturn, 1)

		for _, rid := range rids {
			c.roundEvents.AddRoundEventChan(
				rid, sendResults, 1*time.Minute,
				states.COMPLETED,
				states.FAILED)
		}
		success, numTimeOut, _ := cmix.TrackResults(sendResults,
			len(rids))
		if !success {
			if numTimeOut > 0 {
				jww.ERROR.Printf("Critical e2e message resend "+
					"to %s (msgDigest: %s) on round %d "+
					"failed to transmit due to timeout",
					recipient,
					format.DigestContents(payload),
					rids)
			} else {
				jww.ERROR.Printf("Critical raw message resend "+
					"to %s (msgDigest: %s) on round %d "+
					"failed to transmit "+
					"due to send failure",
					recipient,
					format.DigestContents(payload),
					rids)
			}

			c.Failed(mt, recipient, payload)
			return
		}

		jww.INFO.Printf("Successful resend of critical raw message to "+
			"%s (msgDigest: %s) on round %d", recipient,
			format.DigestContents(payload), rids)

		c.Succeeded(mt, recipient, payload)
	}

}

// evaluate tries to send every message in the critical messages and the raw
// critical messages buffer in parallel.
func (c *critical) evaluate(stop *stoppable.Single) {
	mt, recipient, payload, params, has := c.Next()
	for ; has; mt, recipient, payload, params, has = c.Next() {
		go func(mt catalog.MessageType, recipient *id.ID,
			payload []byte, params Params) {

			params.CMIX.Stop = stop
			jww.INFO.Printf("Resending critical raw message to %s "+
				"(msgDigest: %s)", recipient,
				format.DigestContents(payload))

			// Send the message
			round, _, _, err := c.send(mt, recipient, payload,
				params)

			// Pass to the handler
			c.handle(mt, recipient, payload, round, err)
		}(mt, recipient, payload, params)
	}

}
