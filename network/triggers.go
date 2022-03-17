///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package network

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/interfaces"
	"sync"
)

type Triggers struct {
	triggers map[string][]*Trigger
	sync.RWMutex
}

type Trigger struct {
	interfaces.Trigger
	interfaces.MessageProcessorTrigger
}

func NewTriggers() *Triggers {
	// todo: implement me
	return nil
}

func (t *Triggers) Lookup(identityFp,
	ecrMsgContents []byte) (*Trigger, bool) {
	t.RLock()
	defer t.RUnlock()

}

// Add - Adds a trigger which can call a message
// handing function or be used for notifications.
// Multiple triggers can be registered for the same preimage.
//   preimage - the preimage which is triggered on
//   type - a descriptive string of the trigger. Generally used in notifications
//   source - a byte buffer of related data. Generally used in notifications.
//     Example: Sender ID
func (t *Triggers) Add(trigger interfaces.Trigger,
	response interfaces.MessageProcessorTrigger) error {
	t.Lock()
	defer t.Unlock()

	preimage := trigger.Preimage.String()

	newTrigger := &Trigger{
		Trigger:                 trigger,
		MessageProcessorTrigger: response,
	}

	if existingTriggers, exists := t.triggers[preimage]; exists {
		t.triggers[preimage] = append(existingTriggers, newTrigger)
		return nil
	}

	t.triggers[preimage] = []*Trigger{newTrigger}

	return nil
}

// RemoveTrigger - If only a single response is associated with the preimage,
// the entire preimage is removed. If there is more than one response, only
// the given response is removed. If nil is passed in for response,
// all triggers for the preimage will be removed.
func (t *Triggers) RemoveTrigger(preimage interfaces.Preimage,
	response interfaces.MessageProcessorTrigger) error {
	t.Lock()
	defer t.Unlock()

	triggers, exists := t.triggers[preimage.String()]
	if !exists {
		return errors.Errorf("No triggers exist with preimage %q", preimage.String())
	}

	if response == nil {
		delete(t.triggers, preimage.String())
		return nil
	}

	for _, trigger := range triggers {
		if trigger.Equals(response) {
			delete(t.triggers, trigger.Preimage.String())
		}
	}

	return nil
}
