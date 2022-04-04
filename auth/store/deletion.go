package store

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/xx_network/primitives/id"
)

// DeleteAllRequests clears the request map and all associated storage objects
// containing request data.
func (s *Store) DeleteAllRequests() error {
	s.mux.Lock()
	defer s.mux.Unlock()

	// Delete all requests
	s.deleteSentRequests()
	s.deleteReceiveRequests()

	if err := s.save(); err != nil {
		jww.FATAL.Panicf("Failed to store updated request map after "+
			"deleting all receivedByID: %+v", err)
	}

	return nil
}

// DeleteRequest deletes a request from Store given a partner ID.
// If the partner ID exists as a request,  then the request will be deleted
// and the state stored. If the partner does not exist, then an error will
// be returned.
func (s *Store) DeleteRequest(partner *id.ID) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	authId := makeAuthIdentity(partner, s.defaultID)

	// Check if this is a relationship in either map
	_, isReceivedRelationship := s.receivedByID[authId]
	_, isSentRelationship := s.sentByID[authId]

	// If it is not a relationship in either, return an error
	if !isSentRelationship && !isReceivedRelationship {
		return errors.New(fmt.Sprintf("No relationship exists with "+
			"identity %s", partner))
	}

	// Delete relationship. It should exist in at least one map,
	// for the other the delete operation is a no-op
	delete(s.receivedByID, authId)
	delete(s.sentByID, authId)

	// Save to storage
	if err := s.save(); err != nil {
		jww.FATAL.Panicf("Failed to store updated request map after "+
			"deleting partner request for partner %s: %+v", partner, err)
	}

	return nil
}

// DeleteSentRequests deletes all Sent receivedByID from Store.
func (s *Store) DeleteSentRequests() error {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.deleteSentRequests()

	if err := s.save(); err != nil {
		jww.FATAL.Panicf("Failed to store updated request map after "+
			"deleting all sent receivedByID: %+v", err)
	}

	return nil
}

// DeleteReceiveRequests deletes all Receive receivedByID from Store.
func (s *Store) DeleteReceiveRequests() error {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.deleteReceiveRequests()

	if err := s.save(); err != nil {
		jww.FATAL.Panicf("Failed to store updated request map after "+
			"deleting all partner receivedByID: %+v", err)
	}

	return nil
}

// deleteSentRequests is a helper function which deletes a Sent request from storage.
func (s *Store) deleteSentRequests() {
	for partnerId := range s.sentByID {
		delete(s.sentByID, partnerId)
	}
}

// deleteReceiveRequests is a helper function which deletes a Receive request from storage.
func (s *Store) deleteReceiveRequests() {
	for partnerId := range s.receivedByID {
		delete(s.receivedByID, partnerId)
	}
}
