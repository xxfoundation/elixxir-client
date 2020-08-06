package storage

import (
	"encoding/json"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

const currentContactVersion = 0

func (s *Session) GetContact(name string) (*Contact, error) {
	// Make key
	// If upgrading version, may need to add logic to update version number in key prefix
	key := MakeKeyPrefix("Contact", currentContactVersion) + name

	obj, err := s.Get(key)
	if err != nil {
		return nil, err
	}
	// Correctly implemented upgrade should always change the version number to what's current
	if obj.Version != currentContactVersion {
		globals.Log.WARN.Printf("Session.GetContact: got unexpected version %v, expected version %v", obj.Version, currentContactVersion)
	}

	// deserialize
	var contact Contact
	err = json.Unmarshal(obj.Data, &contact)
	return &contact, err
}

func (s *Session) SetContact(name string, record *Contact) error {
	now, err := time.Now().MarshalText()
	if err != nil {
		return err
	}

	key := MakeKeyPrefix("Contact", currentContactVersion) + name
	var data []byte
	data, err = json.Marshal(record)
	if err != nil {
		return err
	}
	obj := VersionedObject{
		Version:   currentContactVersion,
		Timestamp: now,
		Data:      data,
	}
	return s.Set(key, &obj)
}

type Contact struct {
	Id        *id.ID
	PublicKey []byte
}
