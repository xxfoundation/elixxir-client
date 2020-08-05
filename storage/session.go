////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	"encoding/json"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/xx_network/comms/connect"
	"time"
)

type Session struct {
	kv *VersionedKV
}

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

func (s *Session) Get(key string) (*VersionedObject, error) {
	return s.kv.Get(key)
}

func (s *Session) Set(key string, object *VersionedObject) error {
	return s.kv.Set(key, object)
}

// Obtain the LastMessageID from the Session
func (s *Session) GetLastMessageId() (string, error) {
	v, err := s.kv.Get("LastMessageID")
	if err != nil {
		return "", err
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
