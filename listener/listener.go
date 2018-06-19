package listener

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/client/parse"
	"strconv"
	"sync"
)

type Listener interface {
	Hear(msg *parse.Message)
}

type listenerRecord struct {
	l          Listener
	id         string
	isFallback bool
}

type ListenerMap struct {
	// Hmmm...
	listeners map[globals.UserID]map[int64][]*listenerRecord
	lastID    int
	// TODO right mutex type?
	mux sync.RWMutex
}

func NewListenerMap() *ListenerMap {
	return &ListenerMap{
		listeners: make(map[globals.UserID]map[int64][]*listenerRecord),
		lastID:    0,
		mux:       sync.RWMutex{},
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
// isFallback: if true, this listener will only hear messages that no
// non-fallback listeners have already heard.
//
// If a message matches multiple listeners, all of them will hear the message.
func (lm *ListenerMap) Listen(user globals.UserID, messageType int64,
	newListener Listener, isFallback bool) string {
	lm.mux.Lock()
	defer lm.mux.Unlock()

	lm.lastID++
	if lm.listeners[user] == nil {
		lm.listeners[user] = make(map[int64][]*listenerRecord)
	}

	if lm.listeners[user][messageType] == nil {
		lm.listeners[user][messageType] = make([]*listenerRecord, 0)
	}

	newListenerRecord := &listenerRecord{
		l:          newListener,
		id:         strconv.Itoa(lm.lastID),
		isFallback: isFallback,
	}
	lm.listeners[user][messageType] = append(lm.listeners[user][messageType],
		newListenerRecord)

	return newListenerRecord.id
}

func (lm *ListenerMap) StopListening(listenerID string) {
	lm.mux.Lock()
	defer lm.mux.Unlock()

	// Iterate over all listeners in the map
	for user, perUser := range lm.listeners {
		for messageType, perType := range perUser {
			for i, listener := range perType {
				if listener.id == listenerID {
					// this matches. remove listener from the data structure
					lm.listeners[user][messageType] = append(perType[:i],
						perType[i+1:]...)
					// since the id is unique per listener,
					// we can terminate the loop early.
					return
				}
			}
		}
	}
}

func (lm *ListenerMap) matchListeners(userID globals.UserID,
	messageType int64) (normals []*listenerRecord,
	fallbacks []*listenerRecord) {

	normals = make([]*listenerRecord, 0)
	fallbacks = make([]*listenerRecord, 0)

	for _, listener := range lm.listeners[userID][messageType] {
		if listener.isFallback {
			// matched a fallback listener
			fallbacks = append(fallbacks, listener)
		} else {
			// matched a normal listener
			normals = append(normals, listener)
		}
	}
	return normals, fallbacks
}

// Broadcast a message to the appropriate listeners
func (lm *ListenerMap) Speak(msg *parse.Message) {
	jww.INFO.Printf("Speaking message: %v", string(msg.Body))
	lm.mux.RLock()
	defer lm.mux.RUnlock()

	var zeroUserID globals.UserID
	accumNormals := make([]*listenerRecord, 0)
	accumFallbacks := make([]*listenerRecord, 0)
	// match perfect matches
	normals, fallbacks := lm.matchListeners(msg.Sender, msg.BodyType)
	accumNormals = append(accumNormals, normals...)
	accumFallbacks = append(accumFallbacks, fallbacks...)
	// match listeners that want just the user ID for all message types
	normals, fallbacks = lm.matchListeners(msg.Sender, 0)
	accumNormals = append(accumNormals, normals...)
	accumFallbacks = append(accumFallbacks, fallbacks...)
	// match just the type
	normals, fallbacks = lm.matchListeners(zeroUserID, msg.BodyType)
	accumNormals = append(accumNormals, normals...)
	accumFallbacks = append(accumFallbacks, fallbacks...)
	// match wildcard listeners that hear everything
	normals, fallbacks = lm.matchListeners(zeroUserID, 0)
	accumNormals = append(accumNormals, normals...)
	accumFallbacks = append(accumFallbacks, fallbacks...)

	if len(accumNormals) > 0 {
		// notify all normal listeners
		for _, listener := range accumNormals {
			jww.INFO.Printf("Hearing on listener %v", listener.id)
			listener.l.Hear(msg)
		}
	} else if len(accumFallbacks) > 0 {
		// notify all fallback listeners
		for _, listener := range accumFallbacks {
			jww.INFO.Printf("Hearing on listener %v", listener.id)
			listener.l.Hear(msg)
		}
	} else {
		jww.ERROR.Println("Message didn't match any listeners in the map")
		// dump representation of the map
		for user, perUser := range lm.listeners {
			for messageType, perType := range perUser {
				for i, listener := range perType {
					jww.ERROR.Printf("Listener %v: %v, user %v, type %v, is fallback: %v", i, listener.id, user, messageType, listener.isFallback)
				}
			}
		}
	}
}
