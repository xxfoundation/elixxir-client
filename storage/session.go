////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Session object definition

package storage

import "gitlab.com/elixxir/ekv"

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
