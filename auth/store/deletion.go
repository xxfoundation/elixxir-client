package store

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/xx_network/primitives/id"
)

// DeleteAllRequests clears the request map and all associated storage objects
// containing request data.
func (s *Store) DeleteAllRequests() error {
	s.mux.Lock()
	defer s.mux.Unlock()

	for partnerId, req := range s.receivedByID {
		switch req.rt {
		case Sent:
			s.deleteSentRequest(req)
			delete(s.receivedByID, partnerId)
		case Receive:
			s.deleteReceiveRequest(req)
			delete(s.receivedByID, partnerId)
		}

	}

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
func (s *Store) DeleteRequest(partnerId *id.ID) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	req, ok := s.receivedByID[*partnerId]
	if !ok {
		return errors.Errorf("Request for %s does not exist", partnerId)
	}

	switch req.rt {
	case Sent:
		s.deleteSentRequest(req)
	case Receive:
		s.deleteReceiveRequest(req)
	}

	delete(s.receivedByID, *partnerId)

	if err := s.save(); err != nil {
		jww.FATAL.Panicf("Failed to store updated request map after "+
			"deleting partner request for partner %s: %+v", partnerId, err)
	}

	return nil
}

// DeleteSentRequests deletes all Sent receivedByID from Store.
func (s *Store) DeleteSentRequests() error {
	s.mux.Lock()
	defer s.mux.Unlock()

	for partnerId, req := range s.receivedByID {
		switch req.rt {
		case Sent:
			s.deleteSentRequest(req)
			delete(s.receivedByID, partnerId)
		case Receive:
			continue
		}
	}

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

	for partnerId, req := range s.receivedByID {
		switch req.rt {
		case Sent:
			continue
		case Receive:
			s.deleteReceiveRequest(req)
			delete(s.receivedByID, partnerId)
		}
	}

	if err := s.save(); err != nil {
		jww.FATAL.Panicf("Failed to store updated request map after "+
			"deleting all partner receivedByID: %+v", err)
	}

	return nil
}

// deleteSentRequest is a helper function which deletes a Sent request from storage.
func (s *Store) deleteSentRequest(r *ReceivedRequest) {
	delete(s.sentByFingerprints, r.sent.fingerprint)
	if err := r.sent.delete(); err != nil {
		jww.FATAL.Panicf("Failed to delete sent request: %+v", err)
	}
}

// deleteReceiveRequest is a helper function which deletes a Receive request from storage.
func (s *Store) deleteReceiveRequest(r *ReceivedRequest) {
	if err := util.DeleteContact(s.kv, r.partner.ID); err != nil {
		jww.FATAL.Panicf("Failed to delete recieved request "+
			"contact: %+v", err)
	}
}
