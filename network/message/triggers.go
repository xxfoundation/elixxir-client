///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package message

import (
	"sync"

	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/crypto/fingerprint"
	"gitlab.com/xx_network/primitives/id"
)

/* Trigger - predefined hash based tags appended to all cMix messages which,
though trial hashing, are used to determine if a message applies to this client.

Triggers are used for 2 purposes - can be processed by the notification system,
or can be used to implement custom non fingerprint processing of payloads (i.e.
key negotiation and broadcast negotiation).

A tag is appended to the message of the format tag = H(H(messageContents),preimage)
and trial hashing is used to determine if a message adheres to a tag.
WARNING: If a preimage is known by an adversary, they can determine which
messages are for the client.

Due to the extra overhead of trial hashing, triggers are processed after
fingerprints. If a fingerprint match occurs on the message, triggers will not be
handled.

Triggers are ephemeral to the session. When starting a new client, all triggers
must be re-added before StartNetworkFollower is called.
*/

type TriggersManager struct {
	tmap map[id.ID]map[interfaces.Preimage][]trigger
	sync.Mutex
}

type trigger struct {
	interfaces.Trigger
	interfaces.MessageProcessor
}

func NewTriggers() *TriggersManager {
	// todo: implement me
	return &TriggersManager{
		tmap: make(map[id.ID]map[interfaces.Preimage][]trigger, 0),
	}
}

// Lookup will see if a trigger exists for the given preimage and message
// contents. It will do this by trial hashing the preimages in the map with the
// received message contents, until either a match to the received identity
// fingerprint is received or it has exhausted the map.
// If a match is found, this means the message received is for the client, and
// that one or multiple triggers exist to process this message.
// These triggers are returned to the caller along with the a true boolean.
// If the map has been exhausted with no matches found, it returns nil and false.
func (t *TriggersManager) get(clientID *id.ID, receivedIdentityFp,
	ecrMsgContents []byte) ([]trigger,
	bool) {
	t.Lock()
	defer t.Unlock()
	cid := *clientID

	triggers, exists := t.tmap[cid]
	if !exists {
		return nil, false
	}

	for pi, triggerList := range triggers {
		if fingerprint.CheckIdentityFP(receivedIdentityFp,
			ecrMsgContents, pi[:]) {
			return triggerList, true
		}
	}

	return nil, false
}

// AddTrigger adds a trigger which can call a message handing function or be
// used for notifications. Multiple triggers can be registered for the same
// preimage.
//   preimage - the preimage which is triggered on
//   type - a descriptive string of the trigger. Generally used in notifications
//   source - a byte buffer of related data. Generally used in notifications.
//     Example: Sender ID
func (t *TriggersManager) AddTrigger(clientID *id.ID, newTrigger interfaces.Trigger,
	response interfaces.MessageProcessor) {
	t.Lock()
	defer t.Unlock()

	newEntry := trigger{
		Trigger:          newTrigger,
		MessageProcessor: response,
	}

	cid := *clientID
	if _, exists := t.tmap[cid]; !exists {
		t.tmap[cid] = make(map[interfaces.Preimage][]trigger)
	}

	pi := newTrigger.Preimage
	if existingTriggers, exists := t.tmap[cid][pi]; exists {
		t.tmap[cid][pi] = append(existingTriggers, newEntry)
	}

	t.tmap[cid][pi] = []trigger{newEntry}

}

// DeleteTriggers - If only a single response is associated with the preimage,
// the entire preimage is removed. If there is more than one response, only the
// given response is removed. If nil is passed in for response, all triggers for
// the preimage will be removed.
func (t *TriggersManager) DeleteTriggers(clientID *id.ID, preimage interfaces.Preimage,
	response interfaces.MessageProcessor) error {
	t.Lock()
	defer t.Unlock()

	if response == nil {
		return errors.Errorf("response cannot be nil when deleting")
	}

	cid := *clientID

	idTmap, exists := t.tmap[cid]
	if !exists {
		return nil
	}

	triggers, exists := idTmap[preimage]
	if !exists {
		return nil
	}

	if len(triggers) == 1 && triggers[0].MessageProcessor == response {
		if len(idTmap) == 1 {
			delete(t.tmap, cid)
		} else {
			delete(t.tmap[cid], preimage)
		}
	}

	for idx, cur := range triggers {
		if cur.MessageProcessor == response {
			t.tmap[cid][preimage] = append(triggers[:idx],
				triggers[idx+1:]...)
			return nil
		}
	}

	return nil
}

// DeleteClientTriggers deletes the mapping associated with an ID.
func (t *TriggersManager) DeleteClientTriggers(clientID *id.ID) {
	t.Lock()
	defer t.Unlock()

	delete(t.tmap, *clientID)
}
