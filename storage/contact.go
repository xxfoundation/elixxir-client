////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	"encoding/json"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

const currentContactVersion = 0

// Contact holds the public key and ID of a given contact.
type Contact struct {
	Id        *id.ID
	PublicKey []byte
}

// GetContact reads contact information from disk
func (s *Session) GetContact(name string) (*Contact, error) {
	key := MakeKeyWithPrefix("Contact", name)

	obj, err := s.Get(key)
	if err != nil {
		return nil, err
	}
	// Correctly implemented upgrade should always change the version number to what's current
	if obj.Version != currentContactVersion {
		globals.Log.WARN.Printf("Session.GetContact: got unexpected "+
			"version %v, expected version %v", obj.Version,
			currentContactVersion)
	}

	// deserialize
	var contact Contact
	err = json.Unmarshal(obj.Data, &contact)
	return &contact, err
}

// SetContact saves contact information to disk.
func (s *Session) SetContact(name string, record *Contact) error {
	key := MakeKeyWithPrefix("Contact", name)
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	obj := VersionedObject{
		Version:   currentContactVersion,
		Timestamp: time.Now(),
		Data:      data,
	}
	return s.Set(key, &obj)
}
