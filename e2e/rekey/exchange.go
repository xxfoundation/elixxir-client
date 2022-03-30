///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package rekey

import (
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/switchboard"
	"gitlab.com/xx_network/primitives/id"
)

const keyExchangeTriggerName = "KeyExchangeTrigger"
const keyExchangeConfirmName = "KeyExchangeConfirm"
const keyExchangeMulti = "KeyExchange"

func Start(switchboard *receive.Switchboard, sess *storage.Session, net interfaces.NetworkManager,
	params params.Rekey) (stoppable.Stoppable, error) {

	// register the rekey trigger thread
	triggerCh := make(chan message.Receive, 100)
	triggerID := switchboard.RegisterChannel(keyExchangeTriggerName,
		&id.ID{}, message.KeyExchangeTrigger, triggerCh)

	// create the trigger stoppable
	triggerStop := stoppable.NewSingle(keyExchangeTriggerName)

	cleanupTrigger := func() {
		switchboard.Unregister(triggerID)
	}

	// start the trigger thread
	go startTrigger(sess, net, triggerCh, triggerStop, params, cleanupTrigger)

	//register the rekey confirm thread
	confirmCh := make(chan message.Receive, 100)
	confirmID := switchboard.RegisterChannel(keyExchangeConfirmName,
		&id.ID{}, message.KeyExchangeConfirm, confirmCh)

	// register the confirm stoppable
	confirmStop := stoppable.NewSingle(keyExchangeConfirmName)
	cleanupConfirm := func() {
		switchboard.Unregister(confirmID)
	}

	// start the confirm thread
	go startConfirm(sess, confirmCh, confirmStop, cleanupConfirm)

	//bundle the stoppables and return
	exchangeStop := stoppable.NewMulti(keyExchangeMulti)
	exchangeStop.Add(triggerStop)
	exchangeStop.Add(confirmStop)
	return exchangeStop, nil
}
