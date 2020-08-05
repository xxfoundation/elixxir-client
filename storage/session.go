////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	"gitlab.com/elixxir/ekv"
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
	return string(v.Data), err
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
