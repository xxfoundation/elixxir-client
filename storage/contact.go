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
	Email     string
}

// loadAllContacts populates the "contacts" variable for the session
func (s *Session) loadAllContacts() {
	s.contactsLck.Lock()
	defer s.contactsLck.Unlock()
	obj, err := s.Get("AllContacts")
	if err != nil {
		s.contacts = make(map[string]*Contact)
		return
	}
	err = json.Unmarshal(obj.Data, s.contacts)
	if err != nil {
		s.contacts = make(map[string]*Contact)
	}
}

func (s *Session) saveContacts() error {
	data, err := json.Marshal(s.contacts)
	if err != nil {
		return err
	}
	obj := VersionedObject{
		Version:   currentContactVersion,
		Timestamp: time.Now(),
		Data:      data,
	}
	return s.Set("AllContacts", &obj)
}

func (s *Session) updateContact(record *Contact) error {
	s.contactsLck.Lock()
	defer s.contactsLck.Unlock()
	s.contacts[record.Id.String()] = record
	return s.saveContacts()
}

// GetContactByEmail reads contact information from disk
func (s *Session) GetContactByEmail(email string) (*Contact, error) {
	key := MakeKeyWithPrefix("Contact", email)

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

// SetContactByEmail saves contact information to disk.
func (s *Session) SetContactByEmail(email string, record *Contact) error {
	err := s.updateContact(record)
	if err != nil {
		return err
	}

	key := MakeKeyWithPrefix("Contact", email)
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

func (s *Session) GetContactByID(ID *id.ID) *Contact {
	s.contactsLck.Lock()
	defer s.contactsLck.Unlock()
	c, ok := s.contacts[ID.String()]
	if !ok {
		return nil
	}
	return c
}

// DeleteContactByID removes the contact from disk
func (s *Session) DeleteContactByID(ID *id.ID) error {
	s.contactsLck.Lock()
	defer s.contactsLck.Unlock()
	record, ok := s.contacts[ID.String()]
	if !ok {
		return nil
	}
	delete(s.contacts, record.Id.String())
	err := s.saveContacts()
	if err != nil {
		return err
	}

	key := MakeKeyWithPrefix("Contact", record.Email)
	return s.Delete(key)
}
