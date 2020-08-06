////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Session object definition

package storage

import (
	"gitlab.com/elixxir/ekv"
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
	vo := &VersionedObject{
		Timestamp: time.Now(),
		Data:      []byte(id),
	}
	return s.kv.Set("LastMessageID", vo)
}
