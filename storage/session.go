////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Session object definition

package storage

import (
	"fmt"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/ekv"
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
	fmt.Printf("key val: %v\n", s.kv)
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
