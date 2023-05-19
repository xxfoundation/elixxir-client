////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package conversation

import (
	"sync"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
)

const conversationKeyPrefix = "conversation"

type Store struct {
	loadedConversations map[id.ID]*Conversation
	kv                  versioned.KV
	mux                 sync.RWMutex
}

// NewStore returns a new conversation Store made off of the KV.
func NewStore(kv versioned.KV) *Store {
	kv, err := kv.Prefix(conversationKeyPrefix)
	if err != nil {
		jww.FATAL.Panicf("Failed to add prefix %s to KV", conversationKeyPrefix)
	}
	return &Store{
		loadedConversations: make(map[id.ID]*Conversation),
		kv:                  kv,
	}
}

// Get gets the conversation with the given partner ID from memory, if it is
// there. Otherwise, it loads it from disk.
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

// Delete deletes the conversation with the given partner ID from memory and
// storage. Panics if the object cannot be deleted from storage.
func (s *Store) Delete(partner *id.ID) {
	s.mux.Lock()
	defer s.mux.Unlock()

	// Get contact from memory
	c, exists := s.loadedConversations[*partner]
	if !exists {
		return
	}

	// Delete contact from storage
	err := c.delete()
	if err != nil {
		jww.FATAL.Panicf("Failed to remove conversation with ID %s from "+
			"storage: %+v", partner, err)
	}

	// Delete contact from memory
	delete(s.loadedConversations, *partner)
}
