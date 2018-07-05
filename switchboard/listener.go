////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package switchboard

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/privategrity/client/parse"
	"gitlab.com/privategrity/client/user"
	"strconv"
	"sync"
)

// This is an interface so you can receive callbacks through the Gomobile boundary
type Listener interface {
	Hear(msg *parse.Message, isHeardElsewhere bool)
}

type listenerRecord struct {
	l  Listener
	id string
}

type ListenerMap struct {
	// Hmmm...
	listeners map[user.ID]map[parse.Type][]*listenerRecord
	lastID    int
	// TODO right mutex type?
	mux sync.RWMutex
}

var Listeners = NewListenerMap()

func NewListenerMap() *ListenerMap {
	return &ListenerMap{
		listeners: make(map[user.ID]map[parse.Type][]*listenerRecord),
		lastID:    0,
	}
}

// Add a new listener to the map
// Returns ID of the new listener. Keep this around if you want to be able to
// delete the listener later.
//
// user: 0 for all,
// or any user ID to listen for messages from a particular user.
// messageType: 0 for all, or any message type to listen for messages of that
// type.
// newListener: something implementing the Listener callback interface.
// Don't pass nil to this.
//
// If a message matches multiple listeners, all of them will hear the message.
func (lm *ListenerMap) Register(user user.ID, messageType parse.Type,
	newListener Listener) string {
	lm.mux.Lock()
	defer lm.mux.Unlock()

	lm.lastID++
	if lm.listeners[user] == nil {
		lm.listeners[user] = make(map[parse.Type][]*listenerRecord)
	}

	if lm.listeners[user][messageType] == nil {
		lm.listeners[user][messageType] = make([]*listenerRecord, 0)
	}

	newListenerRecord := &listenerRecord{
		l:  newListener,
		id: strconv.Itoa(lm.lastID),
	}
	lm.listeners[user][messageType] = append(lm.listeners[user][messageType],
		newListenerRecord)

	return newListenerRecord.id
}

func (lm *ListenerMap) Unregister(listenerID string) {
	lm.mux.Lock()
	defer lm.mux.Unlock()

	// Iterate over all listeners in the map
	for u, perUser := range lm.listeners {
		for messageType, perType := range perUser {
			for i, listener := range perType {
				if listener.id == listenerID {
					// this matches. remove listener from the data structure
					lm.listeners[u][messageType] = append(perType[:i],
						perType[i+1:]...)
					// since the id is unique per listener,
					// we can terminate the loop early.
					return
				}
			}
		}
	}
}

func (lm *ListenerMap) matchListeners(userID user.ID,
	messageType parse.Type) []*listenerRecord {

	normals := make([]*listenerRecord, 0)

	for _, listener := range lm.listeners[userID][messageType] {
		normals = append(normals, listener)
	}
	return normals
}

// Broadcast a message to the appropriate listeners
func (lm *ListenerMap) Speak(msg *parse.Message) {
	jww.INFO.Printf("Speaking message: %v", []byte(msg.Body))
	lm.mux.RLock()
	defer lm.mux.RUnlock()

	var zeroUserID user.ID
	accumNormals := make([]*listenerRecord, 0)
	// match perfect matches
	normals := lm.matchListeners(msg.Sender, msg.Type)
	accumNormals = append(accumNormals, normals...)
	// match listeners that want just the user ID for all message types
	normals = lm.matchListeners(msg.Sender, 0)
	accumNormals = append(accumNormals, normals...)
	// match just the type
	normals = lm.matchListeners(zeroUserID, msg.Type)
	accumNormals = append(accumNormals, normals...)
	// match wildcard listeners that hear everything
	normals = lm.matchListeners(zeroUserID, 0)
	accumNormals = append(accumNormals, normals...)

	if len(accumNormals) > 0 {
		// notify all normal listeners
		for _, listener := range accumNormals {
			jww.INFO.Printf("Hearing on listener %v", listener.id)
			listener.l.Hear(msg, len(accumNormals) > 1)
		}
	} else {
		jww.ERROR.Println("Message didn't match any listeners in the map")
		// dump representation of the map
		for u, perUser := range lm.listeners {
			for messageType, perType := range perUser {
				for i, listener := range perType {
					jww.ERROR.Printf("Listener %v: %v, user %v, type %v, ",
						i, listener.id, u, messageType)
				}
			}
		}
	}
}
