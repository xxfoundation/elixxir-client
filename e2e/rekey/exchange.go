////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package rekey

import (
	"gitlab.com/elixxir/client/v4/catalog"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/e2e/ratchet"
	"gitlab.com/elixxir/client/v4/e2e/receive"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/xx_network/primitives/id"
)

type E2eSender func(mt catalog.MessageType, recipient *id.ID, payload []byte,
	cmixParams cmix.CMIXParams) (e2e.SendReport, error)

func Start(switchboard *receive.Switchboard, ratchet *ratchet.Ratchet,
	sender E2eSender, net cmix.Client, grp *cyclic.Group, params Params) (stoppable.Stoppable, error) {

	// register the rekey trigger thread
	triggerCh := make(chan receive.Message, 100)
	triggerID := switchboard.RegisterChannel(params.TriggerName,
		&id.ID{}, params.Trigger, triggerCh)

	// create the trigger stoppable
	triggerStop := stoppable.NewSingle(params.TriggerName)

	cleanupTrigger := func() {
		switchboard.Unregister(triggerID)
	}

	// start the trigger thread
	go startTrigger(ratchet, sender, net, grp, triggerCh, triggerStop, params,
		cleanupTrigger)

	//register the rekey confirm thread
	confirmCh := make(chan receive.Message, 100)
	confirmID := switchboard.RegisterChannel(params.ConfirmName,
		&id.ID{}, params.Confirm, confirmCh)

	// register the confirm stoppable
	confirmStop := stoppable.NewSingle(params.ConfirmName)
	cleanupConfirm := func() {
		switchboard.Unregister(confirmID)
	}

	// start the confirm thread
	go startConfirm(ratchet, confirmCh, confirmStop, cleanupConfirm)

	//bundle the stoppables and return
	exchangeStop := stoppable.NewMulti(params.StoppableName)
	exchangeStop.Add(triggerStop)
	exchangeStop.Add(confirmStop)
	return exchangeStop, nil
}
