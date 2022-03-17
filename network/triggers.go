///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package network

import (
	"encoding/base64"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/interfaces"
	fingerprint2 "gitlab.com/elixxir/crypto/fingerprint"
	"sync"
)

/* Trigger - predefined hash based tags appended to all cmix messages
which, though trial hashing, are used to determine if a message applies
to this client

Triggers are used for 2 purposes -  can be processed by the notifications system,
or can be used to implement custom non fingerprint processing of payloads.
I.E. key negotiation, broadcast negotiation

A tag is appended to the message of the format tag = H(H(messageContents),preimage)
and trial hashing is used to determine if a message adheres to a tag.
WARNING: If a preiamge is known by an adversary, they can determine which messages
are for the client.

Due to the extra overhead of trial hashing, triggers are processed after fingerprints.
If a fingerprint match occurs on the message, triggers will not be handled.

Triggers are ephemeral to the session. When starting a new client, all triggers must be
re-added before StartNetworkFollower is called.
*/

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

// Lookup will see if a trigger exists for the given preimage and message
// contents. It will do this by trial hashing the preimages in the map with
// the received message contents, until either a match to the
// received identity fingerprint is received or it has exhausted the map.
// If a match is found, this means the message received is for the client,
// and that one or multiple triggers exist to process this message.
// These triggers are returned to the caller along with the a true boolean.
// If the map has been exhausted with no matches found, it returns nil and false.
// todo: reorganize this interface. Lookup needs to be called
//  by handleMessage, which should not have access to the other
//  state modifying methods below. Possible options include:
//  - privatizing the state-changing methods
//  - leaking lookup on this layer and migrating the state modifiation methods
//    a layer down in a sepearate package
func (t *Triggers) Lookup(receivedIdentityFp,
	ecrMsgContents []byte) (triggers []*Trigger, forMe bool) {
	t.RLock()
	defer t.RUnlock()

	for preimage, triggerList := range t.triggers {
		preimageBytes, err := unmarshalPreimage(preimage)
		if err != nil {
			// fixme: panic here?, An error here would mean there's a bad
			//  key-value pair in the map (specifically the preimage-key is bad,
			//  as it should be base64 encoded).
		}
		// fixme, there probably needs to be a small refactor.
		//  Terminology and variable names are being used misused. For example:
		//  phrases like tag, preimage and identityFP are being used
		//  interchangeably in the code and it's getting unwieldy.
		if fingerprint2.CheckIdentityFP(receivedIdentityFp, ecrMsgContents, preimageBytes) {
			return triggerList, true
		}
	}

	return nil, false
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

	marshalledPreimage := marshalPreimage(trigger.Preimage)

	newTrigger := &Trigger{
		Trigger:                 trigger,
		MessageProcessorTrigger: response,
	}

	if existingTriggers, exists := t.triggers[marshalledPreimage]; exists {
		// fixme Should there be a check if this response exists already?
		t.triggers[marshalledPreimage] = append(existingTriggers, newTrigger)
		return nil
	}

	t.triggers[marshalledPreimage] = []*Trigger{newTrigger}

	return nil
}

// RemoveTrigger - If only a single response is associated with the preimage,
// the entire preimage is removed. If there is more than one response, only
// the given response is removed. If nil is passed in for response,
// all triggers for the preimage will be removed.
func (t *Triggers) RemoveTrigger(preimage []byte,
	response interfaces.MessageProcessorTrigger) error {
	t.Lock()
	defer t.Unlock()

	marshalledPreimage := marshalPreimage(preimage)

	triggers, exists := t.triggers[marshalledPreimage]
	if !exists {
		return errors.Errorf("No trigger with preimage %q found",
			marshalledPreimage)
	}

	if response == nil {
		delete(t.triggers, marshalledPreimage)
		return nil
	}

	for _, trigger := range triggers {
		if trigger.Equals(response) {
			delete(t.triggers, marshalPreimage(trigger.Preimage))
			return nil
		}
	}

	return errors.Errorf("No response (%q) exists with preimage %q",
		response.String(), marshalledPreimage)
}

// fixme: maybe make preimage a type or struct and place this in primitives?

// marshalPreimage is a helper which encodes the preimage byte data to
// a base64 encoded string.
func marshalPreimage(pi []byte) string {
	return base64.StdEncoding.EncodeToString(pi)
}

// unmarshalPreimage is a helper which decodes the preimage base64 string to
// bytes.
func unmarshalPreimage(data string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}

	return decoded, nil
}
