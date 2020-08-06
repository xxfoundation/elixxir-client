////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Session object definition

package storage

import (
	"encoding/json"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/xx_network/comms/connect"
	"testing"
	"time"
)

// Session object, backed by encrypted filestore
type Session struct {
	kv *VersionedKV
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
	v, err := s.kv.Get("LastMessageID")
	if v == nil || err != nil {
		return "", nil
	}
	return string(v.Data), nil
}

// Set the LastMessageID in the Session
func (s *Session) SetLastMessageId(id string) error {
	ts, err := time.Now().MarshalText()
	if err != nil {
		return err
	}
	vo := &VersionedObject{
		Timestamp: ts,
		Data:      []byte(id),
	}
	return s.kv.Set("LastMessageID", vo)
}

// Helper for obtaining NodeKeys map
func (s *Session) getNodeKeys() (map[id.ID]user.NodeKeys, error) {
	v, err := s.kv.Get("NodeKeys")
	if err != nil {
		return nil, err
	}

	var nodeKeys map[id.ID]user.NodeKeys
	err = json.Unmarshal(v.Data, &nodeKeys)
	return nodeKeys, err
}

// Obtain NodeKeys from the Session
func (s *Session) GetNodeKeys(topology *connect.Circuit) ([]user.NodeKeys, error) {
	nodeKeys, err := s.getNodeKeys()
	if err != nil {
		return nil, err
	}

	keys := make([]user.NodeKeys, topology.Len())
	for i := 0; i < topology.Len(); i++ {
		keys[i] = nodeKeys[*topology.GetNodeAtIndex(i)]
	}

	return keys, nil
}

// Set NodeKeys in the Session
func (s *Session) PushNodeKey(id *id.ID, key user.NodeKeys) error {
	// Obtain NodeKeys map
	nodeKeys, err := s.getNodeKeys()
	if err != nil {
		return err
	}

	// Set new value inside of map
	nodeKeys[*id] = key

	// Marshal the map
	pushValue, err := json.Marshal(nodeKeys)
	if err != nil {
		return err
	}

	// Insert the map back into the Session
	ts, err := time.Now().MarshalText()
	if err != nil {
		return err
	}
	vo := &VersionedObject{
		Timestamp: ts,
		Data:      pushValue,
	}
	return s.kv.Set("NodeKeys", vo)
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
	return &Session{NewVersionedKV(store)}

}
