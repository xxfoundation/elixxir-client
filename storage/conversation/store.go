package conversation

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"sync"
)

const conversationKeyPrefix = "conversation"

type Store struct {
	loadedConversations map[id.ID]*Conversation
	kv                  *versioned.KV
	mux                 sync.RWMutex
}

//Returns a new conversation store made off of the KV
func NewStore(kv *versioned.KV) *Store {
	kv = kv.Prefix(conversationKeyPrefix).Prefix("Partner")
	return &Store{
		loadedConversations: make(map[id.ID]*Conversation),
		kv:                  kv,
	}
}

// Gets the conversation with the partner from ram if it is there, otherwise
// loads it from disk
func (s *Store) Get(partner *id.ID) *Conversation {
	s.mux.RLock()
	c, ok := s.loadedConversations[*partner]
	s.mux.RUnlock()
	if !ok {
		s.mux.Lock()
		c, ok = s.loadedConversations[*partner]
		if !ok {
			c = LoadOrMakeConversation(s.kv, partner)
			s.loadedConversations[*partner] = c
		}
		s.mux.Unlock()
	}
	return c
}
