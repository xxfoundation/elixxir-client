////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package pickup

import (
	"strconv"

	"gitlab.com/elixxir/client/v4/cmix/gateway"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/cmix/pickup/store"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/client/v4/storage"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/id"
)

type Pickup interface {
	StartProcessors() stoppable.Stoppable
	GetMessagesFromRound(roundID id.Round, identity receptionID.EphemeralIdentity)
}

type pickup struct {
	params  Params
	sender  gateway.Sender
	session storage.Session

	comms MessageRetrievalComms

	historical rounds.Retriever

	rng *fastRNG.StreamGenerator

	instance RoundGetter

	lookupRoundMessages    chan roundLookup
	messageBundles         chan<- message.Bundle
	gatewayMessageRequests chan *pickupRequest

	unchecked *store.UncheckedRoundStore
}

func NewPickup(params Params, bundles chan<- message.Bundle,
	sender gateway.Sender, historical rounds.Retriever,
	comms MessageRetrievalComms,
	rng *fastRNG.StreamGenerator, instance RoundGetter,
	session storage.Session) Pickup {
	unchecked := store.NewOrLoadUncheckedStore(session.GetKV())

	m := &pickup{
		params:                 params,
		lookupRoundMessages:    make(chan roundLookup, params.LookupRoundsBufferLen),
		messageBundles:         bundles,
		sender:                 sender,
		historical:             historical,
		rng:                    rng,
		instance:               instance,
		unchecked:              unchecked,
		session:                session,
		comms:                  comms,
		gatewayMessageRequests: make(chan *pickupRequest, params.LookupRoundsBufferLen),
	}

	return m
}

func (m *pickup) StartProcessors() stoppable.Stoppable {

	multi := stoppable.NewMulti("Pickup")

	// Start the message retrieval worker pool
	if m.params.BatchMessageRetrieval {
		stopper := stoppable.NewSingle("Batch Message Retriever")
		go m.processBatchMessageRetrieval(m.comms, stopper)
		multi.Add(stopper)
	} else {
		for i := uint(0); i < m.params.NumMessageRetrievalWorkers; i++ {
			stopper := stoppable.NewSingle(
				"Message Retriever " + strconv.Itoa(int(i)))
			go m.processMessageRetrieval(m.comms, stopper)
			multi.Add(stopper)
		}
	}

	// Start the periodic unchecked round worker
	stopper := stoppable.NewSingle("UncheckRound")
	go m.processUncheckedRounds(
		m.params.UncheckRoundPeriod, backOffTable, stopper)
	multi.Add(stopper)

	return multi
}
