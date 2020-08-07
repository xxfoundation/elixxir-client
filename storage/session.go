////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Session object definition

package storage

import (
	"bytes"
	"encoding/gob"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"testing"
	"time"
)

// Session object, backed by encrypted filestore
type Session struct {
	kv  *VersionedKV
	mux sync.Mutex
}

// Initialize a new Session object
func Init(baseDir, password string) (*Session, error) {
	fs, err := ekv.NewFilestore(baseDir, password)
	var s *Session
	if err == nil {
		s = &Session{
			kv: NewVersionedKV(fs),
		}
	}

	return s, err
}

// Get an object from the session
func (s *Session) Get(key string) (*VersionedObject, error) {
	return s.kv.Get(key)
}

// Set a value in the session
func (s *Session) Set(key string, object *VersionedObject) error {
	return s.kv.Set(key, object)
}

// Obtain the LastMessageID from the Session
func (s *Session) GetLastMessageId() (string, error) {
	v, err := s.Get("LastMessageID")
	if v == nil || err != nil {
		return "", nil
	}
	return string(v.Data), nil
}

// Set the LastMessageID in the Session
func (s *Session) SetLastMessageId(id string) error {
	vo := &VersionedObject{
		Timestamp: time.Now(),
		Data:      []byte(id),
	}
	return s.Set("LastMessageID", vo)
}

// GetNodeKeys returns all keys
func (s *Session) GetNodeKeys() (map[string]user.NodeKeys, error) {
	key := "NodeKeys"
	var nodeKeys map[string]user.NodeKeys

	// Attempt to locate the keys map
	v, err := s.Get(key)

	// If the map doesn't exist, initialize it
	if err != nil {
		// Encode the new map
		nodeKeys = make(map[string]user.NodeKeys)
		var nodeKeysBuffer bytes.Buffer
		enc := gob.NewEncoder(&nodeKeysBuffer)
		err = enc.Encode(nodeKeys)
		if err != nil {
			return nil, err
		}

		// Store the new map
		vo := &VersionedObject{
			Timestamp: time.Now(),
			Data:      nodeKeysBuffer.Bytes(),
		}
		err = s.Set(key, vo)
		if err != nil {
			return nil, err
		}

		// Return newly-initialized map
		return nodeKeys, nil
	}

	// If the map exists, decode and return it
	var nodeKeyBuffer bytes.Buffer
	nodeKeyBuffer.Write(v.Data)
	dec := gob.NewDecoder(&nodeKeyBuffer)
	err = dec.Decode(&nodeKeys)

	return nodeKeys, err
}

// GetNodeKeysFromCircuit obtains NodeKeys for a given circuit
func (s *Session) GetNodeKeysFromCircuit(topology *connect.Circuit) (
	[]user.NodeKeys, error) {
	nodeKeys, err := s.GetNodeKeys()
	if err != nil {
		return nil, err
	}

	// Build a list of NodeKeys from the map
	keys := make([]user.NodeKeys, topology.Len())
	for i := 0; i < topology.Len(); i++ {
		nid := topology.GetNodeAtIndex(i)
		keys[i] = nodeKeys[nid.String()]
		globals.Log.INFO.Printf("Read NodeKey: %s: %v", nid, keys[i])
	}

	return keys, nil
}

// Set NodeKeys in the Session
func (s *Session) PushNodeKey(id *id.ID, key user.NodeKeys) error {
	// Thread-safety
	s.mux.Lock()
	defer s.mux.Unlock()

	// Obtain NodeKeys map
	nodeKeys, err := s.GetNodeKeys()
	if err != nil {
		return err
	}

	// Set new value inside of map
	nodeKeys[id.String()] = key

	globals.Log.INFO.Printf("Adding NodeKey: %s: %v", id.String(), key)

	// Encode the map
	var nodeKeysBuffer bytes.Buffer
	enc := gob.NewEncoder(&nodeKeysBuffer)
	err = enc.Encode(nodeKeys)

	// Insert the map back into the Session
	vo := &VersionedObject{
		Timestamp: time.Now(),
		Data:      nodeKeysBuffer.Bytes(),
	}
	return s.Set("NodeKeys", vo)
}

// Initializes a Session object wrapped around a MemStore object.
// FOR TESTING ONLY
func InitTestingSession(i interface{}) *Session {
	switch i.(type) {
	case *testing.T:
		break
	case *testing.M:
		break
	case *testing.B:
		break
	default:
		globals.Log.FATAL.Panicf("InitTestingSession is restricted to testing only. Got %T", i)
	}

	store := make(ekv.Memstore)
	return &Session{kv: NewVersionedKV(store)}

}
