////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package message

import (
	"crypto/hmac"
	"sync"

	jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/elixxir/crypto/sih"
	"gitlab.com/xx_network/primitives/id"
)

/* Service Identification Hash - predefined hash based tags appended to all cMix
messages which,though trial hashing, are used to determine if a message applies
to this client.

Services are used for 2 purposes - can be processed by the notification system,
or can be used to implement custom non fingerprint processing of payloads (i.e.
key negotiation and broadcast negotiation).

A tag is appended to the message of the format tag = H(H(messageContents),preimage)
and trial hashing is used to determine if a message adheres to a tag.
WARNING: If a preimage is known by an adversary, they can determine which
messages are for the client.

Due to the extra overhead of trial hashing, services are processed after
fingerprints. If a fingerprint match occurs on the message, triggers will not be
handled.

Services are address to the session. When starting a new client, all triggers
must be re-added before StartNetworkFollower is called.
*/

type ServicesManager struct {
	// Map reception ID to sih.Preimage to service
	tmap        map[id.ID]map[sih.Preimage]service
	trackers    []ServicesTracker
	numServices uint
	sync.Mutex
}

type service struct {
	Service
	Processor
	defaultList []Processor
}

func NewServices() *ServicesManager {
	return &ServicesManager{
		tmap:        make(map[id.ID]map[sih.Preimage]service),
		trackers:    make([]ServicesTracker, 0),
		numServices: 0,
	}
}

// Lookup will see if a service exists for the given preimage and message
// contents. It will do this by trial hashing the preimages in the map with the
// received message contents, until either a match to the received identity
// fingerprint is received or it has exhausted the map.
// If a match is found, this means the message received is for the client, and
// that one or multiple services exist to process this message.
// These services are returned to the caller along with a true boolean.
// If the map has been exhausted with no matches found, it returns nil and false.
func (sm *ServicesManager) get(clientID *id.ID, receivedSIH,
	ecrMsgContents []byte) ([]Processor, bool) {
	sm.Lock()
	defer sm.Unlock()
	cid := *clientID

	services, exists := sm.tmap[cid]
	if !exists {
		return nil, false
	}

	// NOTE: We exit on the first service match
	for _, s := range services {
		// Check if the SIH matches this service
		if s.ForMe(ecrMsgContents, receivedSIH) {
			if s.defaultList == nil && s.Tag != sih.Default {
				//skip if the processor is nil
				if s.Processor == nil {
					jww.ERROR.Printf("<nil> processor: %s",
						s.Tag)
					return []Processor{}, true
				}
				// Return this service directly if not
				// the default service
				return []Processor{s}, true

			} else if s.defaultList != nil {
				// If it is default and the default
				// list is not empty, then return the
				// default list
				return s.defaultList, true
			}

			// Return false if it is for me, but I have
			// nothing registered to respond to default
			// queries
			return []Processor{}, false
		}
		jww.TRACE.Printf("Evaluated service not for me (%s): %s",
			clientID, s)
	}

	return nil, false
}

// AddService adds a service which can call a message handing function or be
// used for notifications. In general a single service can only be registered
// for the same identifier/tag pair.
//
//	preimage - the preimage which is triggered on
//	type - a descriptive string of the service. Generally used in notifications
//	source - a byte buffer of related data. Mostly used in notifications.
//	  Example: Sender ID
//
// There can be multiple "default" services, they must use the "default" tag
// and the identifier must be the client reception ID.
// A service may have a nil response unless it is default.
func (sm *ServicesManager) AddService(clientID *id.ID, newService Service, response Processor) {
	sm.Lock()
	defer sm.Unlock()

	newEntry := service{
		Service:     newService,
		Processor:   response,
		defaultList: nil,
	}

	// Initialize the map for the ID if needed
	if _, exists := sm.tmap[*clientID]; !exists {
		sm.tmap[*clientID] = make(map[sih.Preimage]service)
	}

	// Handle default tag behavior
	if newService.Tag == sih.Default {
		if !hmac.Equal(newService.Identifier, clientID[:]) {
			jww.FATAL.Panicf("Cannot accept a malformed 'Default' " +
				"service, Identifier must match clientID")
		}
		oldDefault, exists := sm.tmap[*clientID][newService.preimage()]
		if exists {
			newEntry = oldDefault
			oldDefault.defaultList = append(oldDefault.defaultList, response)
		} else {
			newEntry.Metadata = clientID[:]
		}
	} else if _, exists := sm.tmap[*clientID][newService.preimage()]; exists {
		jww.FATAL.Panicf("Cannot add service %s, an identical "+
			"service already exists", newService.Tag)
	}

	jww.DEBUG.Printf("Adding service %s, clientID: %s", newService,
		clientID)

	// Add the service to the internal map
	sm.tmap[*clientID][newService.preimage()] = newEntry
	sm.numServices++

	// Signal that a new service was added
	sm.triggerServiceTracking()
}

// DeleteService - If only a single response is associated with the preimage,
// the entire preimage is removed. If there is more than one response, only the
// given response is removed. If nil is passed in for response, all triggers for
// the preimage will be removed.
func (sm *ServicesManager) DeleteService(clientID *id.ID, toDelete Service,
	processor Processor) {
	sm.Lock()
	defer sm.Unlock()
	cid := *clientID

	idTmap, exists := sm.tmap[cid]
	if !exists {
		return
	}

	services, exists := idTmap[toDelete.preimage()]
	if !exists {
		return
	}

	// Do unique handling if this is a default service and there is more than
	// one registered
	if services.defaultList != nil && len(services.defaultList) > 1 {
		for i, p := range services.defaultList {
			if p == processor {
				services.defaultList = append(
					services.defaultList[:i], services.defaultList[i+1:]...)
				idTmap[toDelete.preimage()] = services
				return
			}
		}
	}

	delete(idTmap, toDelete.preimage())
	sm.numServices--
	sm.triggerServiceTracking()
	return
}

// DeleteClientService deletes the mapping associated with an ID.
func (sm *ServicesManager) DeleteClientService(clientID *id.ID) {
	sm.Lock()
	defer sm.Unlock()

	delete(sm.tmap, *clientID)
}

func (s service) String() string {
	return s.Service.String()
}
